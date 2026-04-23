package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestOpenAppliesMigrationsAndPragmas verifies that Open runs the embedded
// goose migrations (users table appears) and applies the pragma baseline
// from docs/spec/sqlite-schema.md §8 (foreign_keys ON + WAL journal).
func TestOpenAppliesMigrationsAndPragmas(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "helling.db") + "?cache=shared"

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	// Migrations applied: a users row schema exists.
	var count int
	if err := pool.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users'`,
	).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected users table to exist, got count=%d", count)
	}

	// PRAGMA foreign_keys should be ON.
	var fk int
	if err := pool.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatalf("pragma foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("expected foreign_keys=1, got %d", fk)
	}

	// PRAGMA journal_mode should be wal.
	var journal string
	if err := pool.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatalf("pragma journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Fatalf("expected journal_mode=wal, got %q", journal)
	}
}

// TestOpenRejectsEmptyDSN ensures the caller gets a clear error instead of
// a useless nil *sql.DB.
func TestOpenRejectsEmptyDSN(t *testing.T) {
	_, err := Open(context.Background(), "")
	if err == nil {
		t.Fatal("expected error on empty DSN, got nil")
	}
}

// TestOpenRejectsUnwritablePath exercises the pragma/migration failure path
// when the DSN refers to a directory SQLite cannot create a file in.
func TestOpenRejectsUnwritablePath(t *testing.T) {
	_, err := Open(context.Background(), "file:/nonexistent/directory/does-not-exist.db?cache=shared")
	if err == nil {
		t.Fatal("expected migrate/pragma error for unwritable path")
	}
}

// TestAllSpecTablesPresent sanity-checks that every top-level table listed
// in docs/spec/sqlite-schema.md exists after migration.
func TestAllSpecTablesPresent(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "spec.db") + "?cache=shared"

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	expected := []string{
		"users", "sessions", "api_tokens",
		"totp_secrets", "recovery_codes", "auth_events",
		"helling_ca", "incus_user_certs",
		"webhooks", "webhook_deliveries",
		"kubernetes_clusters", "kubernetes_nodes",
		"firewall_host_rules", "warnings",
	}

	for _, name := range expected {
		var present int
		err := pool.QueryRowContext(ctx,
			`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&present)
		if err != nil {
			t.Fatalf("query %s: %v", name, err)
		}
		if present != 1 {
			t.Errorf("expected table %q to exist", name)
		}
	}
}
