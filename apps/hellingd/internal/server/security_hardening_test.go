package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/proxy"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

const testPassword = "secret-password-123"

func seedRegularUser(t *testing.T, st *store.Store, username string) store.User {
	t.Helper()
	hash, err := auth.Hash(testPassword, auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32})
	if err != nil {
		t.Fatalf("auth.Hash: %v", err)
	}
	u, err := st.CreateUser(context.Background(), username, hash, false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

func postJSON(t *testing.T, client *http.Client, url string, body any, cookie *http.Cookie) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestTOTPLoginRequiresChallengeBeforeSession(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	u := seedRegularUser(t, st, "mfa-user")
	if err := st.SetTOTPSecret(t.Context(), u.ID, "JBSWY3DPEHPK3PXP", true); err != nil {
		t.Fatalf("SetTOTPSecret: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/login", loginRequest{
		Username: "mfa-user",
		Password: testPassword,
	}, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login status: got %d want 202 body=%s", resp.StatusCode, string(body))
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == CookieName {
			t.Fatalf("password-only MFA login issued a %s cookie", CookieName)
		}
	}
	var body struct {
		Data struct {
			MFARequired bool   `json:"mfa_required"`
			MFAToken    string `json:"mfa_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode challenge response: %v", err)
	}
	if !body.Data.MFARequired || body.Data.MFAToken == "" {
		t.Fatalf("challenge response missing MFA token: %+v", body.Data)
	}
}

func TestMFACompleteConsumesRecoveryCodeAndIssuesSession(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	u := seedRegularUser(t, st, "recovery-user")
	if err := st.SetTOTPSecret(t.Context(), u.ID, "JBSWY3DPEHPK3PXP", true); err != nil {
		t.Fatalf("SetTOTPSecret: %v", err)
	}
	recoveryCode := "ABCD-EFGH-IJKL-MNOP"
	recoveryHash, err := auth.Hash(recoveryCode, auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32})
	if err != nil {
		t.Fatalf("hash recovery code: %v", err)
	}
	if err := st.SaveRecoveryCodes(t.Context(), u.ID, []string{recoveryHash}); err != nil {
		t.Fatalf("SaveRecoveryCodes: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/login", loginRequest{
		Username: "recovery-user",
		Password: testPassword,
	}, nil)
	var loginResp struct {
		Data struct {
			MFAToken string `json:"mfa_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login challenge: %v", err)
	}
	_ = resp.Body.Close()
	if loginResp.Data.MFAToken == "" {
		t.Fatal("missing MFA token")
	}

	resp = postJSON(t, ts.Client(), ts.URL+"/v1/auth/mfa/complete", map[string]string{
		"mfa_token":     loginResp.Data.MFAToken,
		"recovery_code": recoveryCode,
	}, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("mfa complete status: got %d want 204 body=%s", resp.StatusCode, string(body))
	}
	var sessionCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == CookieName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("MFA completion did not issue %s", CookieName)
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/me", http.NoBody)
	req.AddCookie(sessionCookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/auth/me: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth/me status after MFA: got %d want 200", resp.StatusCode)
	}
}

func TestAPITokenBearerAuthAndRevoke(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	seedRegularUser(t, st, "token-user")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "token-user", testPassword)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/tokens", map[string]string{
		"name":   "cli",
		"scopes": "read",
	}, cookie)
	var created createTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated || created.Token == "" || created.ID == "" {
		t.Fatalf("token response status=%d body=%+v", resp.StatusCode, created)
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+created.Token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /auth/me with token: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bearer auth status: got %d want 200", resp.StatusCode)
	}

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodDelete, ts.URL+"/v1/auth/tokens/"+created.ID, http.NoBody)
	req.AddCookie(cookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE token: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke status: got %d want 204", resp.StatusCode)
	}

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/auth/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+created.Token)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /auth/me after revoke: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-revoke bearer status: got %d want 401", resp.StatusCode)
	}
}

func TestAPIV1LoginReturnsJWTBearer(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	seedRegularUser(t, st, "jwt-user")
	signer, err := auth.NewJWTSigner()
	if err != nil {
		t.Fatalf("NewJWTSigner: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := New(&Config{
		Store:   st,
		Logger:  logger,
		Version: VersionInfo{Version: "test"},
		Auth: AuthSettings{
			SessionTTL:     time.Hour,
			AccessTTL:      15 * time.Minute,
			UsernameLimit:  5,
			UsernameWindow: time.Minute,
			IPLimit:        20,
			IPWindow:       time.Minute,
			Argon2:         auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32},
			JWTSigner:      signer,
		},
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/api/v1/auth/login", loginRequest{
		Username: "jwt-user",
		Password: testPassword,
	}, nil)
	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			MFARequired bool   `json:"mfa_required"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || loginResp.Data.AccessToken == "" || loginResp.Data.TokenType != "Bearer" || loginResp.Data.MFARequired {
		t.Fatalf("unexpected login response status=%d body=%+v", resp.StatusCode, loginResp.Data)
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/api/v1/auth/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+loginResp.Data.AccessToken)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/auth/me: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("bearer JWT status: got %d want 200 body=%s", resp.StatusCode, string(body))
	}
}

func TestNonAdminCannotManageUsersOrDeferredSurfaces(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	seedRegularUser(t, st, "basic")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "basic", testPassword)

	checks := []struct {
		name string
		req  *http.Request
	}{
		{
			name: "create user",
			req:  newJSONRequest(t, http.MethodPost, ts.URL+"/v1/users", map[string]any{"username": "evil", "password": testPassword, "is_admin": true}),
		},
		{
			name: "system config",
			req:  newJSONRequest(t, http.MethodPut, ts.URL+"/v1/system/config", map[string]any{"auth": map[string]any{}}),
		},
		{
			name: "firewall",
			req:  newJSONRequest(t, http.MethodPost, ts.URL+"/v1/firewall/host", map[string]any{"name": "allow", "action": "allow"}),
		},
		{
			name: "webhook",
			req:  newJSONRequest(t, http.MethodPost, ts.URL+"/v1/webhooks", map[string]any{"name": "hook", "url": "https://example.test/hook"}),
		},
		{
			name: "schedule",
			req:  newJSONRequest(t, http.MethodPost, ts.URL+"/v1/schedules", map[string]any{"name": "nightly", "kind": "backup", "target": "web-1", "cron_expr": "0 0 * * *"}),
		},
		{
			name: "bmc",
			req:  newJSONRequest(t, http.MethodPost, ts.URL+"/v1/bmc", map[string]any{"name": "ipmi", "address": "192.0.2.10", "username": "root", "password": testPassword}),
		},
		{
			name: "kubernetes",
			req:  newJSONRequest(t, http.MethodPost, ts.URL+"/v1/kubernetes", map[string]any{"name": "cluster", "version": "v1.30.0"}),
		},
	}
	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.req.AddCookie(cookie)
			resp, err := ts.Client().Do(tt.req)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusForbidden {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("%s status: got %d want 403 body=%s", tt.name, resp.StatusCode, string(body))
			}
		})
	}
}

func TestNonAdminCannotUseRawProxy(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	seedRegularUser(t, st, "proxy-user")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
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
			Argon2:         auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32},
		},
		IncusProxy: proxy.NewIncusProxy("/tmp/helling-test-missing-incus.sock", logger),
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "proxy-user", testPassword)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/api/incus/1.0", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/incus/1.0: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("proxy status: got %d want 403 body=%s", resp.StatusCode, string(body))
	}
}

func newJSONRequest(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
