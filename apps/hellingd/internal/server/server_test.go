package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

func newTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := New(&Config{
		Store:   st,
		Logger:  logger,
		Version: VersionInfo{Version: "test", Commit: "test", BuildTime: "test"},
		Auth: AuthSettings{
			SessionTTL:     time.Hour,
			UsernameLimit:  5,
			UsernameWindow: time.Minute,
			IPLimit:        20,
			IPWindow:       time.Minute,
			Argon2:         auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32},
		},
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return srv, st
}

func seedUser(t *testing.T, st *store.Store, username, password string) {
	t.Helper()
	hash, err := auth.Hash(password, auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32})
	if err != nil {
		t.Fatalf("auth.Hash: %v", err)
	}
	if _, err := st.CreateUser(context.Background(), username, hash, false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

func loginCookie(t *testing.T, ts *httptest.Server, username, password string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(loginRequest{Username: username, Password: password})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("login POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("login status: got %d body=%s", resp.StatusCode, string(buf))
	}
	for _, c := range resp.Cookies() {
		if c.Name == CookieName {
			return c
		}
	}
	t.Fatal("login: no helling_session cookie set")
	return nil
}

func TestHealthAndVersion(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/healthz", http.NoBody)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status: %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/version", http.NoBody)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/version: %v", err)
	}
	var ver map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&ver); err != nil {
		t.Fatalf("decode version: %v", err)
	}
	_ = resp.Body.Close()
	if ver["version"] != "test" {
		t.Fatalf("version: got %v", ver)
	}
}

func TestLoginLogoutMeFlow(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cookie := loginCookie(t, ts, "alice", "secret-password-123")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/me", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/auth/me: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/v1/auth/me status: %d", resp.StatusCode)
	}
	var me userResponse
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	_ = resp.Body.Close()
	if me.Username != "alice" {
		t.Fatalf("me.Username: got %q want alice", me.Username)
	}

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/logout", http.NoBody)
	req.AddCookie(cookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/auth/logout: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status: %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/me", http.NoBody)
	req.AddCookie(cookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/auth/me post-logout: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout status: got %d want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	seedUser(t, st, "bob", "rightpw")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(loginRequest{Username: "bob", Password: "wrongpw"})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("login POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", resp.StatusCode)
	}
}

func TestMeWithoutCookieReturns401(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/me", http.NoBody)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/auth/me: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", resp.StatusCode)
	}
}

func TestLoginRateLimitTriggers429(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	seedUser(t, st, "carol", "rightpw")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(loginRequest{Username: "carol", Password: "wrongpw"})
	for i := range 5 {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
		_ = resp.Body.Close()
	}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("rate-limit POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status: got %d want 429", resp.StatusCode)
	}
}

func TestLoginRejectsMalformedJSON(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("login POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", resp.StatusCode)
	}
}
