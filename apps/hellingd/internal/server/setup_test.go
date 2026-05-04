package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

const setupTokenFixture = "0123456789abcdef0123456789abcdef"

func TestSetupRequiresInstallerToken(t *testing.T) {
	t.Parallel()
	srv, _, _ := newSetupTokenServer(t, io.Discard)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/setup", setupRequest{
		Username: "admin",
		Password: testPassword,
	}, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("setup without token status: got %d want 400 body=%s", resp.StatusCode, string(body))
	}
}

func TestSetupCreatesAdminWithInstallerTokenAndLogsAuditEvent(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	srv, st, tokenPath := newSetupTokenServer(t, &logs)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/setup", setupRequest{
		Username:   "admin",
		Password:   testPassword,
		SetupToken: setupTokenFixture,
	}, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("setup with token status: got %d want 201 body=%s", resp.StatusCode, string(body))
	}
	u, err := st.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetUserByUsername(admin): %v", err)
	}
	if !u.IsAdmin {
		t.Fatal("setup user is not admin")
	}
	logText := logs.String()
	for _, want := range []string{"setup_admin_created", "username=admin", "source_ip="} {
		if !strings.Contains(logText, want) {
			t.Fatalf("setup audit log %q missing %q", logText, want)
		}
	}
	if _, err := os.Stat(tokenPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("setup token file after success: err=%v, want not exist", err)
	}
}

func TestSetupStatusReflectsUserCount(t *testing.T) {
	t.Parallel()
	srv, _, _ := newSetupTokenServer(t, io.Discard)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/setup/status", http.NoBody)
	if err != nil {
		t.Fatalf("new setup status request: %v", err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET setup status: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var before setupStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&before); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if !before.SetupRequired {
		t.Fatal("setup status before first user = false, want true")
	}

	createResp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/setup", setupRequest{
		Username:   "admin",
		Password:   testPassword,
		SetupToken: setupTokenFixture,
	}, nil)
	_ = createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("setup create status = %d, want 201", createResp.StatusCode)
	}

	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/setup/status", http.NoBody)
	if err != nil {
		t.Fatalf("new setup status request after create: %v", err)
	}
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET setup status after create: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var after setupStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&after); err != nil {
		t.Fatalf("decode setup status after create: %v", err)
	}
	if after.SetupRequired {
		t.Fatal("setup status after first user = true, want false")
	}
}

func TestRetireSetupTokenTruncatesWhenDirectoryCannotUnlink(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root can unlink from non-writable test directories")
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "setup-token")
	if err := os.WriteFile(tokenPath, []byte(setupTokenFixture+"\n"), 0o600); err != nil {
		t.Fatalf("write setup token: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod setup token dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	svc := newFirstAdminSetupService(nil, auth.Argon2Params{}, tokenPath)
	if err := svc.RetireSetupToken(); err != nil {
		t.Fatalf("retireSetupToken: %v", err)
	}
	raw, err := os.ReadFile(tokenPath) // #nosec G304 -- path is a test fixture under t.TempDir.
	if err != nil {
		t.Fatalf("read truncated setup token: %v", err)
	}
	if len(raw) != 0 {
		t.Fatalf("setup token length after fallback = %d, want 0", len(raw))
	}
}

func TestSetupRejectsOversizedBody(t *testing.T) {
	t.Parallel()
	srv, _, _ := newSetupTokenServer(t, io.Discard)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		ts.URL+"/v1/auth/setup",
		strings.NewReader(`{"username":"admin","password":"`+strings.Repeat("x", 2048)+`","setup_token":"`+setupTokenFixture+`"}`),
	)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST oversized setup: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("oversized setup status: got %d want 400 body=%s", resp.StatusCode, string(body))
	}
}

func newSetupTokenServer(t *testing.T, logOut io.Writer) (*Server, *store.Store, string) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	tokenPath := filepath.Join(t.TempDir(), "setup-token")
	if err := os.WriteFile(tokenPath, []byte(setupTokenFixture+"\n"), 0o600); err != nil {
		t.Fatalf("write setup token: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(logOut, nil))
	srv, err := New(&Config{
		Store:   st,
		Logger:  logger,
		Version: VersionInfo{Version: "test"},
		Auth: AuthSettings{
			SessionTTL:     time.Hour,
			UsernameLimit:  5,
			UsernameWindow: time.Minute,
			IPLimit:        20,
			IPWindow:       time.Minute,
			SetupTokenPath: tokenPath,
			Argon2:         auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32},
		},
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return srv, st, tokenPath
}
