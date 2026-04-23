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
	body := `{"name":"ci-bot","scope":"user"}`
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
