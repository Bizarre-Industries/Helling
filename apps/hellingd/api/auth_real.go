package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

// refreshCookieName is the httpOnly cookie used for refresh tokens. Matches
// docs/spec/auth.md §2.2. The cookie carries the opaque refresh token; the
// server stores only its SHA-256 hash.
const refreshCookieName = "helling_refresh"

type authSetupCookieInput struct {
	UserAgent string `header:"User-Agent"`
	XRealIP   string `header:"X-Real-IP"`
	Body      AuthSetupRequest
}

type authLoginCookieInput struct {
	UserAgent string `header:"User-Agent"`
	XRealIP   string `header:"X-Real-IP"`
	Body      AuthLoginRequest
}

// authRefreshCookieBody accepts refresh_token optionally; cookie is the
// primary transport per docs/spec/auth.md §2.2, so a body-less request with
// a valid cookie still succeeds.
type authRefreshCookieBody struct {
	RefreshToken string `json:"refresh_token,omitempty" maxLength:"4096" doc:"Opaque refresh token. Optional when helling_refresh cookie is present."`
}

type authRefreshCookieInput struct {
	UserAgent     string `header:"User-Agent"`
	XRealIP       string `header:"X-Real-IP"`
	RefreshCookie string `cookie:"helling_refresh"`
	Body          authRefreshCookieBody
}

type authLogoutCookieInput struct {
	UserAgent     string `header:"User-Agent"`
	XRealIP       string `header:"X-Real-IP"`
	RefreshCookie string `cookie:"helling_refresh"`
}

type authLoginCookieOutput struct {
	Status    int    `status:"200"`
	SetCookie string `header:"Set-Cookie"`
	Body      AuthLoginEnvelope
}

type authRefreshCookieOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      AuthRefreshEnvelope
}

type authLogoutCookieOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      AuthLogoutEnvelope
}

type authSetupCookieOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      AuthSetupEnvelope
}

func refreshCookieFor(token string, maxAgeSeconds int) string {
	return fmt.Sprintf(
		"%s=%s; Path=/; Max-Age=%d; HttpOnly; Secure; SameSite=Strict",
		refreshCookieName, token, maxAgeSeconds,
	)
}

func refreshClearCookie() string {
	return refreshCookieName + "=; Path=/; Max-Age=0; HttpOnly; Secure; SameSite=Strict"
}

func registerAuthSetupReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authSetup",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/setup",
		Summary:     "Create the initial admin account",
		Description: "Bootstraps the first administrator on a fresh install. Returns 409 when an admin already exists.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{Description: "Initial admin credentials.", Required: true},
		Errors:      []int{http.StatusConflict},
	}, func(ctx context.Context, input *authSetupCookieInput) (*authSetupCookieOutput, error) {
		ident, err := svc.Setup(ctx, input.Body.Username, input.Body.Password, input.XRealIP, input.UserAgent)
		if errors.Is(err, auth.ErrSetupNotRequired) {
			return nil, huma.Error409Conflict("AUTH_ALREADY_SET_UP")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("AUTH_SETUP_FAILED")
		}
		return &authSetupCookieOutput{
			SetCookie: refreshCookieFor(ident.RefreshToken, int(svc.Signer().RefreshTTL().Seconds())),
			Body: AuthSetupEnvelope{
				Data: AuthSetupData{
					AccessToken: ident.AccessToken,
					TokenType:   "Bearer",
					ExpiresIn:   ident.AccessExpires,
				},
				Meta: AuthSetupMeta{RequestID: "req_auth_setup"},
			},
		}, nil
	})
}

func registerAuthLoginReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authLogin",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Authenticate and issue JWT pair",
		Description: "Verifies credentials. Returns an access token + helling_refresh cookie for TOTP-disabled accounts, or an MFA challenge (202) for accounts with an active factor.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{Description: "Credentials with optional inline TOTP code.", Required: true},
		Errors:      []int{http.StatusUnauthorized, http.StatusTooManyRequests},
	}, func(ctx context.Context, input *authLoginCookieInput) (*authLoginCookieOutput, error) {
		// If TOTP code inlined, try the classic one-call path: login + MFA in a single request.
		if input.Body.TOTPCode != "" {
			res, err := svc.LoginWithMFA(ctx, input.Body.Username, input.Body.Password, input.XRealIP, input.UserAgent)
			if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrUserDisabled) {
				return nil, huma.Error401Unauthorized("AUTH_INVALID_CREDENTIALS")
			}
			if err != nil {
				return nil, huma.Error500InternalServerError("AUTH_LOGIN_FAILED")
			}
			if res.Pending != nil {
				ident, err := svc.CompleteMFA(ctx, res.Pending.MFAToken, input.Body.TOTPCode, input.XRealIP, input.UserAgent)
				if err != nil {
					return nil, huma.Error401Unauthorized("AUTH_MFA_CODE_INVALID")
				}
				return loginOK(svc, &ident), nil
			}
			return loginOK(svc, res.Identity), nil
		}

		res, err := svc.LoginWithMFA(ctx, input.Body.Username, input.Body.Password, input.XRealIP, input.UserAgent)
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrUserDisabled) {
			return nil, huma.Error401Unauthorized("AUTH_INVALID_CREDENTIALS")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("AUTH_LOGIN_FAILED")
		}
		if res.Pending != nil {
			return &authLoginCookieOutput{
				Status: http.StatusAccepted,
				Body: AuthLoginEnvelope{
					Data: AuthLoginData{
						MFARequired: true,
						MFAToken:    res.Pending.MFAToken,
					},
					Meta: AuthLoginMeta{RequestID: "req_auth_login_mfa"},
				},
			}, nil
		}
		return loginOK(svc, res.Identity), nil
	})
}

// loginOK builds a 200 login response from a fully-issued Identity.
func loginOK(svc *auth.Service, ident *auth.Identity) *authLoginCookieOutput {
	return &authLoginCookieOutput{
		Status:    http.StatusOK,
		SetCookie: refreshCookieFor(ident.RefreshToken, int(svc.Signer().RefreshTTL().Seconds())),
		Body: AuthLoginEnvelope{
			Data: AuthLoginData{
				AccessToken: ident.AccessToken,
				TokenType:   "Bearer",
				ExpiresIn:   ident.AccessExpires,
			},
			Meta: AuthLoginMeta{RequestID: "req_auth_login_ok"},
		},
	}
}

func registerAuthRefreshReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authRefresh",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Rotate the refresh token and issue a new access token",
		Description: "Accepts refresh_token from the body (v0.1-alpha) or the helling_refresh cookie (v0.1-beta+). Rotates on success.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{Description: "Refresh token exchange payload.", Required: false},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *authRefreshCookieInput) (*authRefreshCookieOutput, error) {
		token := input.Body.RefreshToken
		if token == "" {
			token = input.RefreshCookie
		}
		ident, err := svc.Refresh(ctx, token, input.XRealIP, input.UserAgent)
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrUserDisabled) {
			return nil, huma.Error401Unauthorized("AUTH_REFRESH_INVALID")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("AUTH_REFRESH_FAILED")
		}
		return &authRefreshCookieOutput{
			SetCookie: refreshCookieFor(ident.RefreshToken, int(svc.Signer().RefreshTTL().Seconds())),
			Body: AuthRefreshEnvelope{
				Data: AuthRefreshData{
					AccessToken: ident.AccessToken,
					TokenType:   "Bearer",
					ExpiresIn:   ident.AccessExpires,
				},
				Meta: AuthRefreshMeta{RequestID: "req_auth_refresh"},
			},
		}, nil
	})
}

func registerAuthLogoutReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "authLogout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Revoke the current session",
		Description: "Invalidates the caller's refresh token and clears the helling_refresh cookie.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *authLogoutCookieInput) (*authLogoutCookieOutput, error) {
		if input.RefreshCookie != "" {
			_ = svc.Logout(ctx, input.RefreshCookie, input.XRealIP, input.UserAgent)
		}
		return &authLogoutCookieOutput{
			SetCookie: refreshClearCookie(),
			Body: AuthLogoutEnvelope{
				Data: AuthLogoutData{},
				Meta: AuthLogoutMeta{RequestID: "req_auth_logout"},
			},
		}, nil
	})
}
