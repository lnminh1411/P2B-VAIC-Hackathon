package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p2b/p2b/internal/platform"
)

var (
	ErrNotFound = errors.New("application cache not found")
	ErrConflict = errors.New("application cache version conflict")
)

type Template struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Filename     string    `json:"filename"`
	ContentType  string    `json:"content_type"`
	SourceText   string    `json:"-"`
	Placeholders []string  `json:"placeholders"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Store struct{ database *pgxpool.Pool }

func NewStore(database *pgxpool.Pool) *Store { return &Store{database: database} }

func (s *Store) CreateTemplate(ctx context.Context, workspaceID, name, filename, contentType, sourceText string) (Template, error) {
	name, filename, sourceText = strings.TrimSpace(name), strings.TrimSpace(filename), strings.TrimSpace(sourceText)
	if name == "" || len(name) > 160 || filename == "" || len(filename) > 255 || sourceText == "" || len(sourceText) > 500_000 {
		return Template{}, errors.New("invalid application template")
	}
	placeholders := ExtractPlaceholders(sourceText)
	placeholderJSON, err := json.Marshal(placeholders)
	if err != nil {
		return Template{}, fmt.Errorf("encode template placeholders: %w", err)
	}
	template := Template{ID: uuid.NewString(), Name: name, Filename: filename, ContentType: contentType, SourceText: sourceText, Placeholders: placeholders}
	err = s.database.QueryRow(ctx, `
		INSERT INTO application_draft_templates (id, workspace_id, name, filename, content_type, source_text, placeholders)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::jsonb)
		RETURNING created_at, updated_at`, template.ID, workspaceID, name, filename, contentType, sourceText, placeholderJSON,
	).Scan(&template.CreatedAt, &template.UpdatedAt)
	if err != nil {
		return Template{}, fmt.Errorf("create application template: %w", err)
	}
	return template, nil
}

func (s *Store) Templates(ctx context.Context, workspaceID string) ([]Template, error) {
	rows, err := s.database.Query(ctx, `
		SELECT id::text, name, filename, content_type, placeholders, created_at, updated_at
		FROM application_draft_templates WHERE workspace_id = $1::uuid ORDER BY updated_at DESC LIMIT 100`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list application templates: %w", err)
	}
	defer rows.Close()
	result := []Template{}
	for rows.Next() {
		var template Template
		var placeholders []byte
		if err = rows.Scan(&template.ID, &template.Name, &template.Filename, &template.ContentType, &placeholders, &template.CreatedAt, &template.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan application template: %w", err)
		}
		if err = json.Unmarshal(placeholders, &template.Placeholders); err != nil {
			return nil, fmt.Errorf("decode application placeholders: %w", err)
		}
		result = append(result, template)
	}
	return result, rows.Err()
}

func (s *Store) Template(ctx context.Context, workspaceID, id string) (Template, error) {
	var template Template
	var placeholders []byte
	err := s.database.QueryRow(ctx, `
		SELECT id::text, name, filename, content_type, source_text, placeholders, created_at, updated_at
		FROM application_draft_templates WHERE workspace_id = $1::uuid AND id = $2::uuid`, workspaceID, id,
	).Scan(&template.ID, &template.Name, &template.Filename, &template.ContentType, &template.SourceText, &placeholders, &template.CreatedAt, &template.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("get application template: %w", err)
	}
	if err = json.Unmarshal(placeholders, &template.Placeholders); err != nil {
		return Template{}, fmt.Errorf("decode application placeholders: %w", err)
	}
	return template, nil
}

func (s *Store) SaveDraft(ctx context.Context, workspaceID string, draft platform.Application) error {
	sections, err := json.Marshal(draft.Sections)
	if err != nil {
		return fmt.Errorf("encode application sections: %w", err)
	}
	blockers, err := json.Marshal(draft.BlockingReasons)
	if err != nil {
		return fmt.Errorf("encode application blockers: %w", err)
	}
	var templateID any
	if draft.TemplateID != "" {
		templateID = draft.TemplateID
	}
	result, err := s.database.Exec(ctx, `
		INSERT INTO application_draft_cache (
			application_id, workspace_id, checklist_id, policy_id, policy_title, policy_agency,
			passport_version, policy_version, template_id, template_name, template_version,
			version, status, sections, blocking_reasons, generation_warning, updated_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9::uuid, $10, $11, $12, $13, $14::jsonb, $15::jsonb, $16, $17)
		ON CONFLICT (application_id) DO UPDATE SET
			template_id = EXCLUDED.template_id, template_name = EXCLUDED.template_name,
			version = EXCLUDED.version, status = EXCLUDED.status, sections = EXCLUDED.sections,
			blocking_reasons = EXCLUDED.blocking_reasons, generation_warning = EXCLUDED.generation_warning,
			updated_at = EXCLUDED.updated_at
		WHERE application_draft_cache.workspace_id = EXCLUDED.workspace_id AND application_draft_cache.version < EXCLUDED.version`,
		draft.ID, workspaceID, draft.ChecklistID, draft.PolicyID, draft.PolicyTitle, draft.PolicyAgency,
		draft.PassportVersion, draft.PolicyVersion, templateID, draft.TemplateName, draft.TemplateVersion,
		draft.Version, draft.Status, sections, blockers, draft.GenerationWarning, draft.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save application draft: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (s *Store) Draft(ctx context.Context, workspaceID, id string) (platform.Application, error) {
	return scanDraft(s.database.QueryRow(ctx, draftSelect+` WHERE workspace_id = $1::uuid AND application_id = $2::uuid`, workspaceID, id))
}

func (s *Store) LatestDraft(ctx context.Context, workspaceID string) (platform.Application, error) {
	return scanDraft(s.database.QueryRow(ctx, draftSelect+` WHERE workspace_id = $1::uuid ORDER BY updated_at DESC LIMIT 1`, workspaceID))
}

const draftSelect = `SELECT application_id::text, checklist_id, policy_id, policy_title, policy_agency,
	passport_version, policy_version, COALESCE(template_id::text, ''), template_name, template_version,
	version, status, sections, blocking_reasons, generation_warning, updated_at FROM application_draft_cache`

type rowScanner interface{ Scan(...any) error }

func scanDraft(row rowScanner) (platform.Application, error) {
	var draft platform.Application
	var sections, blockers []byte
	err := row.Scan(&draft.ID, &draft.ChecklistID, &draft.PolicyID, &draft.PolicyTitle, &draft.PolicyAgency,
		&draft.PassportVersion, &draft.PolicyVersion, &draft.TemplateID, &draft.TemplateName, &draft.TemplateVersion,
		&draft.Version, &draft.Status, &sections, &blockers, &draft.GenerationWarning, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.Application{}, ErrNotFound
	}
	if err != nil {
		return platform.Application{}, fmt.Errorf("get application draft: %w", err)
	}
	if err = json.Unmarshal(sections, &draft.Sections); err != nil {
		return platform.Application{}, fmt.Errorf("decode application sections: %w", err)
	}
	if err = json.Unmarshal(blockers, &draft.BlockingReasons); err != nil {
		return platform.Application{}, fmt.Errorf("decode application blockers: %w", err)
	}
	return draft, nil
}
