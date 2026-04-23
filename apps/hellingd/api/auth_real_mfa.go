package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// bearerInput captures the Authorization header for endpoints that require
// a JWT access token or API token.
type bearerInput struct {
	Authorization string `header:"Authorization"`
	UserAgent     string `header:"User-Agent"`
	XRealIP       string `header:"X-Real-IP"`
}

// resolveCaller extracts the user id from either a JWT access token or a
// helling_* API token carried in the Authorization header. Role-based
// authorisation is layered on top by a future middleware.
func resolveCaller(ctx context.Context, svc *auth.Service, authz string) (string, error) {
	tok := strings.TrimPrefix(authz, "Bearer ")
	tok = strings.TrimSpace(tok)
	if tok == "" || authz == tok {
		return "", auth.ErrInvalidToken
	}

	if strings.HasPrefix(tok, auth.APITokenPrefix) {
		u, _, err := svc.VerifyAPIToken(ctx, tok)
		if err != nil {
			return "", auth.ErrInvalidToken
		}
		return u.ID, nil
	}

	claims, err := svc.Signer().Verify(tok)
	if err != nil {
		return "", err
	}
	return claims.Subject, nil
}

type authMfaCompleteCookieInput struct {
	UserAgent string `header:"User-Agent"`
	XRealIP   string `header:"X-Real-IP"`
	Body      AuthMfaCompleteRequest
}

type authMfaCompleteCookieOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      AuthMfaCompleteEnvelope
}

func registerAuthMfaCompleteReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authMfaComplete",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/mfa/complete",
		Summary:     "Complete an MFA challenge with a TOTP code",
		Description: "Exchanges an mfa_token plus a valid TOTP (or recovery) code for a full JWT pair.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{Description: "MFA completion payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *authMfaCompleteCookieInput) (*authMfaCompleteCookieOutput, error) {
		ident, err := svc.CompleteMFA(ctx, input.Body.MfaToken, input.Body.TotpCode, input.XRealIP, input.UserAgent)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_MFA_INVALID")
		}
		return &authMfaCompleteCookieOutput{
			SetCookie: refreshCookieFor(ident.RefreshToken, int(svc.Signer().RefreshTTL().Seconds())),
			Body: AuthMfaCompleteEnvelope{
				Data: AuthMfaCompleteData{
					AccessToken: ident.AccessToken,
					TokenType:   "Bearer",
					ExpiresIn:   ident.AccessExpires,
				},
				Meta: AuthMfaCompleteMeta{RequestID: "req_auth_mfa_complete"},
			},
		}, nil
	})
}

func registerAuthTotpSetupReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpSetup",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/setup",
		Summary:     "Begin TOTP enrollment for the current user",
		Description: "Issues a new TOTP secret, provisioning URI, and recovery codes. The factor must be confirmed via /auth/totp/verify.",
		Tags:        []string{"Auth"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *bearerInput) (*AuthTotpSetupOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		res, err := svc.EnrollTOTP(ctx, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("AUTH_TOTP_ENROLL_FAILED")
		}
		return &AuthTotpSetupOutput{
			Body: AuthTotpSetupEnvelope{
				Data: AuthTotpSetupData{
					ProvisioningURI: res.ProvisioningURI,
					Secret:          res.Secret,
					RecoveryCodes:   res.RecoveryCodes,
				},
				Meta: AuthTotpSetupMeta{RequestID: "req_auth_totp_setup"},
			},
		}, nil
	})
}

type authTotpVerifyBearerInput struct {
	Authorization string `header:"Authorization"`
	Body          AuthTotpVerifyRequest
}

func registerAuthTotpVerifyReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpVerify",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/verify",
		Summary:     "Confirm a pending TOTP enrollment",
		Description: "Activates the pending TOTP factor when the supplied code is valid.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{Description: "TOTP verification payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *authTotpVerifyBearerInput) (*AuthTotpVerifyOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if err := svc.VerifyTOTPEnroll(ctx, userID, input.Body.TotpCode); err != nil {
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

func registerAuthTotpDisableReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpDisable",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/disable",
		Summary:     "Disable TOTP for the current user",
		Description: "Removes the active TOTP factor and all recovery codes.",
		Tags:        []string{"Auth"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *bearerInput) (*AuthTotpDisableOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if err := svc.DisableTOTP(ctx, userID); err != nil {
			return nil, huma.Error500InternalServerError("AUTH_TOTP_DISABLE_FAILED")
		}
		return &AuthTotpDisableOutput{
			Body: AuthTotpDisableEnvelope{
				Data: AuthTotpDisableData{},
				Meta: AuthTotpDisableMeta{RequestID: "req_auth_totp_disable"},
			},
		}, nil
	})
}

type authTokenListBearerInput struct {
	Authorization string `header:"Authorization"`
	Limit         int    `query:"limit" default:"50" minimum:"1" maximum:"500"`
	Cursor        string `query:"cursor" maxLength:"512"`
}

func registerAuthTokenListReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenList",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/tokens",
		Summary:     "List API tokens",
		Description: "Lists API tokens belonging to the authenticated caller.",
		Tags:        []string{"Auth"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *authTokenListBearerInput) (*AuthTokenListOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		toks, err := svc.ListAPITokens(ctx, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("AUTH_TOKEN_LIST_FAILED")
		}
		records := make([]AuthTokenRecord, 0, len(toks))
		for i := range toks {
			records = append(records, apiTokenToRecord(&toks[i]))
		}
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		return &AuthTokenListOutput{
			Body: AuthTokenListEnvelope{
				Data: records,
				Meta: AuthTokenListMeta{
					RequestID: "req_auth_token_list",
					Page: AuthTokenPageMeta{
						HasNext:    false,
						NextCursor: "",
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

type authTokenCreateBearerInput struct {
	Authorization string `header:"Authorization"`
	Body          AuthTokenCreateRequest
}

func registerAuthTokenCreateReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/tokens",
		Summary:     "Create a new API token",
		Description: "Creates and returns a new API token. Plaintext token is surfaced exactly once.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{Description: "Token creation payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized, http.StatusBadRequest},
	}, func(ctx context.Context, input *authTokenCreateBearerInput) (*AuthTokenCreateOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		issued, err := svc.CreateAPIToken(ctx, userID, input.Body.Name, input.Body.Scope, 0)
		if errors.Is(err, auth.ErrInvalidScope) {
			return nil, huma.Error400BadRequest("AUTH_TOKEN_SCOPE_INVALID")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("AUTH_TOKEN_CREATE_FAILED")
		}
		return &AuthTokenCreateOutput{
			Body: AuthTokenCreateEnvelope{
				Data: AuthTokenCreateData{
					ID:    issued.ID,
					Name:  issued.Name,
					Scope: issued.Scope,
					Token: issued.Plaintext,
				},
				Meta: AuthTokenCreateMeta{RequestID: "req_auth_token_create"},
			},
		}, nil
	})
}

type authTokenRevokeBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
}

func registerAuthTokenRevokeReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenRevoke",
		Method:      http.MethodDelete,
		Path:        "/api/v1/auth/tokens/{id}",
		Summary:     "Revoke an API token",
		Description: "Revokes an owned API token immediately.",
		Tags:        []string{"Auth"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *authTokenRevokeBearerInput) (*AuthTokenRevokeOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if err := svc.RevokeAPIToken(ctx, userID, input.ID); err != nil {
			if errors.Is(err, authrepo.ErrNotFound) {
				return nil, huma.Error404NotFound("AUTH_TOKEN_NOT_FOUND")
			}
			return nil, huma.Error500InternalServerError("AUTH_TOKEN_REVOKE_FAILED")
		}
		return &AuthTokenRevokeOutput{
			Body: AuthTokenRevokeEnvelope{
				Data: AuthTokenRevokeData{},
				Meta: AuthTokenRevokeMeta{RequestID: "req_auth_token_revoke"},
			},
		}, nil
	})
}

func apiTokenToRecord(t *authrepo.APIToken) AuthTokenRecord {
	rec := AuthTokenRecord{
		ID:        t.ID,
		Name:      t.Name,
		Scope:     t.Scope,
		CreatedAt: time.Unix(t.CreatedAt, 0).UTC().Format(time.RFC3339),
	}
	if t.LastUsedAt.Valid {
		rec.LastUsed = time.Unix(t.LastUsedAt.Int64, 0).UTC().Format(time.RFC3339)
	}
	return rec
}
