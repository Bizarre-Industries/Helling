package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hellingapi "github.com/Bizarre-Industries/Helling/apps/hellingd/api"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
	httpserver "github.com/Bizarre-Industries/Helling/apps/hellingd/internal/http"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// spinUp boots a realistic in-memory Helling API with the cookie-based real
// auth handlers wired in, returning an httptest.Server ready for use.
func spinUp(t *testing.T) *httptest.Server {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "real.db") + "?cache=shared"
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	_, priv, err := auth.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := auth.NewSigner(priv, "hellingd-itest", 15*time.Minute, 7*24*time.Hour, 30*time.Minute)
	svc := auth.NewService(authrepo.New(pool), signer, auth.Argon2idParams{})

	mux := httpserver.NewMuxWith(hellingapi.Deps{Auth: svc})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body any, cookie string) *http.Response {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+path, bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	return resp
}

func readJSON(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func discardBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

const refreshCookieNameForTest = "helling_refresh"

// extractRefreshCookie pulls the Set-Cookie value for helling_refresh from a
// response, or "" when absent.
func extractRefreshCookie(resp *http.Response) string {
	for _, c := range resp.Cookies() {
		if c.Name == refreshCookieNameForTest {
			return c.Name + "=" + c.Value
		}
	}
	return ""
}

func TestRealAuth_SetupLoginRefreshLogout(t *testing.T) {
	srv := spinUp(t)

	setupResp := postJSON(t, srv, "/api/v1/auth/setup", map[string]string{"username": "admin", "password": "supersecret12345"}, "")
	defer func() { _ = setupResp.Body.Close() }()
	if setupResp.StatusCode != http.StatusOK {
		t.Fatalf("setup status: %d", setupResp.StatusCode)
	}
	var setupBody struct {
		Data struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
	}
	readJSON(t, setupResp, &setupBody)
	if setupBody.Data.AccessToken == "" || setupBody.Data.TokenType != "Bearer" {
		t.Fatalf("setup missing access token: %+v", setupBody.Data)
	}
	cookie := extractRefreshCookie(setupResp)
	if cookie == "" || !strings.Contains(cookie, "helling_refresh=") {
		t.Fatalf("refresh cookie not set")
	}

	dup := postJSON(t, srv, "/api/v1/auth/setup", map[string]string{"username": "x", "password": "aaaaaaaaaaaa"}, "")
	if dup.StatusCode != http.StatusConflict {
		t.Fatalf("second setup should 409, got %d", dup.StatusCode)
	}
	discardBody(dup)

	loginResp := postJSON(t, srv, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "supersecret12345"}, "")
	defer func() { _ = loginResp.Body.Close() }()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status: %d", loginResp.StatusCode)
	}
	var loginBody struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	readJSON(t, loginResp, &loginBody)
	if loginBody.Data.AccessToken == "" {
		t.Fatal("login missing access token")
	}
	loginCookie := extractRefreshCookie(loginResp)
	if loginCookie == "" {
		t.Fatal("login cookie missing")
	}

	wrong := postJSON(t, srv, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "nope"}, "")
	if wrong.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong login should 401, got %d", wrong.StatusCode)
	}
	discardBody(wrong)

	refreshResp := postJSON(t, srv, "/api/v1/auth/refresh", map[string]string{}, loginCookie)
	defer func() { _ = refreshResp.Body.Close() }()
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status: %d", refreshResp.StatusCode)
	}
	var refreshBody struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	readJSON(t, refreshResp, &refreshBody)
	if refreshBody.Data.AccessToken == "" {
		t.Fatal("refresh missing access token")
	}

	logout := postJSON(t, srv, "/api/v1/auth/logout", struct{}{}, loginCookie)
	if logout.StatusCode != http.StatusOK {
		t.Fatalf("logout status: %d", logout.StatusCode)
	}
	discardBody(logout)

	stale := postJSON(t, srv, "/api/v1/auth/refresh", map[string]string{}, loginCookie)
	if stale.StatusCode != http.StatusUnauthorized {
		t.Fatalf("stale refresh should 401, got %d", stale.StatusCode)
	}
	discardBody(stale)
}

func TestRealAuth_RefreshBodyTokenAlsoWorks(t *testing.T) {
	srv := spinUp(t)

	resp := postJSON(t, srv, "/api/v1/auth/setup", map[string]string{"username": "admin", "password": "anothersecret12"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup: %d", resp.StatusCode)
	}
	cookie := extractRefreshCookie(resp)
	rawRefresh := strings.TrimPrefix(cookie, "helling_refresh=")
	discardBody(resp)

	resp = postJSON(t, srv, "/api/v1/auth/refresh", map[string]string{"refresh_token": rawRefresh}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("body-token refresh: %d", resp.StatusCode)
	}
	discardBody(resp)
}
