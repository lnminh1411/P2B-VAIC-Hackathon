package tenancy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/p2b/p2b/internal/authn"
	"github.com/p2b/p2b/internal/domain"
)

var ErrInvalidSubject = errors.New("invalid authenticated subject")
var ErrWorkspaceAccess = errors.New("workspace access denied")

type Execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Bootstrapper struct {
	database Execer
	queryer  Queryer
}

func NewBootstrapper(database Execer) *Bootstrapper {
	bootstrapper := &Bootstrapper{database: database}
	if queryer, ok := database.(Queryer); ok {
		bootstrapper.queryer = queryer
	}
	return bootstrapper
}

func (b *Bootstrapper) Ensure(ctx context.Context, principal authn.Principal) error {
	if b.database == nil {
		return errors.New("workspace database is unavailable")
	}
	if _, err := uuid.Parse(principal.Subject); err != nil {
		return ErrInvalidSubject
	}
	displayName := strings.TrimSpace(principal.Name)
	if displayName == "" {
		displayName = "Workspace doanh nghiệp"
	}
	_, err := b.database.Exec(ctx, bootstrapWorkspaceSQL, principal.Subject, displayName, strings.TrimSpace(principal.Email), "auth.bootstrap")
	if err != nil {
		return fmt.Errorf("bootstrap workspace: %w", err)
	}
	return nil
}

func (b *Bootstrapper) Resolve(ctx context.Context, principal authn.Principal, requestedID string) (string, error) {
	if err := validateSubject(principal.Subject); err != nil {
		return "", err
	}
	requestedID = strings.TrimSpace(requestedID)
	if requestedID == "" {
		return principal.Subject, nil
	}
	if _, err := uuid.Parse(requestedID); err != nil || b.queryer == nil {
		return "", ErrWorkspaceAccess
	}
	var workspaceID string
	err := b.queryer.QueryRow(ctx, `
		SELECT workspace_id::text FROM workspace_members
		WHERE subject = $1::uuid AND workspace_id = $2::uuid`, principal.Subject, requestedID).Scan(&workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrWorkspaceAccess
	}
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	return workspaceID, nil
}

func (b *Bootstrapper) List(ctx context.Context, principal authn.Principal) ([]domain.Workspace, error) {
	if err := validateSubject(principal.Subject); err != nil {
		return nil, err
	}
	if b.queryer == nil {
		return nil, errors.New("workspace database is unavailable")
	}
	rows, err := b.queryer.Query(ctx, `
		SELECT w.id::text, COALESCE(c.legal_name, w.display_name) AS display_name, wm.role, w.created_at
		FROM workspace_members wm 
		JOIN workspaces w ON w.id = wm.workspace_id
		LEFT JOIN companies c ON c.workspace_id = w.id
		WHERE wm.subject = $1::uuid ORDER BY w.created_at, w.id`, principal.Subject)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()
	workspaces := make([]domain.Workspace, 0)
	for rows.Next() {
		var workspace domain.Workspace
		if err := rows.Scan(&workspace.ID, &workspace.DisplayName, &workspace.Role, &workspace.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		workspaces = append(workspaces, workspace)
	}
	return workspaces, rows.Err()
}

func (b *Bootstrapper) Create(ctx context.Context, principal authn.Principal, displayName string) (domain.Workspace, error) {
	if err := validateSubject(principal.Subject); err != nil {
		return domain.Workspace{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len([]rune(displayName)) > 200 {
		return domain.Workspace{}, errors.New("display_name is required and limited to 200 characters")
	}
	if b.database == nil {
		return domain.Workspace{}, errors.New("workspace database is unavailable")
	}
	workspaceID := uuid.New()
	if _, err := b.database.Exec(ctx, `
		INSERT INTO workspaces (id, owner_subject, display_name) VALUES ($1, $2::uuid, $3)`, workspaceID, principal.Subject, displayName); err != nil {
		return domain.Workspace{}, fmt.Errorf("create workspace: %w", err)
	}
	if _, err := b.database.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, subject, role) VALUES ($1, $2::uuid, 'OWNER')`, workspaceID, principal.Subject); err != nil {
		return domain.Workspace{}, fmt.Errorf("create workspace membership: %w", err)
	}
	return domain.Workspace{ID: workspaceID.String(), DisplayName: displayName, Role: "OWNER", CreatedAt: time.Now().UTC()}, nil
}

func (b *Bootstrapper) Delete(ctx context.Context, principal authn.Principal, workspaceID string) error {
	if err := validateSubject(principal.Subject); err != nil {
		return err
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return ErrWorkspaceAccess
	}
	if workspaceID == principal.Subject {
		return errors.New("the default workspace cannot be deleted")
	}
	if b.database == nil {
		return errors.New("workspace database is unavailable")
	}
	tag, err := b.database.Exec(ctx, `
		DELETE FROM workspaces WHERE id = $1::uuid AND owner_subject = $2::uuid`, workspaceID, principal.Subject)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWorkspaceAccess
	}
	return nil
}

func validateSubject(subject string) error {
	if _, err := uuid.Parse(subject); err != nil {
		return ErrInvalidSubject
	}
	return nil
}

const bootstrapWorkspaceSQL = `
WITH workspace_insert AS (
    INSERT INTO workspaces (id, owner_subject, display_name)
    VALUES ($1::uuid, $1::uuid, $2)
    ON CONFLICT (id) DO UPDATE SET updated_at = now()
    RETURNING id
), member_insert AS (
    INSERT INTO workspace_members (workspace_id, subject, role)
    VALUES ($1::uuid, $1::uuid, 'OWNER')
    ON CONFLICT (workspace_id, subject) DO NOTHING
)
INSERT INTO audit_events (workspace_id, actor_subject, action, aggregate_type, aggregate_id, metadata)
SELECT $1::uuid, $1::uuid, $4, 'workspace', $1, jsonb_build_object('email', $3::text)
ON CONFLICT (workspace_id, action) WHERE action = 'auth.bootstrap' DO NOTHING`
