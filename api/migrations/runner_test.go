package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedRailwayMigrationsAreProviderIndependent(t *testing.T) {
	migrations, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least one embedded migration")
	}

	combined := ""
	for _, migration := range migrations {
		combined += migration.SQL
	}
	for _, forbidden := range []string{"auth.users", "storage.buckets", "CREATE POLICY", "supabase"} {
		if strings.Contains(strings.ToLower(combined), strings.ToLower(forbidden)) {
			t.Fatalf("Railway migration contains provider-specific token %q", forbidden)
		}
	}
	for _, required := range []string{"CREATE EXTENSION IF NOT EXISTS vector", "CREATE TABLE workspaces", "CREATE TABLE jobs"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("Railway migration missing %q", required)
		}
	}
}

func TestEmbeddedMigrationVersionsAreStrictlyIncreasing(t *testing.T) {
	migrations, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index < len(migrations); index++ {
		if migrations[index-1].Version >= migrations[index].Version {
			t.Fatalf("migration versions are not increasing: %d then %d", migrations[index-1].Version, migrations[index].Version)
		}
	}
}

func TestApplicationDraftCacheMigrationIsLoaded(t *testing.T) {
	migrations, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version == 7 && strings.Contains(migration.SQL, "application_draft_templates") && strings.Contains(migration.SQL, "application_draft_cache") {
			return
		}
	}
	t.Fatal("migration 7 must create application draft template and cache tables")
}
