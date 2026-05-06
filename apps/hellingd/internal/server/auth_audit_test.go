package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/audit"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
)

type auditRecorder struct {
	mu      sync.Mutex
	records []audit.Record
}

func (r *auditRecorder) emit(_ *slog.Logger, rec *audit.Record) {
	if rec == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, *rec)
}

func (r *auditRecorder) requireSuccess(t *testing.T, action, actor string) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.records {
		rec := &r.records[i]
		if rec.Action == action && rec.Outcome == outcomeSuccess && (actor == "" || rec.Actor == actor) {
			return
		}
	}
	t.Fatalf("missing successful audit action=%q actor=%q in %#v", action, actor, r.records)
}

func (r *auditRecorder) find(action, outcome string) (audit.Record, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.records {
		rec := r.records[i]
		if rec.Action == action && rec.Outcome == outcome {
			return rec, true
		}
	}
	return audit.Record{}, false
}

func newAuditedTestServer(t *testing.T) (*Server, *auditRecorder) {
	t.Helper()
	rec := &auditRecorder{}
	srv, _ := newTestServerWithConfig(t, func(cfg *Config) {
		cfg.AuditEmit = rec.emit
	})
	return srv, rec
}

func TestAuthLoginLogoutEmitAuditRecords(t *testing.T) {
	srv, st := newTestServer(t)
	rec := &auditRecorder{}
	srv.cfg.AuditEmit = rec.emit
	seedUser(t, st, "audit-login", testPassword)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cookie := loginCookie(t, ts, "audit-login", testPassword)
	rec.requireSuccess(t, "auth.login", "audit-login")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/auth/logout", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST logout: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status: got %d want 204", resp.StatusCode)
	}
	rec.requireSuccess(t, "auth.logout", "audit-login")
}

func TestAuthFailedLoginEmitsAuditRecord(t *testing.T) {
	srv, rec := newAuditedTestServer(t)
	seedUser(t, srv.cfg.Store, "audit-fail", testPassword)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/login", loginRequest{
		Username: "audit-fail",
		Password: "wrong-password",
	}, nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login status: got %d want 401", resp.StatusCode)
	}
	got, ok := rec.find("auth.login", outcomeFailure)
	if !ok {
		t.Fatalf("missing auth.login failure audit in %#v", rec.records)
	}
	if got.Actor != anonymousAuditActor || got.ActorID != anonymousAuditActor || got.ActorRole != anonymousAuditRole {
		t.Fatalf("pre-auth actor fields: got actor=%q id=%q role=%q", got.Actor, got.ActorID, got.ActorRole)
	}
}

func TestAdminPolicyDenyEmitsAuditReason(t *testing.T) {
	srv, rec := newAuditedTestServer(t)
	seedRegularUser(t, srv.cfg.Store, "audit-deny")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "audit-deny", testPassword)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/users", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET users: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("admin deny status: got %d want 403", resp.StatusCode)
	}
	got, ok := rec.find("policy.deny", outcomeDenied)
	if !ok {
		t.Fatalf("missing policy.deny audit in %#v", rec.records)
	}
	if got.Actor != "audit-deny" || got.ActorRole != "user" {
		t.Fatalf("policy actor fields: got actor=%q role=%q", got.Actor, got.ActorRole)
	}
	if got.TargetType != "admin" || got.TargetID != "/v1/users" || got.PolicyReason != "rbac.insufficient_role" {
		t.Fatalf("policy reason fields: got target_type=%q target_id=%q reason=%q", got.TargetType, got.TargetID, got.PolicyReason)
	}
}

func TestIncusProxyMutationEmitsAuditRecord(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	rec := &auditRecorder{}
	srv.cfg.AuditEmit = rec.emit
	seedAdminUser(t, st)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "admin", testPassword)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPut, ts.URL+"/api/incus/1.0/instances/demo/state", strings.NewReader("{}"))
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("PUT /api/incus: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("proxy status: got %d want 503", resp.StatusCode)
	}
	got, ok := rec.find("incus.proxy", outcomeFailure)
	if !ok {
		t.Fatalf("missing incus.proxy failure audit in %#v", rec.records)
	}
	if got.Actor != "admin" || got.TargetID != "/api/incus/1.0/instances/demo/state" || got.RequestPath != got.TargetID || got.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("incus proxy audit = %#v", got)
	}
}

func TestAuthTokenLifecycleEmitsAuditRecords(t *testing.T) {
	srv, rec := newAuditedTestServer(t)
	seedUser(t, srv.cfg.Store, "audit-token", testPassword)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "audit-token", testPassword)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/tokens", map[string]string{
		"name":   "cli",
		"scopes": "read",
	}, cookie)
	var created createTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated || created.ID == "" {
		t.Fatalf("token status=%d body=%+v", resp.StatusCode, created)
	}
	rec.requireSuccess(t, "auth.token.create", "audit-token")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodDelete, ts.URL+"/v1/auth/tokens/"+created.ID, http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE token: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke status: got %d want 204", resp.StatusCode)
	}
	rec.requireSuccess(t, "auth.token.revoke", "audit-token")
}

func TestAuthTOTPLifecycleEmitsAuditRecords(t *testing.T) {
	srv, rec := newAuditedTestServer(t)
	seedUser(t, srv.cfg.Store, "audit-totp", testPassword)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "audit-totp", testPassword)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/totp/setup", map[string]string{}, cookie)
	var setup totpSetupResponse
	if err := json.NewDecoder(resp.Body).Decode(&setup); err != nil {
		t.Fatalf("decode totp setup: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || setup.Secret == "" {
		t.Fatalf("totp setup status=%d body=%+v", resp.StatusCode, setup)
	}
	rec.requireSuccess(t, "auth.totp.setup", "audit-totp")

	resp = postJSON(t, ts.Client(), ts.URL+"/v1/auth/totp/verify", totpVerifyRequest{
		Code: totpCodeForTest(t, setup.Secret, time.Now().UTC()),
	}, cookie)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("totp verify status: got %d want 204", resp.StatusCode)
	}
	rec.requireSuccess(t, "auth.totp.enable", "audit-totp")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodDelete, ts.URL+"/v1/auth/totp", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE totp: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("totp delete status: got %d want 204", resp.StatusCode)
	}
	rec.requireSuccess(t, "auth.totp.delete", "audit-totp")
}

func TestAuthMFACompletionEmitsAuditRecords(t *testing.T) {
	srv, rec := newAuditedTestServer(t)
	u := seedRegularUser(t, srv.cfg.Store, "audit-mfa")
	if err := srv.cfg.Store.SetTOTPSecret(t.Context(), u.ID, "JBSWY3DPEHPK3PXP", true); err != nil {
		t.Fatalf("SetTOTPSecret: %v", err)
	}
	recoveryCode := "ABCD-EFGH-IJKL-MNOP"
	recoveryHash, err := auth.Hash(recoveryCode, auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32})
	if err != nil {
		t.Fatalf("hash recovery code: %v", err)
	}
	if err := srv.cfg.Store.SaveRecoveryCodes(t.Context(), u.ID, []string{recoveryHash}); err != nil {
		t.Fatalf("SaveRecoveryCodes: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/auth/login", loginRequest{
		Username: "audit-mfa",
		Password: testPassword,
	}, nil)
	var challenge struct {
		Data struct {
			MFAToken string `json:"mfa_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted || challenge.Data.MFAToken == "" {
		t.Fatalf("challenge status=%d body=%+v", resp.StatusCode, challenge.Data)
	}
	rec.requireSuccess(t, "auth.mfa.challenge", "audit-mfa")

	resp = postJSON(t, ts.Client(), ts.URL+"/v1/auth/mfa/complete", map[string]string{
		"mfa_token":     challenge.Data.MFAToken,
		"recovery_code": recoveryCode,
	}, nil)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("mfa complete status: got %d want 204", resp.StatusCode)
	}
	rec.requireSuccess(t, "auth.mfa.complete", "audit-mfa")
	rec.requireSuccess(t, "auth.login", "audit-mfa")
}

func totpCodeForTest(t *testing.T, secret string, now time.Time) string {
	t.Helper()
	normalized := strings.ToUpper(secret)
	for len(normalized)%8 != 0 {
		normalized += "="
	}
	key, err := base32.StdEncoding.DecodeString(normalized)
	if err != nil {
		t.Fatalf("decode TOTP secret: %v", err)
	}
	counter := uint64(now.Unix() / 30)
	mac := hmac.New(sha1.New, key)
	if err := binary.Write(mac, binary.BigEndian, counter); err != nil {
		t.Fatalf("write TOTP counter: %v", err)
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", bin%1_000_000)
}
