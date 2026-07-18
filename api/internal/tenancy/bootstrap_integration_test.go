package tenancy

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/p2b/p2b/internal/authn"
)

func TestBootstrapperAgainstPostgres(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	connection, err := pgx.Connect(t.Context(), databaseURL)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close(context.Background()) })

	transaction, err := connection.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback(context.Background()) })

	subject := uuid.NewString()
	principal := authn.Principal{
		Subject: subject,
		Email:   "integration-test@p2b.invalid",
		Name:    "Integration test workspace",
	}
	if err := NewBootstrapper(transaction).Ensure(t.Context(), principal); err != nil {
		t.Fatalf("bootstrap workspace: %v", err)
	}
}
