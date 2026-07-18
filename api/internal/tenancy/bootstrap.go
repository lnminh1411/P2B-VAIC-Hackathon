package tenancy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/p2b/p2b/internal/authn"
)

var ErrInvalidSubject = errors.New("invalid authenticated subject")

type Execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Bootstrapper struct {
	database Execer
}

func NewBootstrapper(database Execer) *Bootstrapper {
	return &Bootstrapper{database: database}
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
SELECT $1::uuid, $1::uuid, $4, 'workspace', $1, jsonb_build_object('email', $3)
ON CONFLICT (workspace_id, action) WHERE action = 'auth.bootstrap' DO NOTHING`
