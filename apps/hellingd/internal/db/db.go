// Package db owns the SQLite connection pool and embedded goose migrations
// for hellingd. See docs/spec/sqlite-schema.md and ADR-038.
//
// The driver is modernc.org/sqlite — a pure-Go SQLite, chosen so hellingd
// builds and tests without cgo (aligning with the no-heavy-SDK policy in
// ADR-018). A cgo-backed driver can be swapped in later via build tags if
// performance warrants.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// The modernc driver registers itself as "sqlite"; goose wants its own
// dialect name. Both point at the same *sql.DB.
const (
	driverName   = "sqlite"
	gooseDialect = "sqlite3"
)

// Open returns a configured *sql.DB with the schema fully migrated.
//
// dsn is a SQLite DSN understood by modernc.org/sqlite — for example
// `file:/var/lib/helling/helling.db?cache=shared` for production or
// `file::memory:?cache=shared` for tests.
//
// The returned handle has the PRAGMAs from docs/spec/sqlite-schema.md §8
// applied and migrations at migrations/*.sql advanced to head.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("db: empty DSN")
	}

	pool, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", dsn, err)
	}

	if err := applyPragmas(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("db: apply pragmas: %w", err)
	}

	if err := migrate(pool); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}

	return pool, nil
}

// Pragmas applied to every connection in the pool. The list mirrors
// docs/spec/sqlite-schema.md §8 SQLite PRAGMA Baseline.
var pragmas = []string{
	"PRAGMA journal_mode = WAL",
	"PRAGMA synchronous = NORMAL",
	"PRAGMA foreign_keys = ON",
	"PRAGMA busy_timeout = 5000",
	"PRAGMA temp_store = MEMORY",
}

func applyPragmas(ctx context.Context, pool *sql.DB) error {
	for _, stmt := range pragmas {
		if _, err := pool.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("pragma %q: %w", stmt, err)
		}
	}
	return nil
}

func migrate(pool *sql.DB) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect(gooseDialect); err != nil {
		return fmt.Errorf("set dialect %q: %w", gooseDialect, err)
	}
	if err := goose.Up(pool, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
