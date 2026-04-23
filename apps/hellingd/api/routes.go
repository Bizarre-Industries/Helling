// Package api defines Huma operations for Helling-owned routes.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	authUsernameAdmin     = "admin"
	authPasswordAdmin     = "correct-horse-battery-staple"
	authUsernameMFA       = "mfa"
	authUsernameRateLimit = "ratelimit"
)

// HealthData is the minimal payload used to prove the Huma pipeline wiring.
type HealthData struct {
	Status string `json:"status" doc:"Service health state." enum:"ok"`
}

// HealthMeta keeps the envelope shape aligned with docs/spec/api.md.
type HealthMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// HealthEnvelope follows the Helling success envelope contract.
type HealthEnvelope struct {
	Data HealthData `json:"data"`
	Meta HealthMeta `json:"meta"`
}

// HealthOutput is the response shape for GET /api/v1/health.
type HealthOutput struct {
	Body HealthEnvelope
}

// AuthLoginRequest is the login request payload for auth login.
type AuthLoginRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64" doc:"Username for PAM authentication."`
	Password string `json:"password" minLength:"1" maxLength:"256" doc:"Password for PAM authentication."`
	TOTPCode string `json:"totp_code,omitempty" minLength:"6" maxLength:"8" doc:"Optional TOTP code for MFA completion."`
}

// AuthLoginInput wraps the auth login request body.
type AuthLoginInput struct {
	Body AuthLoginRequest `doc:"PAM credentials with optional inline TOTP code."`
}

// AuthLoginData is the result payload for auth login.
type AuthLoginData struct {
	AccessToken string `json:"access_token,omitempty" doc:"JWT access token when login succeeds without MFA challenge."`
	TokenType   string `json:"token_type,omitempty" doc:"Token scheme for access token responses." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in,omitempty" doc:"Access token TTL in seconds." minimum:"1"`
	MFARequired bool   `json:"mfa_required,omitempty" doc:"Indicates whether MFA completion is required before token issuance."`
	MFAToken    string `json:"mfa_token,omitempty" doc:"Opaque token used to complete MFA challenge."`
}

// AuthLoginMeta contains request metadata for auth responses.
type AuthLoginMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthLoginEnvelope follows the Helling success envelope shape for auth login.
type AuthLoginEnvelope struct {
	Data AuthLoginData `json:"data"`
	Meta AuthLoginMeta `json:"meta"`
}

// AuthLoginOutput supports 200 and 202 responses for auth login.
type AuthLoginOutput struct {
	Status int `status:"200"`
	Body   AuthLoginEnvelope
}

// UserListInput contains pagination controls for listing users.
type UserListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of users to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// UserRecord is a lightweight user summary payload.
type UserRecord struct {
	ID       string `json:"id" doc:"User identifier."`
	Username string `json:"username" doc:"Username."`
	Role     string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status   string `json:"status" doc:"Account status." enum:"active,disabled"`
}

// UserPageMeta is pagination metadata for list endpoints.
type UserPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// UserListMeta captures request and paging metadata for user list responses.
type UserListMeta struct {
	RequestID string       `json:"request_id" doc:"Request correlation ID."`
	Page      UserPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// UserListEnvelope follows the Helling list envelope shape.
type UserListEnvelope struct {
	Data []UserRecord `json:"data"`
	Meta UserListMeta `json:"meta"`
}

// UserListOutput is the response shape for GET /api/v1/users.
type UserListOutput struct {
	Body UserListEnvelope
}

var stubUsers = []UserRecord{
	{ID: "user_admin", Username: "admin", Role: "admin", Status: "active"},
	{ID: "user_alice", Username: "alice", Role: "user", Status: "active"},
}

const userIDUnknown = "user_missing"

// UserCreateRequest creates a new PAM-backed user account.
type UserCreateRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64" doc:"Unix-safe username. PAM constraints apply."`
	Role     string `json:"role" doc:"Fixed role assignment." enum:"admin,user,auditor"`
	Password string `json:"password" minLength:"8" maxLength:"256" doc:"Initial password, fed to passwd(1). Optional when delegating to external provisioning."`
}

// UserCreateInput wraps the create body.
type UserCreateInput struct {
	Body UserCreateRequest `doc:"User creation payload."`
}

// UserCreateData returns the created user summary.
type UserCreateData struct {
	ID       string `json:"id" doc:"New user identifier."`
	Username string `json:"username" doc:"Username."`
	Role     string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status   string `json:"status" doc:"Account status." enum:"active,disabled"`
}

// UserCreateMeta contains request metadata.
type UserCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserCreateEnvelope follows the success envelope shape.
type UserCreateEnvelope struct {
	Data UserCreateData `json:"data"`
	Meta UserCreateMeta `json:"meta"`
}

// UserCreateOutput is the response shape for POST /api/v1/users.
type UserCreateOutput struct {
	Body UserCreateEnvelope
}

// UserGetInput binds the path id.
type UserGetInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
}

// UserGetData returns detailed user fields including MFA status.
type UserGetData struct {
	ID          string `json:"id" doc:"User identifier."`
	Username    string `json:"username" doc:"Username."`
	Role        string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status      string `json:"status" doc:"Account status." enum:"active,disabled"`
	TotpEnabled bool   `json:"totp_enabled" doc:"Whether TOTP is currently enrolled."`
	LastLogin   string `json:"last_login,omitempty" doc:"ISO-8601 timestamp of last successful login." format:"date-time"`
}

// UserGetMeta contains request metadata.
type UserGetMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserGetEnvelope follows the success envelope shape.
type UserGetEnvelope struct {
	Data UserGetData `json:"data"`
	Meta UserGetMeta `json:"meta"`
}

// UserGetOutput is the response shape for GET /api/v1/users/{id}.
type UserGetOutput struct {
	Body UserGetEnvelope
}

// UserUpdateRequest applies a partial update to a user.
type UserUpdateRequest struct {
	Role   string `json:"role,omitempty" doc:"New role assignment." enum:"admin,user,auditor"`
	Status string `json:"status,omitempty" doc:"New account status." enum:"active,disabled"`
}

// UserUpdateInput combines path id with update body.
type UserUpdateInput struct {
	ID   string            `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
	Body UserUpdateRequest `doc:"Partial update payload."`
}

// UserUpdateData returns the updated user summary.
type UserUpdateData struct {
	ID       string `json:"id" doc:"User identifier."`
	Username string `json:"username" doc:"Username."`
	Role     string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status   string `json:"status" doc:"Account status." enum:"active,disabled"`
}

// UserUpdateMeta contains request metadata.
type UserUpdateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserUpdateEnvelope follows the success envelope shape.
type UserUpdateEnvelope struct {
	Data UserUpdateData `json:"data"`
	Meta UserUpdateMeta `json:"meta"`
}

// UserUpdateOutput is the response shape for PATCH /api/v1/users/{id}.
type UserUpdateOutput struct {
	Body UserUpdateEnvelope
}

// UserDeleteInput binds the path id.
type UserDeleteInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
}

// UserDeleteData is empty on success.
type UserDeleteData struct{}

// UserDeleteMeta contains request metadata.
type UserDeleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserDeleteEnvelope follows the success envelope shape.
type UserDeleteEnvelope struct {
	Data UserDeleteData `json:"data"`
	Meta UserDeleteMeta `json:"meta"`
}

// UserDeleteOutput is the response shape for DELETE /api/v1/users/{id}.
type UserDeleteOutput struct {
	Body UserDeleteEnvelope
}

// UserSetScopeRequest applies an Incus trust-scope change.
type UserSetScopeRequest struct {
	Scope string `json:"scope" doc:"Incus trust scope. Controls the set of Incus projects/resources the user's cert is allowed to touch." enum:"default,restricted,admin"`
}

// UserSetScopeInput combines path id with scope body.
type UserSetScopeInput struct {
	ID   string              `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
	Body UserSetScopeRequest `doc:"Trust-scope assignment payload."`
}

// UserSetScopeData returns the updated scope.
type UserSetScopeData struct {
	ID    string `json:"id" doc:"User identifier."`
	Scope string `json:"scope" doc:"Applied trust scope." enum:"default,restricted,admin"`
}

// UserSetScopeMeta contains request metadata.
type UserSetScopeMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserSetScopeEnvelope follows the success envelope shape.
type UserSetScopeEnvelope struct {
	Data UserSetScopeData `json:"data"`
	Meta UserSetScopeMeta `json:"meta"`
}

// UserSetScopeOutput is the response shape for PUT /api/v1/users/{id}/scope.
type UserSetScopeOutput struct {
	Body UserSetScopeEnvelope
}

// AuthLogoutData is the payload returned on successful logout.
// Empty object preserves the envelope contract (data + meta) without leaking session material.
type AuthLogoutData struct{}

// AuthLogoutMeta contains request metadata for logout responses.
type AuthLogoutMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthLogoutEnvelope follows the Helling success envelope shape for logout.
type AuthLogoutEnvelope struct {
	Data AuthLogoutData `json:"data"`
	Meta AuthLogoutMeta `json:"meta"`
}

// AuthLogoutOutput is the response shape for POST /api/v1/auth/logout.
type AuthLogoutOutput struct {
	Body AuthLogoutEnvelope
}

// AuthRefreshRequest is the refresh request payload.
// v0.1-alpha accepts the refresh token in the body; v0.1-beta moves to the httpOnly
// cookie model documented in docs/spec/auth.md §2.2.
type AuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token" minLength:"1" maxLength:"4096" doc:"Opaque refresh token issued by a prior login."`
}

// AuthRefreshInput wraps the refresh request body.
type AuthRefreshInput struct {
	Body AuthRefreshRequest `doc:"Refresh token exchange payload."`
}

// AuthRefreshData is the result payload for refresh.
type AuthRefreshData struct {
	AccessToken string `json:"access_token" doc:"New JWT access token."`
	TokenType   string `json:"token_type" doc:"Token scheme for access token responses." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in" doc:"Access token TTL in seconds." minimum:"1"`
}

// AuthRefreshMeta contains request metadata for refresh responses.
type AuthRefreshMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthRefreshEnvelope follows the Helling success envelope shape for refresh.
type AuthRefreshEnvelope struct {
	Data AuthRefreshData `json:"data"`
	Meta AuthRefreshMeta `json:"meta"`
}

// AuthRefreshOutput is the response shape for POST /api/v1/auth/refresh.
type AuthRefreshOutput struct {
	Body AuthRefreshEnvelope
}

const (
	cursorPage2 = "cursor_page_2"

	authRefreshTokenStub  = "stub_refresh_token_01JZREFRESHABCDEFGHJK"
	authRefreshTokenInval = "invalid"
	authMfaTokenStub      = "mfa_01JZABC0123456789ABCDEFGJK" //nolint:gosec // G101: stub fixture, not a real credential.
	authTotpCodeValid     = "123456"
	authTokenIDExisting   = "tok_01JZTOKEN000000000000001" //nolint:gosec // G101: stub path parameter, not a real credential.
	authTokenIDUnknown    = "tok_missing"                  //nolint:gosec // G101: stub path parameter, not a real credential.
)

// AuthSetupRequest is the initial-admin-setup payload.
type AuthSetupRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64" doc:"Username for the initial admin account."`
	Password string `json:"password" minLength:"8" maxLength:"256" doc:"Password for the initial admin account."`
}

// AuthSetupInput wraps the setup request body.
type AuthSetupInput struct {
	Body AuthSetupRequest `doc:"Initial admin account credentials."`
}

// AuthSetupData mirrors AuthLoginData on successful first-time setup.
type AuthSetupData struct {
	AccessToken string `json:"access_token" doc:"JWT access token issued for the new admin account."`
	TokenType   string `json:"token_type" doc:"Token scheme." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in" doc:"Access token TTL in seconds." minimum:"1"`
}

// AuthSetupMeta contains request metadata for setup responses.
type AuthSetupMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthSetupEnvelope follows the Helling success envelope shape.
type AuthSetupEnvelope struct {
	Data AuthSetupData `json:"data"`
	Meta AuthSetupMeta `json:"meta"`
}

// AuthSetupOutput is the response shape for POST /api/v1/auth/setup.
type AuthSetupOutput struct {
	Body AuthSetupEnvelope
}

// AuthMfaCompleteRequest completes an MFA challenge with a TOTP code.
type AuthMfaCompleteRequest struct {
	MfaToken string `json:"mfa_token" minLength:"1" maxLength:"256" doc:"MFA token returned by the login endpoint."`
	TotpCode string `json:"totp_code" minLength:"6" maxLength:"8" doc:"Six-digit TOTP code from the user's authenticator."`
}

// AuthMfaCompleteInput wraps the MFA completion body.
type AuthMfaCompleteInput struct {
	Body AuthMfaCompleteRequest `doc:"MFA completion payload."`
}

// AuthMfaCompleteData returns full token pair after successful MFA.
type AuthMfaCompleteData struct {
	AccessToken string `json:"access_token" doc:"JWT access token."`
	TokenType   string `json:"token_type" doc:"Token scheme." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in" doc:"Access token TTL in seconds." minimum:"1"`
}

// AuthMfaCompleteMeta contains request metadata.
type AuthMfaCompleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthMfaCompleteEnvelope follows the Helling success envelope shape.
type AuthMfaCompleteEnvelope struct {
	Data AuthMfaCompleteData `json:"data"`
	Meta AuthMfaCompleteMeta `json:"meta"`
}

// AuthMfaCompleteOutput is the response shape for POST /api/v1/auth/mfa/complete.
type AuthMfaCompleteOutput struct {
	Body AuthMfaCompleteEnvelope
}

// AuthTotpSetupData returns enrollment artifacts for a new TOTP factor.
type AuthTotpSetupData struct {
	ProvisioningURI string   `json:"provisioning_uri" doc:"otpauth:// URI that authenticator apps can scan."`
	Secret          string   `json:"secret" doc:"Raw base32 TOTP secret for manual entry."`
	RecoveryCodes   []string `json:"recovery_codes" doc:"One-time recovery codes for out-of-band access."`
}

// AuthTotpSetupMeta contains request metadata for TOTP setup responses.
type AuthTotpSetupMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTotpSetupEnvelope follows the Helling success envelope shape.
type AuthTotpSetupEnvelope struct {
	Data AuthTotpSetupData `json:"data"`
	Meta AuthTotpSetupMeta `json:"meta"`
}

// AuthTotpSetupOutput is the response shape for POST /api/v1/auth/totp/setup.
type AuthTotpSetupOutput struct {
	Body AuthTotpSetupEnvelope
}

// AuthTotpVerifyRequest confirms a TOTP enrollment with a code.
type AuthTotpVerifyRequest struct {
	TotpCode string `json:"totp_code" minLength:"6" maxLength:"8" doc:"TOTP code from the user's authenticator app."`
}

// AuthTotpVerifyInput wraps the verify request body.
type AuthTotpVerifyInput struct {
	Body AuthTotpVerifyRequest `doc:"TOTP verification payload."`
}

// AuthTotpVerifyData is empty on success; envelope preserved.
type AuthTotpVerifyData struct{}

// AuthTotpVerifyMeta contains request metadata.
type AuthTotpVerifyMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTotpVerifyEnvelope follows the Helling success envelope shape.
type AuthTotpVerifyEnvelope struct {
	Data AuthTotpVerifyData `json:"data"`
	Meta AuthTotpVerifyMeta `json:"meta"`
}

// AuthTotpVerifyOutput is the response shape for POST /api/v1/auth/totp/verify.
type AuthTotpVerifyOutput struct {
	Body AuthTotpVerifyEnvelope
}

// AuthTotpDisableData is empty on success.
type AuthTotpDisableData struct{}

// AuthTotpDisableMeta contains request metadata.
type AuthTotpDisableMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTotpDisableEnvelope follows the Helling success envelope shape.
type AuthTotpDisableEnvelope struct {
	Data AuthTotpDisableData `json:"data"`
	Meta AuthTotpDisableMeta `json:"meta"`
}

// AuthTotpDisableOutput is the response shape for POST /api/v1/auth/totp/disable.
type AuthTotpDisableOutput struct {
	Body AuthTotpDisableEnvelope
}

// AuthTokenRecord is an API token summary.
type AuthTokenRecord struct {
	ID        string `json:"id" doc:"Token identifier."`
	Name      string `json:"name" doc:"User-supplied token name."`
	Scope     string `json:"scope" doc:"Token scope." enum:"admin,user,auditor"`
	CreatedAt string `json:"created_at" doc:"ISO-8601 timestamp when the token was created." format:"date-time"`
	LastUsed  string `json:"last_used,omitempty" doc:"ISO-8601 timestamp of last successful use, if any." format:"date-time"`
}

// AuthTokenPageMeta is pagination metadata for token lists.
type AuthTokenPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// AuthTokenListInput has pagination controls.
type AuthTokenListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of tokens to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// AuthTokenListMeta contains request and paging metadata.
type AuthTokenListMeta struct {
	RequestID string            `json:"request_id" doc:"Request correlation ID."`
	Page      AuthTokenPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// AuthTokenListEnvelope follows the Helling list envelope shape.
type AuthTokenListEnvelope struct {
	Data []AuthTokenRecord `json:"data"`
	Meta AuthTokenListMeta `json:"meta"`
}

// AuthTokenListOutput is the response shape for GET /api/v1/auth/tokens.
type AuthTokenListOutput struct {
	Body AuthTokenListEnvelope
}

// AuthTokenCreateRequest creates a new API token.
type AuthTokenCreateRequest struct {
	Name  string `json:"name" minLength:"1" maxLength:"128" doc:"User-visible token name."`
	Scope string `json:"scope" doc:"Token scope, must be one of the fixed roles." enum:"admin,user,auditor"`
}

// AuthTokenCreateInput wraps the create request body.
type AuthTokenCreateInput struct {
	Body AuthTokenCreateRequest `doc:"Token creation payload."`
}

// AuthTokenCreateData returns the newly-created token plaintext exactly once.
type AuthTokenCreateData struct {
	ID    string `json:"id" doc:"New token identifier."`
	Name  string `json:"name" doc:"Token name."`
	Scope string `json:"scope" doc:"Token scope." enum:"admin,user,auditor"`
	Token string `json:"token" doc:"Plaintext API token. Surfaced only once; store it securely."`
}

// AuthTokenCreateMeta contains request metadata.
type AuthTokenCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTokenCreateEnvelope follows the Helling success envelope shape.
type AuthTokenCreateEnvelope struct {
	Data AuthTokenCreateData `json:"data"`
	Meta AuthTokenCreateMeta `json:"meta"`
}

// AuthTokenCreateOutput is the response shape for POST /api/v1/auth/tokens.
type AuthTokenCreateOutput struct {
	Body AuthTokenCreateEnvelope
}

// AuthTokenRevokeInput binds the path id.
type AuthTokenRevokeInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Token identifier to revoke."`
}

// AuthTokenRevokeData is empty on success.
type AuthTokenRevokeData struct{}

// AuthTokenRevokeMeta contains request metadata.
type AuthTokenRevokeMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTokenRevokeEnvelope follows the Helling success envelope shape.
type AuthTokenRevokeEnvelope struct {
	Data AuthTokenRevokeData `json:"data"`
	Meta AuthTokenRevokeMeta `json:"meta"`
}

// AuthTokenRevokeOutput is the response shape for DELETE /api/v1/auth/tokens/{id}.
type AuthTokenRevokeOutput struct {
	Body AuthTokenRevokeEnvelope
}

var stubAuthTokens = []AuthTokenRecord{
	{ID: authTokenIDExisting, Name: "ci-bot", Scope: "user", CreatedAt: "2026-04-01T00:00:00Z", LastUsed: "2026-04-20T12:00:00Z"},
	{ID: "tok_01JZTOKEN000000000000002", Name: "auditor-readonly", Scope: "auditor", CreatedAt: "2026-04-10T00:00:00Z"},
}

// RegisterOperations wires the current Huma spike operations.
func RegisterOperations(api huma.API) {
	registerAuthSetup(api)
	registerAuthLogin(api)
	registerAuthLogout(api)
	registerAuthRefresh(api)
	registerAuthMfaComplete(api)
	registerAuthTotpSetup(api)
	registerAuthTotpVerify(api)
	registerAuthTotpDisable(api)
	registerAuthTokenList(api)
	registerAuthTokenCreate(api)
	registerAuthTokenRevoke(api)
	registerUserList(api)
	registerUserCreate(api)
	registerUserGet(api)
	registerUserUpdate(api)
	registerUserDelete(api)
	registerUserSetScope(api)
	registerHealth(api)
}

func registerHealth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Health check",
		Description: "Returns service health for unauthenticated readiness checks.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		_ = ctx
		_ = input

		return &HealthOutput{
			Body: HealthEnvelope{
				Data: HealthData{Status: "ok"},
				Meta: HealthMeta{RequestID: "req_huma_spike"},
			},
		}, nil
	})
}

func registerAuthLogin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authLogin",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "PAM authenticate and issue JWT pair",
		Description: "Authenticates a PAM user and returns tokens or an MFA challenge.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "PAM credentials with optional inline TOTP code.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized, http.StatusTooManyRequests},
		Responses: map[string]*huma.Response{
			"202": {
				Description: "MFA challenge required before token issuance.",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{Ref: "#/components/schemas/AuthLoginEnvelope"},
					},
				},
			},
		},
	}, func(ctx context.Context, input *AuthLoginInput) (*AuthLoginOutput, error) {
		_ = ctx

		if input.Body.Username == authUsernameRateLimit {
			return nil, huma.Error429TooManyRequests("AUTH_RATE_LIMITED")
		}

		if input.Body.Username == authUsernameMFA && input.Body.TOTPCode == "" {
			return &AuthLoginOutput{
				Status: http.StatusAccepted,
				Body: AuthLoginEnvelope{
					Data: AuthLoginData{
						MFARequired: true,
						MFAToken:    "mfa_01JZABC0123456789ABCDEFGJK",
					},
					Meta: AuthLoginMeta{RequestID: "req_auth_login_mfa"},
				},
			}, nil
		}

		if input.Body.Username != authUsernameAdmin || input.Body.Password != authPasswordAdmin {
			return nil, huma.Error401Unauthorized("AUTH_INVALID_CREDENTIALS")
		}

		return &AuthLoginOutput{
			Status: http.StatusOK,
			Body: AuthLoginEnvelope{
				Data: AuthLoginData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthLoginMeta{RequestID: "req_auth_login_ok"},
			},
		}, nil
	})
}

func registerAuthLogout(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authLogout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Revoke the current session",
		Description: "Invalidates the caller's refresh token server-side. Stub implementation returns empty success envelope until the token store lands. Bearer-auth requirement will be declared once the bearerAuth scheme ships with the JWT middleware.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *struct{}) (*AuthLogoutOutput, error) {
		_ = ctx
		_ = input

		return &AuthLogoutOutput{
			Body: AuthLogoutEnvelope{
				Data: AuthLogoutData{},
				Meta: AuthLogoutMeta{RequestID: "req_auth_logout"},
			},
		}, nil
	})
}

func registerAuthRefresh(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authRefresh",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Exchange a refresh token for a new access token",
		Description: "Issues a new short-lived access token when the supplied refresh token is valid and within the inactivity window.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "Refresh token exchange payload.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *AuthRefreshInput) (*AuthRefreshOutput, error) {
		_ = ctx

		if input.Body.RefreshToken == authRefreshTokenInval {
			return nil, huma.Error401Unauthorized("AUTH_REFRESH_INVALID")
		}

		if input.Body.RefreshToken != authRefreshTokenStub {
			return nil, huma.Error401Unauthorized("AUTH_REFRESH_INVALID")
		}

		return &AuthRefreshOutput{
			Body: AuthRefreshEnvelope{
				Data: AuthRefreshData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.refresh.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthRefreshMeta{RequestID: "req_auth_refresh"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to registerAuthTokenList; cursor pagination shape is the repo idiom.
func registerUserList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userList",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List users",
		Description: "Lists users using cursor pagination metadata in the response envelope.",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *UserListInput) (*UserListOutput, error) {
		_ = ctx

		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}

		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubUsers) {
			start = len(stubUsers)
		}

		end := start + limit
		if end > len(stubUsers) {
			end = len(stubUsers)
		}

		hasNext := end < len(stubUsers)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}

		users := append([]UserRecord(nil), stubUsers[start:end]...)
		return &UserListOutput{
			Body: UserListEnvelope{
				Data: users,
				Meta: UserListMeta{
					RequestID: "req_user_list",
					Page: UserPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerAuthSetup(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authSetup",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/setup",
		Summary:     "Create the initial admin account",
		Description: "Bootstraps the first administrator on a fresh install. Idempotently refuses if any admin already exists; stub accepts any payload.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "Initial admin credentials.",
			Required:    true,
		},
		Errors: []int{http.StatusConflict},
	}, func(ctx context.Context, input *AuthSetupInput) (*AuthSetupOutput, error) {
		_ = ctx
		_ = input
		return &AuthSetupOutput{
			Body: AuthSetupEnvelope{
				Data: AuthSetupData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.setup.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthSetupMeta{RequestID: "req_auth_setup"},
			},
		}, nil
	})
}

func registerAuthMfaComplete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authMfaComplete",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/mfa/complete",
		Summary:     "Complete an MFA challenge with a TOTP code",
		Description: "Exchanges an mfa_token plus a valid TOTP code for a full JWT access token.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "MFA completion payload.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *AuthMfaCompleteInput) (*AuthMfaCompleteOutput, error) {
		_ = ctx
		if input.Body.MfaToken != authMfaTokenStub {
			return nil, huma.Error401Unauthorized("AUTH_MFA_INVALID")
		}
		if input.Body.TotpCode != authTotpCodeValid {
			return nil, huma.Error401Unauthorized("AUTH_MFA_CODE_INVALID")
		}
		return &AuthMfaCompleteOutput{
			Body: AuthMfaCompleteEnvelope{
				Data: AuthMfaCompleteData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.mfa.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthMfaCompleteMeta{RequestID: "req_auth_mfa_complete"},
			},
		}, nil
	})
}

func registerAuthTotpSetup(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpSetup",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/setup",
		Summary:     "Begin TOTP enrollment for the current user",
		Description: "Issues a new TOTP secret, provisioning URI, and a set of single-use recovery codes. The factor must be confirmed with /auth/totp/verify before it is active.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *struct{}) (*AuthTotpSetupOutput, error) {
		_ = ctx
		_ = input
		return &AuthTotpSetupOutput{
			Body: AuthTotpSetupEnvelope{
				Data: AuthTotpSetupData{
					ProvisioningURI: "otpauth://totp/Helling:admin?secret=JBSWY3DPEHPK3PXP&issuer=Helling",
					Secret:          "JBSWY3DPEHPK3PXP",
					RecoveryCodes: []string{
						"11112222", "33334444", "55556666", "77778888", "99990000",
						"aaaabbbb", "ccccdddd", "eeeeffff", "gggghhhh", "iiiijjjj",
					},
				},
				Meta: AuthTotpSetupMeta{RequestID: "req_auth_totp_setup"},
			},
		}, nil
	})
}

func registerAuthTotpVerify(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpVerify",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/verify",
		Summary:     "Confirm a pending TOTP enrollment",
		Description: "Activates the pending TOTP factor when the supplied code is valid.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "TOTP verification payload.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *AuthTotpVerifyInput) (*AuthTotpVerifyOutput, error) {
		_ = ctx
		if input.Body.TotpCode != authTotpCodeValid {
			return nil, huma.Error401Unauthorized("AUTH_TOTP_CODE_INVALID")
		}
		return &AuthTotpVerifyOutput{
			Body: AuthTotpVerifyEnvelope{
				Data: AuthTotpVerifyData{},
				Meta: AuthTotpVerifyMeta{RequestID: "req_auth_totp_verify"},
			},
		}, nil
	})
}

func registerAuthTotpDisable(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpDisable",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/disable",
		Summary:     "Disable TOTP for the current user",
		Description: "Removes the active TOTP factor. Admin-initiated removals are out of scope in v0.1.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *struct{}) (*AuthTotpDisableOutput, error) {
		_ = ctx
		_ = input
		return &AuthTotpDisableOutput{
			Body: AuthTotpDisableEnvelope{
				Data: AuthTotpDisableData{},
				Meta: AuthTotpDisableMeta{RequestID: "req_auth_totp_disable"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to registerUserList; cursor pagination shape is the repo idiom.
func registerAuthTokenList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenList",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/tokens",
		Summary:     "List API tokens",
		Description: "Lists API tokens belonging to the current user using cursor pagination.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *AuthTokenListInput) (*AuthTokenListOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubAuthTokens) {
			start = len(stubAuthTokens)
		}
		end := start + limit
		if end > len(stubAuthTokens) {
			end = len(stubAuthTokens)
		}
		hasNext := end < len(stubAuthTokens)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}
		tokens := append([]AuthTokenRecord(nil), stubAuthTokens[start:end]...)
		return &AuthTokenListOutput{
			Body: AuthTokenListEnvelope{
				Data: tokens,
				Meta: AuthTokenListMeta{
					RequestID: "req_auth_token_list",
					Page: AuthTokenPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerAuthTokenCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/tokens",
		Summary:     "Create a new API token",
		Description: "Creates and returns a new API token. Plaintext token is surfaced exactly once.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "Token creation payload.",
			Required:    true,
		},
	}, func(ctx context.Context, input *AuthTokenCreateInput) (*AuthTokenCreateOutput, error) {
		_ = ctx
		return &AuthTokenCreateOutput{
			Body: AuthTokenCreateEnvelope{
				Data: AuthTokenCreateData{
					ID:    "tok_01JZTOKEN000000000000003",
					Name:  input.Body.Name,
					Scope: input.Body.Scope,
					Token: "htk_live_stubtokenvalue0123456789abcdef",
				},
				Meta: AuthTokenCreateMeta{RequestID: "req_auth_token_create"},
			},
		}, nil
	})
}

func registerUserCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/users",
		Summary:     "Create a user",
		Description: "Creates a new PAM-backed user and returns the summary. Stub accepts any payload and echoes the username back.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{
			Description: "User creation payload.",
			Required:    true,
		},
		Errors: []int{http.StatusConflict},
	}, func(ctx context.Context, input *UserCreateInput) (*UserCreateOutput, error) {
		_ = ctx
		if input.Body.Username == "admin" {
			return nil, huma.Error409Conflict("USER_ALREADY_EXISTS")
		}
		return &UserCreateOutput{
			Body: UserCreateEnvelope{
				Data: UserCreateData{
					ID:       "user_" + input.Body.Username,
					Username: input.Body.Username,
					Role:     input.Body.Role,
					Status:   "active",
				},
				Meta: UserCreateMeta{RequestID: "req_user_create"},
			},
		}, nil
	})
}

func registerUserGet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/{id}",
		Summary:     "Get a user",
		Description: "Returns detailed user fields including TOTP enrollment state and last-login timestamp.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserGetInput) (*UserGetOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		return &UserGetOutput{
			Body: UserGetEnvelope{
				Data: UserGetData{
					ID:          input.ID,
					Username:    "admin",
					Role:        "admin",
					Status:      "active",
					TotpEnabled: true,
					LastLogin:   "2026-04-23T19:00:00Z",
				},
				Meta: UserGetMeta{RequestID: "req_user_get"},
			},
		}, nil
	})
}

func registerUserUpdate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userUpdate",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/{id}",
		Summary:     "Update a user",
		Description: "Applies a partial update. Only role and status are mutable in v0.1.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{
			Description: "Partial update payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserUpdateInput) (*UserUpdateOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		role := input.Body.Role
		if role == "" {
			role = "user"
		}
		status := input.Body.Status
		if status == "" {
			status = "active"
		}
		return &UserUpdateOutput{
			Body: UserUpdateEnvelope{
				Data: UserUpdateData{
					ID:       input.ID,
					Username: "admin",
					Role:     role,
					Status:   status,
				},
				Meta: UserUpdateMeta{RequestID: "req_user_update"},
			},
		}, nil
	})
}

func registerUserDelete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/users/{id}",
		Summary:     "Delete a user",
		Description: "Removes a PAM-backed user. Deleting an unknown user returns 404.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserDeleteInput) (*UserDeleteOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		return &UserDeleteOutput{
			Body: UserDeleteEnvelope{
				Data: UserDeleteData{},
				Meta: UserDeleteMeta{RequestID: "req_user_delete"},
			},
		}, nil
	})
}

func registerUserSetScope(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userSetScope",
		Method:      http.MethodPut,
		Path:        "/api/v1/users/{id}/scope",
		Summary:     "Assign a trust scope to a user",
		Description: "Applies an Incus trust-scope change. The scope controls which Incus projects the user's cert can touch.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{
			Description: "Trust-scope assignment payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserSetScopeInput) (*UserSetScopeOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		return &UserSetScopeOutput{
			Body: UserSetScopeEnvelope{
				Data: UserSetScopeData{
					ID:    input.ID,
					Scope: input.Body.Scope,
				},
				Meta: UserSetScopeMeta{RequestID: "req_user_set_scope"},
			},
		}, nil
	})
}

func registerAuthTokenRevoke(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenRevoke",
		Method:      http.MethodDelete,
		Path:        "/api/v1/auth/tokens/{id}",
		Summary:     "Revoke an API token",
		Description: "Invalidates the given token. Revoking an already-unknown token returns 404.",
		Tags:        []string{"Auth"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *AuthTokenRevokeInput) (*AuthTokenRevokeOutput, error) {
		_ = ctx
		if input.ID == authTokenIDUnknown {
			return nil, huma.Error404NotFound("AUTH_TOKEN_NOT_FOUND")
		}
		return &AuthTokenRevokeOutput{
			Body: AuthTokenRevokeEnvelope{
				Data: AuthTokenRevokeData{},
				Meta: AuthTokenRevokeMeta{RequestID: "req_auth_token_revoke"},
			},
		}, nil
	})
}
