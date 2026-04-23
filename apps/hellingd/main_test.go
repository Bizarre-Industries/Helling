package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
)

func ed25519GenerateForTest() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func x509MarshalPKCS8PrivateKey(priv ed25519.PrivateKey) ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(priv)
}

func pemEncodeForTest(typ string, data []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: data})
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

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

// TestBuildAuthServiceEphemeralKey exercises the no-env key generation branch.
func TestBuildAuthServiceEphemeralKey(t *testing.T) {
	t.Setenv(jwtKeyPathEnvVar, "")
	dsn := "file:" + filepath.Join(t.TempDir(), "auth.db") + "?cache=shared"
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	svc, err := buildAuthService(newLogger(io.Discard), pool)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if svc == nil || svc.Signer() == nil || svc.Repo() == nil {
		t.Fatal("service should be fully initialized")
	}
}

// TestLoadSigningKey_MissingFileErrors verifies the operator-supplied path
// error path.
func TestLoadSigningKey_MissingFileErrors(t *testing.T) {
	t.Setenv(jwtKeyPathEnvVar, filepath.Join(t.TempDir(), "does-not-exist.pem"))
	if _, err := loadOrGenerateSigningKey(newLogger(io.Discard)); err == nil {
		t.Fatal("expected error for missing key file")
	}
}

// TestParseFlagsHelpExits confirms -h surfaces flag.ErrHelp.
func TestParseFlagsHelpExits(t *testing.T) {
	_, err := parseFlags([]string{"-h"}, io.Discard)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

// TestParseFlagsUnknownReturnsError confirms unknown flag surfaces an error.
func TestParseFlagsUnknownReturnsError(t *testing.T) {
	if _, err := parseFlags([]string{"-unknown"}, io.Discard); err == nil {
		t.Fatal("expected parse error")
	}
}

// TestLoadSigningKey_FromPEMFile covers the successful PEM+PKCS8 load path.
func TestLoadSigningKey_FromPEMFile(t *testing.T) {
	_, priv, err := ed25519GenerateForTest()
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pemEncodeForTest("PRIVATE KEY", pkcs8)
	path := filepath.Join(t.TempDir(), "jwt.pem")
	if err := writeFile(path, pemBytes); err != nil {
		t.Fatal(err)
	}

	t.Setenv(jwtKeyPathEnvVar, path)
	loaded, err := loadOrGenerateSigningKey(newLogger(io.Discard))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) == 0 {
		t.Fatal("loaded key should be non-empty")
	}
}

// TestLoadSigningKey_EmptyFileErrors covers PEM-decode-failed path.
func TestLoadSigningKey_EmptyFileErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.pem")
	if err := writeFile(path, []byte("not-a-pem")); err != nil {
		t.Fatal(err)
	}
	t.Setenv(jwtKeyPathEnvVar, path)
	if _, err := loadOrGenerateSigningKey(newLogger(io.Discard)); err == nil {
		t.Fatal("expected PEM decode error")
	}
}

// TestLoadSigningKey_BadKeyTypeErrors covers non-Ed25519 key material.
func TestLoadSigningKey_BadKeyTypeErrors(t *testing.T) {
	// Create a valid PEM that decodes but isn't PKCS8 / Ed25519.
	pemBytes := pemEncodeForTest("PRIVATE KEY", []byte{0x00, 0x01, 0x02})
	path := filepath.Join(t.TempDir(), "bad.pem")
	if err := writeFile(path, pemBytes); err != nil {
		t.Fatal(err)
	}
	t.Setenv(jwtKeyPathEnvVar, path)
	if _, err := loadOrGenerateSigningKey(newLogger(io.Discard)); err == nil {
		t.Fatal("expected pkcs8 parse error")
	}
}
