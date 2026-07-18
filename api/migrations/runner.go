package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed sql/*.sql
var files embed.FS

var migrationName = regexp.MustCompile(`^(\d+)_.*\.sql$`)

type Migration struct {
	Version  int64
	Name     string
	Checksum string
	SQL      string
}

func Load() ([]Migration, error) {
	names, err := fs.Glob(files, "sql/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	migrations := make([]Migration, 0, len(names))
	versions := make(map[int64]string, len(names))
	for _, name := range names {
		base := filepath.Base(name)
		matches := migrationName.FindStringSubmatch(base)
		if len(matches) != 2 {
			return nil, fmt.Errorf("invalid migration filename %q", base)
		}
		version, parseErr := strconv.ParseInt(matches[1], 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", base, parseErr)
		}
		if previous, exists := versions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %d in %q and %q", version, previous, base)
		}
		body, readErr := files.ReadFile(name)
		if readErr != nil {
			return nil, fmt.Errorf("read migration %q: %w", base, readErr)
		}
		hash := sha256.Sum256(body)
		versions[version] = base
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     base,
			Checksum: hex.EncodeToString(hash[:]),
			SQL:      string(body),
		})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func Run(ctx context.Context, connection *pgx.Conn) error {
	if connection == nil {
		return errors.New("database connection is nil")
	}
	migrations, err := Load()
	if err != nil {
		return err
	}
	if _, err = connection.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended('p2b-schema-migrations', 0))`); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		unlockContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = connection.Exec(unlockContext, `SELECT pg_advisory_unlock(hashtextextended('p2b-schema-migrations', 0))`)
	}()

	if _, err = connection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			checksum text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	applied := map[int64]string{}
	rows, err := connection.Query(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	for rows.Next() {
		var version int64
		var checksum string
		if scanErr := rows.Scan(&version, &checksum); scanErr != nil {
			rows.Close()
			return fmt.Errorf("scan migration ledger: %w", scanErr)
		}
		applied[version] = checksum
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate migration ledger: %w", err)
	}
	rows.Close()

	for _, migration := range migrations {
		if checksum, exists := applied[migration.Version]; exists {
			if checksum != migration.Checksum {
				return fmt.Errorf("migration %d checksum changed after apply", migration.Version)
			}
			continue
		}
		transaction, beginErr := connection.Begin(ctx)
		if beginErr != nil {
			return fmt.Errorf("begin migration %d: %w", migration.Version, beginErr)
		}
		if _, execErr := transaction.Exec(ctx, migration.SQL); execErr != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, execErr)
		}
		if _, execErr := transaction.Exec(ctx,
			`INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`,
			migration.Version, migration.Name, migration.Checksum,
		); execErr != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("record migration %d: %w", migration.Version, execErr)
		}
		if commitErr := transaction.Commit(ctx); commitErr != nil {
			return fmt.Errorf("commit migration %d: %w", migration.Version, commitErr)
		}
	}
	return nil
}
