package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/domain"
	passportdomain "github.com/p2b/p2b/internal/passport"
)

var (
	ErrNotFound        = errors.New("pipeline resource not found")
	ErrVersionConflict = errors.New("passport version conflict")
)

type Source struct {
	ID          string
	WorkspaceID string
	Filename    string
	ContentType string
	SizeBytes   int64
	ObjectKey   string
}

type SourceRecord struct {
	Source
	Status string
}

type BuildRequest struct {
	CompanyName    string
	Website        string
	SupportNeeds   []string
	SourceIDs      []string
	IdempotencyKey string
	ActorSubject   string
	Refresh        bool
}

type Job struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	CreatedAt   time.Time  `json:"created_at"`
	LastError   string     `json:"last_error,omitempty"`
	Attempts    int        `json:"attempts"`
	WorkspaceID string     `json:"-"`
	Payload     JobPayload `json:"-"`
	MaxAttempts int        `json:"-"`
}

type JobPayload struct {
	CompanyName  string   `json:"company_name"`
	Website      string   `json:"website"`
	SupportNeeds []string `json:"support_needs"`
	SourceIDs    []string `json:"source_ids"`
	Refresh      bool     `json:"refresh"`
}

type Store struct{ database *pgxpool.Pool }

func NewStore(database *pgxpool.Pool) *Store { return &Store{database: database} }

func (s *Store) RegisterSource(ctx context.Context, source Source) error {
	_, err := s.database.Exec(ctx, `
		INSERT INTO company_sources (id, workspace_id, filename, content_type, size_bytes, object_key, status)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, 'PENDING_UPLOAD')`,
		source.ID, source.WorkspaceID, source.Filename, source.ContentType, source.SizeBytes, source.ObjectKey)
	if err != nil {
		return fmt.Errorf("register company source: %w", err)
	}
	return nil
}

func (s *Store) MarkUploaded(ctx context.Context, workspaceID, sourceID string) error {
	tag, err := s.database.Exec(ctx, `
		UPDATE company_sources SET status = 'UPLOADED', updated_at = now()
		WHERE id = $1::uuid AND workspace_id = $2::uuid AND status = 'PENDING_UPLOAD'`, sourceID, workspaceID)
	if err != nil {
		return fmt.Errorf("mark source uploaded: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) EnqueueBuild(ctx context.Context, workspaceID string, request BuildRequest) (Job, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return Job{}, errors.New("invalid workspace")
	}
	actorUUID, err := uuid.Parse(request.ActorSubject)
	if err != nil {
		return Job{}, errors.New("invalid actor")
	}
	sourceUUIDs := make([]uuid.UUID, 0, len(request.SourceIDs))
	for _, sourceID := range request.SourceIDs {
		parsed, parseErr := uuid.Parse(sourceID)
		if parseErr != nil {
			return Job{}, errors.New("invalid source id")
		}
		sourceUUIDs = append(sourceUUIDs, parsed)
	}
	if request.Refresh {
		if err := ValidateRefreshRequest(request); err != nil {
			return Job{}, err
		}
	}
	transaction, err := s.database.Begin(ctx)
	if err != nil {
		return Job{}, fmt.Errorf("begin passport build: %w", err)
	}
	defer transaction.Rollback(ctx)
	jobType := "PASSPORT_BUILD"
	idempotencyScope := "passport-build"
	if request.Refresh {
		jobType = "PASSPORT_REFRESH"
		idempotencyScope = "passport-refresh"
	}
	idempotency := workspaceID + ":" + idempotencyScope + ":" + request.IdempotencyKey
	var existing Job
	err = transaction.QueryRow(ctx, `
		SELECT id, type, status, progress, created_at, attempts, max_attempts, COALESCE(last_error, ''), payload
		FROM jobs WHERE idempotency_key = $1`, idempotency).
		Scan(&existing.ID, &existing.Type, &existing.Status, &existing.Progress, &existing.CreatedAt, &existing.Attempts, &existing.MaxAttempts, &existing.LastError, &existing.Payload)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Job{}, fmt.Errorf("check passport build idempotency: %w", err)
	}
	if len(sourceUUIDs) > 0 {
		var count int
		if err = transaction.QueryRow(ctx, `SELECT count(*) FROM company_sources WHERE workspace_id = $1 AND id = ANY($2::uuid[]) AND status = 'UPLOADED'`, workspaceUUID, sourceUUIDs).Scan(&count); err != nil {
			return Job{}, fmt.Errorf("validate sources: %w", err)
		}
		if count != len(sourceUUIDs) {
			return Job{}, errors.New("one or more PDF uploads are missing or incomplete")
		}
	}
	var passportID uuid.UUID
	if request.Refresh {
		if err = transaction.QueryRow(ctx, `SELECT id FROM passports WHERE workspace_id = $1`, workspaceUUID).Scan(&passportID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Job{}, errors.New("cannot refresh a business without a passport")
			}
			return Job{}, fmt.Errorf("load passport for refresh: %w", err)
		}
	} else {
		if _, err = transaction.Exec(ctx, `
			INSERT INTO companies (workspace_id, legal_name, website, support_needs)
			VALUES ($1, $2, NULLIF($3, ''), $4)
			ON CONFLICT (workspace_id) DO UPDATE SET legal_name = EXCLUDED.legal_name, website = EXCLUDED.website,
				support_needs = EXCLUDED.support_needs, updated_at = now()`, workspaceUUID, request.CompanyName, request.Website, request.SupportNeeds); err != nil {
			return Job{}, fmt.Errorf("save company: %w", err)
		}
		var version int
		if err = transaction.QueryRow(ctx, `
			INSERT INTO passports (workspace_id, current_version) VALUES ($1, 1)
			ON CONFLICT (workspace_id) DO UPDATE SET current_version = passports.current_version + 1, updated_at = now()
			RETURNING id, current_version`, workspaceUUID).Scan(&passportID, &version); err != nil {
			return Job{}, fmt.Errorf("save passport: %w", err)
		}
		now := time.Now().UTC()
		fields := initialFields(workspaceID, request.CompanyName, request.Website, now)
		encodedFields, _ := json.Marshal(fields)
		if _, err = transaction.Exec(ctx, `INSERT INTO passport_versions (passport_id, version, fields, support_needs, created_by) VALUES ($1, $2, $3, $4, $5)`, passportID, version, encodedFields, request.SupportNeeds, actorUUID); err != nil {
			return Job{}, fmt.Errorf("save passport version: %w", err)
		}
	}
	payload := JobPayload{CompanyName: request.CompanyName, Website: request.Website, SupportNeeds: request.SupportNeeds, SourceIDs: request.SourceIDs, Refresh: request.Refresh}
	encodedPayload, _ := json.Marshal(payload)
	status, progress := "QUEUED", 0
	if len(sourceUUIDs) == 0 {
		status, progress = "SUCCEEDED", 100
	}
	jobID := uuid.New()
	var job Job
	if err = transaction.QueryRow(ctx, `
		INSERT INTO jobs (id, workspace_id, type, payload, idempotency_key, status, progress, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CASE WHEN $6 = 'SUCCEEDED' THEN now() END)
		ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING id, type, status, progress, created_at, attempts, max_attempts, COALESCE(last_error, ''), payload`,
		jobID, workspaceUUID, jobType, encodedPayload, idempotency, status, progress).Scan(&job.ID, &job.Type, &job.Status, &job.Progress, &job.CreatedAt, &job.Attempts, &job.MaxAttempts, &job.LastError, &job.Payload); err != nil {
		return Job{}, fmt.Errorf("enqueue passport build: %w", err)
	}
	if err = transaction.Commit(ctx); err != nil {
		return Job{}, fmt.Errorf("commit passport build: %w", err)
	}
	return job, nil
}

func (s *Store) EnqueueRefresh(ctx context.Context, workspaceID string, sourceIDs []string, idempotencyKey, actorSubject string) (Job, error) {
	return s.EnqueueBuild(ctx, workspaceID, BuildRequest{SourceIDs: sourceIDs, IdempotencyKey: idempotencyKey, ActorSubject: actorSubject, Refresh: true})
}

func ValidateRefreshRequest(request BuildRequest) error {
	if len(request.SourceIDs) == 0 {
		return errors.New("at least one new PDF source is required")
	}
	if len(request.SourceIDs) > 10 {
		return errors.New("at most 10 PDF sources are allowed")
	}
	return nil
}

func (s *Store) Job(ctx context.Context, workspaceID, jobID string) (Job, error) {
	var job Job
	err := s.database.QueryRow(ctx, `SELECT id, type, status, progress, created_at, attempts, max_attempts, COALESCE(last_error, ''), payload FROM jobs WHERE id = $1::uuid AND workspace_id = $2::uuid`, jobID, workspaceID).
		Scan(&job.ID, &job.Type, &job.Status, &job.Progress, &job.CreatedAt, &job.Attempts, &job.MaxAttempts, &job.LastError, &job.Payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return job, err
}

func (s *Store) Passport(ctx context.Context, workspaceID string) (domain.Passport, error) {
	var result domain.Passport
	var encodedFields []byte
	err := s.database.QueryRow(ctx, `
		SELECT p.id, c.legal_name, COALESCE(c.website, ''), c.support_needs, p.current_version, pv.fields, p.updated_at
		FROM passports p JOIN companies c ON c.workspace_id = p.workspace_id
		JOIN passport_versions pv ON pv.passport_id = p.id AND pv.version = p.current_version
		WHERE p.workspace_id = $1::uuid`, workspaceID).
		Scan(&result.ID, &result.CompanyName, &result.Website, &result.SupportNeeds, &result.Version, &encodedFields, &result.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Passport{Fields: map[string]domain.PassportField{}}, nil
	}
	if err != nil {
		return domain.Passport{}, fmt.Errorf("read passport: %w", err)
	}
	if err = json.Unmarshal(encodedFields, &result.Fields); err != nil {
		return domain.Passport{}, fmt.Errorf("decode passport fields: %w", err)
	}
	return passportdomain.EnsureCanonicalFields(result), nil
}

func (s *Store) Candidates(ctx context.Context, workspaceID string) ([]passportdomain.Candidate, error) {
	rows, err := s.database.Query(ctx, `
		SELECT id, field_key, value, data_type, confidence, evidence
		FROM field_candidates WHERE workspace_id = $1::uuid AND status IN ('EXTRACTED','NEEDS_REVIEW','CONFLICTED')
		ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]passportdomain.Candidate, 0)
	for rows.Next() {
		var candidate passportdomain.Candidate
		var encodedValue, encodedEvidence []byte
		if err = rows.Scan(&candidate.ID, &candidate.FieldKey, &encodedValue, &candidate.DataType, &candidate.Confidence, &encodedEvidence); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(encodedValue, &candidate.Value); err != nil {
			return nil, err
		}
		var evidence domain.Evidence
		if err = json.Unmarshal(encodedEvidence, &evidence); err != nil {
			return nil, err
		}
		candidate.Evidence = evidence
		candidate.Status = "NEEDS_REVIEW"
		result = append(result, candidate)
	}
	return result, rows.Err()
}

func (s *Store) ConfirmField(ctx context.Context, workspaceID, actorSubject, fieldKey string, value any, expectedVersion int) (domain.Passport, error) {
	actorID, err := uuid.Parse(actorSubject)
	if err != nil {
		return domain.Passport{}, errors.New("invalid actor")
	}
	transaction, err := s.database.Begin(ctx)
	if err != nil {
		return domain.Passport{}, err
	}
	defer transaction.Rollback(ctx)
	var passportID uuid.UUID
	var currentVersion int
	var encodedFields []byte
	var supportNeeds []string
	if err = transaction.QueryRow(ctx, `
		SELECT p.id, p.current_version, pv.fields, pv.support_needs
		FROM passports p JOIN passport_versions pv ON pv.passport_id = p.id AND pv.version = p.current_version
		WHERE p.workspace_id = $1::uuid FOR UPDATE OF p`, workspaceID).
		Scan(&passportID, &currentVersion, &encodedFields, &supportNeeds); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Passport{}, ErrNotFound
		}
		return domain.Passport{}, err
	}
	if currentVersion != expectedVersion {
		return domain.Passport{}, ErrVersionConflict
	}
	var candidateID uuid.UUID
	var dataType string
	var encodedEvidence []byte
	if err = transaction.QueryRow(ctx, `
		SELECT id, data_type, evidence FROM field_candidates
		WHERE workspace_id = $1::uuid AND field_key = $2 AND status IN ('EXTRACTED','NEEDS_REVIEW','CONFLICTED')
		ORDER BY created_at DESC LIMIT 1`, workspaceID, fieldKey).Scan(&candidateID, &dataType, &encodedEvidence); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.Passport{}, err
		}
	}
	fields := map[string]domain.PassportField{}
	if err = json.Unmarshal(encodedFields, &fields); err != nil {
		return domain.Passport{}, err
	}
	pass := passportdomain.EnsureCanonicalFields(domain.Passport{Fields: fields})
	fields = pass.Fields
	definition, known := passportdomain.LookupField(fieldKey)
	if !known {
		return domain.Passport{}, fmt.Errorf("unknown passport field %q", fieldKey)
	}
	if value == nil || fmt.Sprint(value) == "" {
		return domain.Passport{}, errors.New("confirmed value is required")
	}
	if err = passportdomain.ValidateFieldValue(fieldKey, value); err != nil {
		return domain.Passport{}, err
	}
	field := fields[fieldKey]
	evidence := field.Evidence
	if len(encodedEvidence) > 0 {
		var candidateEvidence domain.Evidence
		if err = json.Unmarshal(encodedEvidence, &candidateEvidence); err != nil {
			return domain.Passport{}, err
		}
		evidence = append(evidence, candidateEvidence)
	}
	evidence = append(evidence, domain.Evidence{
		SourceID: "user-input", SourceName: "Người dùng xác nhận", Quote: fmt.Sprint(value),
		ContentHash: fmt.Sprintf("user-confirmation:%s:v%d", actorSubject, currentVersion+1), ObservedAt: time.Now().UTC(),
	})
	fields[fieldKey] = domain.PassportField{Key: fieldKey, Label: definition.Label, Value: value, DataType: definition.DataType, Status: domain.FieldConfirmed, Confidence: 1, Evidence: evidence}
	encodedFields, _ = json.Marshal(fields)
	newVersion := currentVersion + 1
	if _, err = transaction.Exec(ctx, `INSERT INTO passport_versions (passport_id, version, fields, support_needs, created_by) VALUES ($1, $2, $3, $4, $5)`, passportID, newVersion, encodedFields, supportNeeds, actorID); err != nil {
		return domain.Passport{}, err
	}
	if _, err = transaction.Exec(ctx, `UPDATE passports SET current_version = $2, updated_at = now() WHERE id = $1`, passportID, newVersion); err != nil {
		return domain.Passport{}, err
	}
	if candidateID != uuid.Nil {
		if _, err = transaction.Exec(ctx, `UPDATE field_candidates SET status = 'ACCEPTED' WHERE id = $1`, candidateID); err != nil {
			return domain.Passport{}, err
		}
	}
	if fieldKey == "legal_name" {
		if _, err = transaction.Exec(ctx, `
			INSERT INTO companies (workspace_id, legal_name, support_needs)
			VALUES ($1::uuid, $2, '{}')
			ON CONFLICT (workspace_id) DO UPDATE SET legal_name = EXCLUDED.legal_name, updated_at = now()`, workspaceID, fmt.Sprint(value)); err != nil {
			return domain.Passport{}, err
		}
		if _, err = transaction.Exec(ctx, `UPDATE workspaces SET display_name = $2, updated_at = now() WHERE id = $1::uuid`, workspaceID, fmt.Sprint(value)); err != nil {
			return domain.Passport{}, err
		}
	} else if fieldKey == "website" {
		if _, err = transaction.Exec(ctx, `
			INSERT INTO companies (workspace_id, legal_name, website, support_needs)
			VALUES ($1::uuid, 'Chưa có tên', NULLIF($2, ''), '{}')
			ON CONFLICT (workspace_id) DO UPDATE SET website = EXCLUDED.website, updated_at = now()`, workspaceID, fmt.Sprint(value)); err != nil {
			return domain.Passport{}, err
		}
	}
	if err = transaction.Commit(ctx); err != nil {
		return domain.Passport{}, err
	}
	return s.Passport(ctx, workspaceID)
}

func initialFields(workspaceID, companyName, website string, now time.Time) map[string]domain.PassportField {
	userEvidence := domain.Evidence{SourceID: "user-input", SourceName: "Thông tin do người dùng cung cấp", Quote: companyName, ContentHash: "user:" + workspaceID, ObservedAt: now}
	fields := map[string]domain.PassportField{
		"legal_name": {Key: "legal_name", Label: "Tên pháp lý", Value: companyName, DataType: "string", Status: domain.FieldConfirmed, Confidence: 1, Evidence: []domain.Evidence{userEvidence}},
	}
	if website != "" {
		fields["website"] = domain.PassportField{Key: "website", Label: "Website", Value: website, DataType: "url", Status: domain.FieldConfirmed, Confidence: 1, Evidence: []domain.Evidence{{SourceID: "user-input", SourceName: "Website do người dùng cung cấp", URL: website, Quote: website, ContentHash: "user:" + workspaceID, ObservedAt: now}}}
	}
	return fields
}

func fieldLabel(key string) string {
	if definition, exists := passportdomain.LookupField(key); exists {
		return definition.Label
	}
	return key
}
