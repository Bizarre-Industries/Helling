package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func testAPI() *http.ServeMux {
	mux := http.NewServeMux()
	api := humago.New(mux, NewConfig())
	RegisterOperations(api)
	EnrichOpenAPI(api.OpenAPI())
	return mux
}

func TestNewConfigProvidesRequiredMetadata(t *testing.T) {
	config := NewConfig()
	if config.Info == nil || config.Info.Description == "" {
		t.Fatal("expected info description in config")
	}
	if len(config.Servers) == 0 {
		t.Fatal("expected at least one server in config")
	}

	tagNames := map[string]bool{}
	for _, tag := range config.Tags {
		tagNames[tag.Name] = true
	}
	for _, expected := range []string{"Auth", "Users", "System"} {
		if !tagNames[expected] {
			t.Fatalf("expected %s tag in config", expected)
		}
	}
}

func TestRegisterOperationsHealthRoute(t *testing.T) {
	mux := testAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("expected status payload in response body, got: %s", body)
	}

	if !strings.Contains(body, `"request_id":"req_huma_spike"`) {
		t.Fatalf("expected request_id in response body, got: %s", body)
	}
}

func TestRegisterOperationsAuthLoginSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"username":"admin","password":"correct-horse-battery-staple"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token_type":"Bearer"`) {
		t.Fatalf("expected bearer token payload, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthLoginMFAChallenge(t *testing.T) {
	mux := testAPI()
	body := `{"username":"mfa","password":"anything"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusAccepted, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"mfa_required":true`) {
		t.Fatalf("expected mfa challenge payload, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthLoginUnauthorized(t *testing.T) {
	mux := testAPI()
	body := `{"username":"wrong","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsAuthLoginRateLimited(t *testing.T) {
	mux := testAPI()
	body := `{"username":"ratelimit","password":"anything"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusTooManyRequests, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsUserListPagination(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"has_next":true`) {
		t.Fatalf("expected has_next=true on first page, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"next_cursor":"cursor_page_2"`) {
		t.Fatalf("expected next cursor on first page, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsUserListSecondPage(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=1&cursor=cursor_page_2", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"has_next":false`) {
		t.Fatalf("expected has_next=false on second page, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthLogoutReturnsEmptyEnvelope(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"request_id":"req_auth_logout"`) {
		t.Fatalf("expected logout request_id in body, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthRefreshSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"refresh_token":"stub_refresh_token_01JZREFRESHABCDEFGHJK"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token_type":"Bearer"`) {
		t.Fatalf("expected bearer token payload, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"request_id":"req_auth_refresh"`) {
		t.Fatalf("expected refresh request_id, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthRefreshInvalidToken(t *testing.T) {
	mux := testAPI()
	body := `{"refresh_token":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsAuthRefreshUnknownToken(t *testing.T) {
	mux := testAPI()
	body := `{"refresh_token":"some-other-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsAuthSetupReturnsTokens(t *testing.T) {
	mux := testAPI()
	body := `{"username":"admin","password":"hunter2hunter"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token_type":"Bearer"`) {
		t.Fatalf("expected bearer token, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthMfaCompleteSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"mfa_token":"mfa_01JZABC0123456789ABCDEFGJK","totp_code":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsAuthMfaCompleteInvalidToken(t *testing.T) {
	mux := testAPI()
	body := `{"mfa_token":"not-a-real-token","totp_code":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRegisterOperationsAuthMfaCompleteBadCode(t *testing.T) {
	mux := testAPI()
	body := `{"mfa_token":"mfa_01JZABC0123456789ABCDEFGJK","totp_code":"000000"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRegisterOperationsAuthTotpSetupReturnsProvisioningURI(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/totp/setup", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "otpauth://totp") {
		t.Fatalf("expected provisioning URI in body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"recovery_codes"`) {
		t.Fatalf("expected recovery codes in body: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthTotpVerifySuccess(t *testing.T) {
	mux := testAPI()
	body := `{"totp_code":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/totp/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsAuthTotpVerifyBadCode(t *testing.T) {
	mux := testAPI()
	body := `{"totp_code":"000000"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/totp/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRegisterOperationsAuthTotpDisableReturnsEmpty(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/totp/disable", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"request_id":"req_auth_totp_disable"`) {
		t.Fatalf("expected disable request_id, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthTokenListPagination(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/tokens?limit=1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"has_next":true`) {
		t.Fatalf("expected has_next=true on first page, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthTokenCreateSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"name":"ci-bot","scope":"write"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/tokens", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token":"htk_live_`) {
		t.Fatalf("expected plaintext token in body, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsAuthTokenRevokeSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/tokens/tok_01JZTOKEN000000000000001", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRegisterOperationsAuthTokenRevokeUnknown(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/tokens/tok_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsUserCreateSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"username":"bob","role":"user","password":"hunter2hunter"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"username":"bob"`) {
		t.Fatalf("expected created username in body, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsUserCreateConflict(t *testing.T) {
	mux := testAPI()
	body := `{"username":"admin","role":"admin","password":"hunter2hunter"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestRegisterOperationsUserGetSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user_admin", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"totp_enabled":true`) {
		t.Fatalf("expected totp_enabled in body: %s", rec.Body.String())
	}
}

func TestRegisterOperationsUserGetNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsUserUpdateSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"role":"auditor","status":"active"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/user_admin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"role":"auditor"`) {
		t.Fatalf("expected updated role in body: %s", rec.Body.String())
	}
}

func TestRegisterOperationsUserUpdateNotFound(t *testing.T) {
	mux := testAPI()
	body := `{"role":"user"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/user_missing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsUserDeleteSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/user_admin", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsUserDeleteNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/user_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsUserSetScopeSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"scope":"restricted"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user_admin/scope", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"scope":"restricted"`) {
		t.Fatalf("expected applied scope in body: %s", rec.Body.String())
	}
}

func TestRegisterOperationsScheduleListPagination(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules?limit=1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"has_next":true`) {
		t.Fatalf("expected has_next=true, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsScheduleCreateEchoesFields(t *testing.T) {
	mux := testAPI()
	body := `{"name":"weekly","type":"snapshot","target":"vm-x","cron_expr":"Sun *-*-* 03:00:00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"weekly"`) {
		t.Fatalf("expected echoed name, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsScheduleGetSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched_01JZSCHEDULE00000000001", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsScheduleGetNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsScheduleUpdateSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/sched_01JZSCHEDULE00000000001", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("expected enabled=false in body, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsScheduleUpdateNotFound(t *testing.T) {
	mux := testAPI()
	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/sched_missing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsScheduleDeleteSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/sched_01JZSCHEDULE00000000001", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsScheduleDeleteNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/sched_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsScheduleRunNowSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched_01JZSCHEDULE00000000001/run", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"triggered"`) {
		t.Fatalf("expected triggered status, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsScheduleRunNowNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched_missing/run", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsWebhookListPagination(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks?limit=1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"has_next":true`) {
		t.Fatalf("expected has_next=true, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsWebhookCreateEchoes(t *testing.T) {
	mux := testAPI()
	body := `{"name":"pager","url":"https://example.com/hook","secret":"long-enough-secret-abc","events":["instance.created"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"pager"`) {
		t.Fatalf("expected name echoed, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsWebhookGetSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/whk_01JZWEBHOOK0000000000001", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsWebhookGetNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/whk_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsWebhookUpdateSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/webhooks/whk_01JZWEBHOOK0000000000001", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("expected enabled=false, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsWebhookUpdateNotFound(t *testing.T) {
	mux := testAPI()
	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/webhooks/whk_missing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsWebhookDeleteSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/webhooks/whk_01JZWEBHOOK0000000000001", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsWebhookDeleteNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/webhooks/whk_missing", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsWebhookTestSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/whk_01JZWEBHOOK0000000000001/test", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"delivered"`) {
		t.Fatalf("expected delivered status, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsWebhookTestNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/whk_missing/test", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsKubernetesListPagination(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/kubernetes?limit=1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"has_next":true`) {
		t.Fatalf("expected has_next=true, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsKubernetesCreateSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"name":"dev-cluster","version":"v1.30.5+k3s1","workers":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kubernetes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"provisioning"`) {
		t.Fatalf("expected provisioning status, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsKubernetesCreateConflict(t *testing.T) {
	mux := testAPI()
	body := `{"name":"prod-cluster","version":"v1.30.5+k3s1","workers":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kubernetes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestRegisterOperationsKubernetesGetNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/kubernetes/missing-cluster", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsKubernetesGetSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/kubernetes/prod-cluster", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsKubernetesDeleteSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/kubernetes/prod-cluster", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRegisterOperationsKubernetesDeleteNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/kubernetes/missing-cluster", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRegisterOperationsKubernetesScaleSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"workers":5}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/kubernetes/prod-cluster/scale", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"workers":5`) {
		t.Fatalf("expected workers=5, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsKubernetesUpgradeSuccess(t *testing.T) {
	mux := testAPI()
	body := `{"version":"v1.31.1+k3s1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/kubernetes/prod-cluster/upgrade", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"upgrading"`) {
		t.Fatalf("expected upgrading status, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsKubernetesKubeconfigSuccess(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/kubernetes/prod-cluster/kubeconfig", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "apiVersion: v1") {
		t.Fatalf("expected kubeconfig yaml body, got: %s", rec.Body.String())
	}
}

func TestRegisterOperationsKubernetesKubeconfigNotFound(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/kubernetes/missing-cluster/kubeconfig", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEnrichOpenAPIPatchesSchemaMetadata(t *testing.T) {
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig(apiTitle, apiVersion))
	RegisterOperations(api)
	EnrichOpenAPI(api.OpenAPI())

	doc := api.OpenAPI()
	if doc.Info == nil || doc.Info.Description == "" {
		t.Fatal("expected info description after enrichment")
	}
	if len(doc.Servers) == 0 {
		t.Fatal("expected servers after enrichment")
	}

	schemas := doc.Components.Schemas.Map()
	required := []string{"ErrorDetail", "ErrorModel", "HealthData", "HealthEnvelope", "HealthMeta", "AuthLoginEnvelope", "AuthLogoutEnvelope", "AuthRefreshEnvelope", "UserListEnvelope"}
	for _, name := range required {
		s := schemas[name]
		if s == nil || s.Description == "" {
			t.Fatalf("expected description for schema %s", name)
		}
	}

	errorDetail := schemas["ErrorDetail"]
	if errorDetail.Properties["value"].Type == "" {
		t.Fatal("expected type for ErrorDetail.value")
	}
	if len(errorDetail.Properties["message"].Examples) == 0 {
		t.Fatal("expected example for ErrorDetail.message")
	}

	for _, path := range []string{"/api/v1/health", "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh", "/api/v1/users"} {
		if doc.Paths[path] == nil {
			t.Fatalf("expected path %s in generated OpenAPI", path)
		}
	}
}

func TestStubFirewallHostList(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/firewall/host?limit=1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"limit":1`) {
		t.Fatalf("no limit: %s", rec.Body.String())
	}
}

func TestStubFirewallHostCreate(t *testing.T) {
	mux := testAPI()
	body := `{"chain":"input","priority":100,"action":"accept","expr":"tcp dport 22"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/firewall/host", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStubFirewallHostDelete(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/firewall/host/fwr_01JZFW00000000000000001", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d", rec.Code)
	}
}

func TestStubAuditQuery(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?actor=admin&action=auth.login&limit=10", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit query: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStubAuditExport(t *testing.T) {
	mux := testAPI()
	for _, f := range []string{"json", "csv"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?format="+f, http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("export %s: %d", f, rec.Code)
		}
	}
}

func TestStubEventsSse(t *testing.T) {
	mux := testAPI()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?limit=2", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("events: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStubSystemReachDefaults(t *testing.T) {
	mux := testAPI()
	for _, p := range []string{"/api/v1/system/info", "/api/v1/system/hardware", "/api/v1/system/diagnostics"} {
		req := httptest.NewRequest(http.MethodGet, p, http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d body=%s", p, rec.Code, rec.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/config/auth.session_inactivity_timeout", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config get: %d body=%s", rec.Code, rec.Body.String())
	}
	body := `{"value":"debug"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/system/config/auth.session_inactivity_timeout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config put: %d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/system/upgrade", strings.NewReader(`{"rollback":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upgrade: %d body=%s", rec.Code, rec.Body.String())
	}
}
