package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/config"
)

// totpSecretFixture is the RFC 6238 §A.1 test vector, used only as a
// placeholder in CLI contract tests. gitleaks:allow
const totpSecretFixture = "JBSWY3DPEHPK3PXP" // gitleaks:allow

func TestAuthMfaSetup_POSTsEmptyBody(t *testing.T) {
	useTempConfigDir(t)
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"secret":"` + totpSecretFixture + `","otpauth_uri":"otpauth://totp/Helling:alice?secret=` + totpSecretFixture + `"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runAuthTop(t, []string{"mfa", "setup"})
	if err != nil {
		t.Fatalf("setup: %v out=%q", err, out)
	}
	if method != http.MethodPost || path != "/api/v1/auth/totp/setup" {
		t.Fatalf("method=%s path=%s", method, path)
	}
	if !strings.Contains(out, "otpauth://totp") {
		t.Fatalf("out: %q", out)
	}
}

func TestAuthMfaVerify_SendsCode(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runAuthTop(t, []string{"mfa", "verify", "123456"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["totp_code"] != "123456" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestAuthMfaDisable_RequiresPassword(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runAuthTop(t, []string{"mfa", "disable", "--password=fixture-pw"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["password"] != "fixture-pw" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestAuthMfaDisable_MissingPasswordFails(t *testing.T) {
	useTempConfigDir(t)
	seedProfile(t, config.Profile{API: "http://example.test", AccessToken: "jwt.x"})
	if _, err := runAuthTop(t, []string{"mfa", "disable"}); err == nil {
		t.Fatal("expected required-flag error")
	}
}
