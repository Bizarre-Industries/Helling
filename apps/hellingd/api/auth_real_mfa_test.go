package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

// setupAdmin drives /api/v1/auth/setup and returns the access token.
func setupAdmin(t *testing.T, srv *httptest.Server, username, password string) string {
	t.Helper()
	resp := postJSON(t, srv, "/api/v1/auth/setup", map[string]string{"username": username, "password": password}, "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup: %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	readJSON(t, resp, &body)
	return body.Data.AccessToken
}

func bearerGet(t *testing.T, srv *httptest.Server, path, access string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func bearerPost(t *testing.T, srv *httptest.Server, path, access string, body any) *http.Response {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+path, bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func bearerDelete(t *testing.T, srv *httptest.Server, path, access string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, srv.URL+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustReadBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestRealTotp_EnrollVerifyLoginFlow(t *testing.T) {
	srv := spinUp(t)

	access := setupAdmin(t, srv, "admin", "supersecret12345")

	enrollResp := bearerPost(t, srv, "/api/v1/auth/totp/setup", access, struct{}{})
	defer func() { _ = enrollResp.Body.Close() }()
	if enrollResp.StatusCode != http.StatusOK {
		t.Fatalf("totp setup: %d", enrollResp.StatusCode)
	}
	var enroll struct {
		Data struct {
			Secret          string   `json:"secret"`
			ProvisioningURI string   `json:"provisioning_uri"`
			RecoveryCodes   []string `json:"recovery_codes"`
		} `json:"data"`
	}
	readJSON(t, enrollResp, &enroll)
	if enroll.Data.Secret == "" || len(enroll.Data.RecoveryCodes) != auth.RecoveryCodeCount {
		t.Fatalf("bad enroll payload: %+v", enroll.Data)
	}

	code, err := auth.GenerateTOTPCode(enroll.Data.Secret)
	if err != nil {
		t.Fatal(err)
	}

	verifyResp := bearerPost(t, srv, "/api/v1/auth/totp/verify", access, map[string]string{"totp_code": code})
	defer func() { _ = verifyResp.Body.Close() }()
	if verifyResp.StatusCode != http.StatusOK {
		t.Fatalf("totp verify: %d", verifyResp.StatusCode)
	}

	loginResp := postJSON(t, srv, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "supersecret12345"}, "")
	defer func() { _ = loginResp.Body.Close() }()
	if loginResp.StatusCode != http.StatusAccepted {
		t.Fatalf("login should 202, got %d body=%s", loginResp.StatusCode, mustReadBody(loginResp))
	}
	var mfa struct {
		Data struct {
			MFARequired bool   `json:"mfa_required"`
			MFAToken    string `json:"mfa_token"`
		} `json:"data"`
	}
	readJSON(t, loginResp, &mfa)
	if !mfa.Data.MFARequired || mfa.Data.MFAToken == "" {
		t.Fatalf("missing MFA challenge: %+v", mfa.Data)
	}

	code2, _ := auth.GenerateTOTPCode(enroll.Data.Secret)
	mfaResp := postJSON(t, srv, "/api/v1/auth/mfa/complete", map[string]string{
		"mfa_token": mfa.Data.MFAToken,
		"totp_code": code2,
	}, "")
	defer func() { _ = mfaResp.Body.Close() }()
	if mfaResp.StatusCode != http.StatusOK {
		t.Fatalf("mfa complete: %d body=%s", mfaResp.StatusCode, mustReadBody(mfaResp))
	}

	// Recovery code path.
	loginResp2 := postJSON(t, srv, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "supersecret12345"}, "")
	defer func() { _ = loginResp2.Body.Close() }()
	var mfa2 struct {
		Data struct {
			MFAToken string `json:"mfa_token"`
		} `json:"data"`
	}
	readJSON(t, loginResp2, &mfa2)

	recResp := postJSON(t, srv, "/api/v1/auth/mfa/complete", map[string]string{
		"mfa_token": mfa2.Data.MFAToken,
		"totp_code": enroll.Data.RecoveryCodes[0],
	}, "")
	defer func() { _ = recResp.Body.Close() }()
	if recResp.StatusCode != http.StatusOK {
		t.Fatalf("recovery code flow: %d", recResp.StatusCode)
	}
}

func TestRealApiTokens_CreateListRevoke(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "supersecret12345")

	createResp := bearerPost(t, srv, "/api/v1/auth/tokens", access, map[string]string{
		"name":  "ci-bot",
		"scope": "write",
	})
	defer func() { _ = createResp.Body.Close() }()
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d body=%s", createResp.StatusCode, mustReadBody(createResp))
	}
	var created struct {
		Data struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		} `json:"data"`
	}
	readJSON(t, createResp, &created)
	if !strings.HasPrefix(created.Data.Token, "helling_") {
		t.Fatalf("plaintext token should start with helling_: %q", created.Data.Token)
	}

	listResp := bearerGet(t, srv, "/api/v1/auth/tokens", created.Data.Token)
	defer func() { _ = listResp.Body.Close() }()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list via api token: %d body=%s", listResp.StatusCode, mustReadBody(listResp))
	}

	revResp := bearerDelete(t, srv, "/api/v1/auth/tokens/"+created.Data.ID, access)
	defer func() { _ = revResp.Body.Close() }()
	if revResp.StatusCode != http.StatusOK {
		t.Fatalf("revoke: %d", revResp.StatusCode)
	}

	listAfter := bearerGet(t, srv, "/api/v1/auth/tokens", created.Data.Token)
	defer func() { _ = listAfter.Body.Close() }()
	if listAfter.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked token should 401, got %d", listAfter.StatusCode)
	}
}

func TestRealTokens_RejectsBadScope(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "supersecret12345")

	resp := bearerPost(t, srv, "/api/v1/auth/tokens", access, map[string]string{
		"name":  "bad",
		"scope": "owner",
	})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400/422 for bad scope, got %d", resp.StatusCode)
	}
}
