package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
)

// stubDBOpen swaps openDB with a harmless opener for tests that don't need
// real migrations. Returns via t.Cleanup to restore the real opener.
func stubDBOpen(t *testing.T) {
	t.Helper()
	orig := openDB
	t.Cleanup(func() { openDB = orig })

	openDB = func(_ context.Context, _ string) (*sql.DB, error) {
		return sql.Open("sqlite", "file::memory:?cache=shared")
	}
}

func TestMainUsesInjectedExitAndServe(t *testing.T) {
	origServe := defaultServe
	origExit := exitFunc
	origStderr := stderr
	origArgs := osArgs
	t.Cleanup(func() {
		defaultServe = origServe
		exitFunc = origExit
		stderr = origStderr
		osArgs = origArgs
	})

	stubDBOpen(t)

	defaultServe = func(*http.Server) error {
		return http.ErrServerClosed
	}
	stderr = io.Discard
	osArgs = []string{"hellingd", "-db", "file::memory:?cache=shared"}

	var code int
	exitFunc = func(c int) {
		code = c
	}

	main()

	if code != 0 {
		t.Fatalf("expected exit code 0 from main, got %d", code)
	}
}

func TestRunReturnsZeroOnServerClosed(t *testing.T) {
	stubDBOpen(t)
	logger := newLogger(io.Discard)
	cfg := runConfig{addr: defaultAddr, dsn: "file::memory:?cache=shared"}
	code := run(logger, cfg, func(*http.Server) error {
		return http.ErrServerClosed
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunReturnsOneOnServeFailure(t *testing.T) {
	stubDBOpen(t)
	logger := newLogger(io.Discard)
	cfg := runConfig{addr: defaultAddr, dsn: "file::memory:?cache=shared"}
	code := run(logger, cfg, func(*http.Server) error {
		return errors.New("boom")
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestRunReturnsOneOnDBOpenFailure(t *testing.T) {
	orig := openDB
	t.Cleanup(func() { openDB = orig })
	openDB = func(context.Context, string) (*sql.DB, error) {
		return nil, errors.New("db down")
	}

	logger := newLogger(io.Discard)
	cfg := runConfig{addr: defaultAddr, dsn: "ignored"}
	code := run(logger, cfg, func(*http.Server) error { return nil })
	if code != 1 {
		t.Fatalf("expected exit code 1 on db failure, got %d", code)
	}
}

func TestRunMigrateOnlyReturnsZeroWithoutServe(t *testing.T) {
	stubDBOpen(t)
	logger := newLogger(io.Discard)
	cfg := runConfig{addr: defaultAddr, dsn: "file::memory:?cache=shared", migrateOnly: true}

	called := false
	code := run(logger, cfg, func(*http.Server) error {
		called = true
		return nil
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if called {
		t.Fatal("serve must not be called in migrate-only mode")
	}
}

func TestRunConfiguresHTTPServer(t *testing.T) {
	stubDBOpen(t)
	logger := newLogger(io.Discard)
	cfg := runConfig{addr: defaultAddr, dsn: "file::memory:?cache=shared"}
	var got *http.Server

	code := run(logger, cfg, func(server *http.Server) error {
		got = server
		return http.ErrServerClosed
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got == nil {
		t.Fatal("expected server to be initialized")
	}
	if got.Addr != defaultAddr {
		t.Fatalf("expected addr %q, got %q", defaultAddr, got.Addr)
	}
	if got.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("expected ReadHeaderTimeout %v, got %v", 10*time.Second, got.ReadHeaderTimeout)
	}
}

func TestParseFlagsDefaults(t *testing.T) {
	cfg, err := parseFlags(nil, io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.addr != defaultAddr {
		t.Errorf("addr default = %q, want %q", cfg.addr, defaultAddr)
	}
	if cfg.dsn != defaultDSN {
		t.Errorf("dsn default = %q, want %q", cfg.dsn, defaultDSN)
	}
	if cfg.migrateOnly {
		t.Error("migrate-only should default false")
	}
}

func TestParseFlagsOverrides(t *testing.T) {
	cfg, err := parseFlags([]string{"-addr", ":9999", "-db", "file:/tmp/x.db", "-migrate-only"}, io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.addr != ":9999" {
		t.Errorf("addr = %q", cfg.addr)
	}
	if cfg.dsn != "file:/tmp/x.db" {
		t.Errorf("dsn = %q", cfg.dsn)
	}
	if !cfg.migrateOnly {
		t.Error("migrate-only should be true")
	}
}

// TestDBOpenRealRunsMigrations ensures db.Open applies migrations against a
// real temp DSN, so migrate-only boots actually create the schema.
func TestDBOpenRealRunsMigrations(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "helling.db")
	dsn := "file:" + tmpDB + "?cache=shared"

	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	var n int
	if err := pool.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users'`).Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 1 {
		t.Fatalf("users table missing, count=%d", n)
	}
}
