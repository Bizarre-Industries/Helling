package cmd_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/cmd"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

func useTempConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

func seedProfile(t *testing.T, prof config.Profile) {
	t.Helper()
	path, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&prof, path); err != nil {
		t.Fatal(err)
	}
}

func fakeHellingd(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for p, h := range routes {
		mux.HandleFunc(p, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func synthAccessJWT(username, role string, exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{
		"username": username, "role": role, "exp": exp,
	})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	return header + "." + payload + "." + sig
}

func runCmd(t *testing.T, args []string, stdin string) (stdoutS string, err error) {
	t.Helper()
	root := cmd.NewAuthCmd()
	root.PersistentFlags().String("api", "", "")
	root.SetArgs(args)
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetIn(strings.NewReader(stdin))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = root.ExecuteContext(ctx)
	return outBuf.String() + errBuf.String(), err
}

func TestAuthLogin_StoresTokensAndPrintsConfirmation(t *testing.T) {
	useTempConfigDir(t)
	srv := fakeHellingd(t, map[string]http.HandlerFunc{
		"/api/v1/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "helling_refresh", Value: "refresh-xyz"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"access_token": "jwt.live", "token_type": "Bearer", "expires_in": 900},
			})
		},
	})

	out, err := runCmd(t,
		[]string{"login", "--username", "admin", "--password", "fixture", "--api", srv.URL},
		"",
	)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !strings.Contains(out, "Logged in as admin") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	path, _ := config.Path()
	prof, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if prof.AccessToken != "jwt.live" || prof.RefreshCookie != "helling_refresh=refresh-xyz" {
		t.Fatalf("profile not persisted: %+v", prof)
	}
}

func TestAuthLogin_MFAFlow(t *testing.T) {
	useTempConfigDir(t)
	srv := fakeHellingd(t, map[string]http.HandlerFunc{
		"/api/v1/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"mfa_required": true, "mfa_token": "mfa-challenge-1"},
			})
		},
		"/api/v1/auth/mfa/complete": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["mfa_token"] != "mfa-challenge-1" || body["totp_code"] != "123456" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "helling_refresh", Value: "mfa-refresh"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"access_token": "jwt.mfa"},
			})
		},
	})

	out, err := runCmd(t,
		[]string{"login", "--username", "admin", "--password", "fixture", "--api", srv.URL},
		"123456\n",
	)
	if err != nil {
		t.Fatalf("login: %v body=%s", err, out)
	}
	path, _ := config.Path()
	prof, _ := config.Load(path)
	if prof.AccessToken != "jwt.mfa" {
		t.Fatalf("mfa token not stored: %+v", prof)
	}
}

func TestAuthWhoami_DecodesStoredJWT(t *testing.T) {
	useTempConfigDir(t)
	exp := time.Now().Add(15 * time.Minute).Unix()
	seedProfile(t, config.Profile{
		API:         "http://127.0.0.1:8080",
		AccessToken: synthAccessJWT("admin", "admin", exp),
	})
	out, err := runCmd(t, []string{"whoami"}, "")
	if err != nil {
		t.Fatalf("whoami: %v", err)
	}
	if !strings.Contains(out, "user:    admin") || !strings.Contains(out, "role:    admin") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestAuthWhoami_NotLoggedIn(t *testing.T) {
	useTempConfigDir(t)
	_, err := runCmd(t, []string{"whoami"}, "")
	if err == nil {
		t.Fatal("expected not-logged-in error")
	}
}

func TestAuthLogout_ClearsProfile(t *testing.T) {
	useTempConfigDir(t)
	srv := fakeHellingd(t, map[string]http.HandlerFunc{
		"/api/v1/auth/logout": func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "meta": map[string]any{}})
		},
	})
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x", RefreshCookie: "helling_refresh=a"})
	out, err := runCmd(t, []string{"logout"}, "")
	if err != nil {
		t.Fatalf("logout: %v out=%q", err, out)
	}
	path, _ := config.Path()
	prof, _ := config.Load(path)
	if prof.AccessToken != "" || prof.RefreshCookie != "" {
		t.Fatalf("creds not cleared: %+v", prof)
	}
}

func TestConfigPath_TempDirIsolation(t *testing.T) {
	useTempConfigDir(t)
	got, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "config.yaml" {
		t.Errorf("unexpected path: %s", got)
	}
}

// TestAuthLogin_PromptsForMissingFields exercises the interactive prompt
// branch for both username and password.
func TestAuthLogin_PromptsForMissingFields(t *testing.T) {
	useTempConfigDir(t)
	srv := fakeHellingd(t, map[string]http.HandlerFunc{
		"/api/v1/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"access_token": "jwt.prompt"},
			})
		},
	})

	_, err := runCmd(t,
		[]string{"login", "--api", srv.URL},
		"admin\npromptpw1234\n",
	)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	path, _ := config.Path()
	prof, _ := config.Load(path)
	if prof.AccessToken != "jwt.prompt" {
		t.Fatalf("token not stored: %+v", prof)
	}
}

// TestAuthLogin_MissingAPIFails confirms the --api requirement on first login.
func TestAuthLogin_MissingAPIFails(t *testing.T) {
	useTempConfigDir(t)
	_, err := runCmd(t, []string{"login", "--username", "u", "--password", "p"}, "")
	if err == nil {
		t.Fatal("expected missing-api error")
	}
}

// TestAuthLogout_NoProfileIsNoop exercises the already-logged-out path.
func TestAuthLogout_NoProfileIsNoop(t *testing.T) {
	useTempConfigDir(t)
	out, err := runCmd(t, []string{"logout"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Already logged out") {
		t.Fatalf("unexpected output: %q", out)
	}
}

// TestAuthWhoami_APITokenBranch exercises the operator-token branch of whoami.
func TestAuthWhoami_APITokenBranch(t *testing.T) {
	useTempConfigDir(t)
	seedProfile(t, config.Profile{API: "http://x", Token: "helling_opx"})
	out, err := runCmd(t, []string{"whoami"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "API token in use") {
		t.Fatalf("unexpected output: %q", out)
	}
}
