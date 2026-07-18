package tenancy

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/p2b/p2b/internal/authn"
)

type execStub struct {
	query string
	args  []any
	err   error
}

func (stub *execStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	stub.query, stub.args = query, args
	return pgconn.NewCommandTag("INSERT 0 1"), stub.err
}

func TestBootstrapperCreatesWorkspaceFromVerifiedSubject(t *testing.T) {
	database := &execStub{}
	bootstrapper := NewBootstrapper(database)
	principal := authn.Principal{Subject: "0f34fe4f-37dc-43c0-9277-67e49b7b06b5", Email: "founder@p2b.vn", Name: "Nguyễn Minh Anh"}

	if err := bootstrapper.Ensure(t.Context(), principal); err != nil {
		t.Fatal(err)
	}
	if len(database.args) != 4 || database.args[0] != principal.Subject || database.args[1] != principal.Name || database.args[2] != principal.Email {
		t.Fatalf("args = %#v", database.args)
	}
}

func TestBootstrapperRejectsInvalidSubject(t *testing.T) {
	database := &execStub{}
	if err := NewBootstrapper(database).Ensure(t.Context(), authn.Principal{Subject: "not-a-uuid"}); !errors.Is(err, ErrInvalidSubject) {
		t.Fatalf("error = %v", err)
	}
}
