package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/p2b/p2b/internal/domain"
	"github.com/p2b/p2b/internal/extraction"
)

func (s *Store) Claim(ctx context.Context) (Job, error) {
	var job Job
	err := s.database.QueryRow(ctx, `
		WITH candidate AS (
			SELECT id FROM jobs
		WHERE type IN ('PASSPORT_BUILD', 'PASSPORT_REFRESH') AND ((status = 'QUEUED' AND available_at <= now())
			   OR (status = 'LEASED' AND lease_expires_at < now()))
			ORDER BY available_at, created_at
			FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE jobs SET status = 'LEASED', attempts = attempts + 1,
			lease_expires_at = now() + interval '5 minutes', updated_at = now()
		WHERE id = (SELECT id FROM candidate)
		RETURNING id, workspace_id, type, status, progress, created_at, attempts, max_attempts,
			COALESCE(last_error, ''), payload`,
	).Scan(&job.ID, &job.WorkspaceID, &job.Type, &job.Status, &job.Progress, &job.CreatedAt, &job.Attempts, &job.MaxAttempts, &job.LastError, &job.Payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return job, err
}

func (s *Store) Sources(ctx context.Context, workspaceID string, sourceIDs []string) ([]SourceRecord, error) {
	parsed := make([]uuid.UUID, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		value, err := uuid.Parse(sourceID)
		if err != nil {
			return nil, errors.New("job contains invalid source id")
		}
		parsed = append(parsed, value)
	}
	rows, err := s.database.Query(ctx, `
		SELECT id, workspace_id, filename, content_type, size_bytes, object_key, status
		FROM company_sources WHERE workspace_id = $1::uuid AND id = ANY($2::uuid[])
		ORDER BY created_at`, workspaceID, parsed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]SourceRecord, 0, len(parsed))
	for rows.Next() {
		var source SourceRecord
		if err = rows.Scan(&source.ID, &source.WorkspaceID, &source.Filename, &source.ContentType, &source.SizeBytes, &source.ObjectKey, &source.Status); err != nil {
			return nil, err
		}
		result = append(result, source)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(result) != len(parsed) {
		return nil, errors.New("one or more job sources no longer exist")
	}
	return result, nil
}

func (s *Store) StartSource(ctx context.Context, sourceID string) error {
	_, err := s.database.Exec(ctx, `UPDATE company_sources SET status = 'EXTRACTING', extraction_error = NULL, updated_at = now() WHERE id = $1::uuid`, sourceID)
	return err
}

func (s *Store) CompleteSource(ctx context.Context, sourceID, contentHash, markdown, model string) error {
	_, err := s.database.Exec(ctx, `
		UPDATE company_sources SET status = 'EXTRACTED', content_hash = $2, extracted_markdown = $3,
			extraction_model = $4, extraction_error = NULL, updated_at = now()
		WHERE id = $1::uuid`, sourceID, contentHash, markdown, model)
	return err
}

func (s *Store) FailSource(ctx context.Context, sourceID, message string) error {
	_, err := s.database.Exec(ctx, `UPDATE company_sources SET status = 'FAILED', extraction_error = $2, updated_at = now() WHERE id = $1::uuid`, sourceID, truncateError(message))
	return err
}

func (s *Store) SaveCandidates(ctx context.Context, workspaceID, sourceID, sourceName, contentHash string, candidates []extraction.Candidate) error {
	transaction, err := s.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Rollback(ctx)
	var passportID uuid.UUID
	var encodedFields []byte
	if err = transaction.QueryRow(ctx, `
		SELECT p.id, pv.fields
		FROM passports p JOIN passport_versions pv ON pv.passport_id = p.id AND pv.version = p.current_version
		WHERE p.workspace_id = $1::uuid`, workspaceID).Scan(&passportID, &encodedFields); err != nil {
		return err
	}
	confirmedFields := map[string]domain.PassportField{}
	if len(encodedFields) > 0 {
		if err = json.Unmarshal(encodedFields, &confirmedFields); err != nil {
			return err
		}
	}
	for _, candidate := range candidates {
		if confirmed, ok := confirmedFields[candidate.FieldKey]; ok && confirmed.Status == domain.FieldConfirmed {
			confirmedValue, marshalErr := json.Marshal(confirmed.Value)
			if marshalErr != nil {
				return marshalErr
			}
			candidateValue, marshalErr := json.Marshal(candidate.Value)
			if marshalErr != nil {
				return marshalErr
			}
			if string(confirmedValue) == string(candidateValue) {
				// Already confirmed with the same value — nothing new to review.
				continue
			}
		}
		value, marshalErr := json.Marshal(candidate.Value)
		if marshalErr != nil {
			return marshalErr
		}
		evidence, marshalErr := json.Marshal(domain.Evidence{SourceID: sourceID, SourceName: sourceName, Quote: candidate.Quote, ContentHash: contentHash, ObservedAt: time.Now().UTC()})
		if marshalErr != nil {
			return marshalErr
		}
		status := "NEEDS_REVIEW"
		if confirmed, ok := confirmedFields[candidate.FieldKey]; ok && confirmed.Status == domain.FieldConfirmed {
			status = "CONFLICTED"
		}
		if _, err = transaction.Exec(ctx, `
			INSERT INTO field_candidates (workspace_id, passport_id, source_id, field_key, value, data_type, status, confidence, evidence)
			VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6, $7, $8, $9)
			ON CONFLICT DO NOTHING`, workspaceID, passportID, sourceID, candidate.FieldKey, value, candidate.DataType, status, candidate.Confidence, evidence); err != nil {
			return fmt.Errorf("save field candidate: %w", err)
		}
	}
	return transaction.Commit(ctx)
}

func (s *Store) SetJobProgress(ctx context.Context, jobID string, progress int) error {
	_, err := s.database.Exec(ctx, `UPDATE jobs SET progress = $2, lease_expires_at = now() + interval '5 minutes', updated_at = now() WHERE id = $1::uuid AND status = 'LEASED'`, jobID, progress)
	return err
}

func (s *Store) CompleteJob(ctx context.Context, jobID string) error {
	_, err := s.database.Exec(ctx, `UPDATE jobs SET status = 'SUCCEEDED', progress = 100, lease_expires_at = NULL, completed_at = now(), updated_at = now() WHERE id = $1::uuid`, jobID)
	return err
}

func (s *Store) FailJob(ctx context.Context, job Job, cause error) error {
	status := "QUEUED"
	if job.Attempts >= job.MaxAttempts {
		status = "DEAD_LETTER"
	}
	delay := time.Duration(1<<min(job.Attempts, 6)) * time.Second
	_, err := s.database.Exec(ctx, `
		UPDATE jobs SET status = $2, last_error = $3, available_at = now() + $4::interval,
			lease_expires_at = NULL, completed_at = CASE WHEN $2 = 'DEAD_LETTER' THEN now() END, updated_at = now()
		WHERE id = $1::uuid`, job.ID, status, truncateError(cause.Error()), fmt.Sprintf("%d seconds", int(delay.Seconds())))
	return err
}

func truncateError(message string) string {
	const limit = 1000
	if len(message) > limit {
		return message[:limit]
	}
	return message
}
