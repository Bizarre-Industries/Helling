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
	authRefreshTokenStub  = "stub_refresh_token_01JZREFRESHABCDEFGHJK"
	authRefreshTokenInval = "invalid"
)

// RegisterOperations wires the current Huma spike operations.
func RegisterOperations(api huma.API) {
	registerAuthLogin(api)
	registerAuthLogout(api)
	registerAuthRefresh(api)
	registerUserList(api)
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
		if input.Cursor == "cursor_page_2" {
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
			nextCursor = "cursor_page_2"
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
