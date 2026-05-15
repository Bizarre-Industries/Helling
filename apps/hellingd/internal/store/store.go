// Package store owns SQLite-backed persistence for hellingd.
//
// All access to the database goes through this package. Handlers and
// services never touch *sql.DB directly. The schema is versioned via
// embedded migration files applied at boot.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/age"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the persistence handle. Safe for concurrent use.
type Store struct {
	db           *sql.DB
	ageIdentity  *age.X25519Identity
	ageRecipient *age.X25519Recipient
}

// Open creates the state directory if needed, opens the SQLite database,
// and configures connection-pool settings sensible for a single-node daemon.
func Open(stateDir string) (*Store, error) {
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		return nil, fmt.Errorf("creating state dir %s: %w", stateDir, err)
	}
	if err := ensurePrivateDir(stateDir, 0o750); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(stateDir, "helling.db")
	if err := ensurePrivateFile(dbPath, 0o600); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite at %s: %w", dbPath, err)
	}

	// Single writer is fine for SQLite in WAL mode; many readers OK.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	identity, err := loadOrCreateAgeIdentity(stateDir)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db, ageIdentity: identity, ageRecipient: identity.Recipient()}, nil
}

func ensurePrivateDir(path string, mode os.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	if info.Mode().Perm() != mode {
		return fmt.Errorf("%s has mode %o, want %o", path, info.Mode().Perm(), mode)
	}
	return nil
}

func ensurePrivateFile(path string, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE, mode)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Mode().Perm() != mode {
		return fmt.Errorf("%s has mode %o, want %o", path, info.Mode().Perm(), mode)
	}
	return nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB exposes the underlying *sql.DB to subpackages within store/.
// It is not exported outside store/ via documented API.
func (s *Store) DB() *sql.DB { return s.db }

// Migrate applies any pending migrations. Migrations are SQL files in
// store/migrations/, named "NNN_description.sql" (zero-padded sequence).
// Each file runs in a single transaction. Applied versions are tracked
// in the schema_migrations table.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		// No migrations directory yet; that's fine for the skeleton.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		version, err := parseMigrationVersion(f.Name())
		if err != nil {
			return fmt.Errorf("migration %s: %w", f.Name(), err)
		}

		var alreadyApplied int
		if err := s.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version,
		).Scan(&alreadyApplied); err != nil {
			return fmt.Errorf("checking migration %d: %w", version, err)
		}
		if alreadyApplied > 0 {
			continue
		}

		body, err := migrationsFS.ReadFile("migrations/" + f.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", f.Name(), err)
		}
		sqlText := migrationUpSQL(string(body))

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, sqlText); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("running migration %d (%s): %w", version, f.Name(), err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, strftime('%s', 'now'))`,
			version,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return s.repairLegacyV02State(ctx)
}

func (s *Store) repairLegacyV02State(ctx context.Context) error {
	if err := s.repairV02Schema(ctx); err != nil {
		return err
	}
	if err := s.encryptLegacyWebhookSecrets(ctx); err != nil {
		return err
	}
	return s.encryptLegacyIncusUserCerts(ctx)
}

// parseMigrationVersion extracts the integer prefix from a migration filename.
func parseMigrationVersion(name string) (int, error) {
	var v int
	_, err := fmt.Sscanf(name, "%d", &v)
	if err != nil {
		return 0, fmt.Errorf("malformed migration filename %q: %w", name, err)
	}
	return v, nil
}

func migrationUpSQL(body string) string {
	const (
		upMarker   = "-- +goose Up"
		downMarker = "-- +goose Down"
	)
	up := strings.Index(body, upMarker)
	if up == -1 {
		return body
	}
	body = body[up+len(upMarker):]
	down := strings.Index(body, downMarker)
	if down != -1 {
		body = body[:down]
	}
	body = strings.ReplaceAll(body, "-- +goose StatementBegin", "")
	body = strings.ReplaceAll(body, "-- +goose StatementEnd", "")
	return body
}
