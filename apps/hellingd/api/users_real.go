package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// User real handlers back the /api/v1/users* stubs with the authrepo store.
// Every endpoint requires a valid Bearer (JWT or helling_* API token). Role
// gating (admin-only for create/delete/update/set-scope) stays TODO until a
// dedicated middleware layer lands in v0.1-beta.

type userListBearerInput struct {
	Authorization string `header:"Authorization"`
	Limit         int    `query:"limit" default:"50" minimum:"1" maximum:"500"`
	Cursor        string `query:"cursor" maxLength:"512"`
}

type userCreateBearerInput struct {
	Authorization string `header:"Authorization"`
	Body          UserCreateRequest
}

type userGetBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
}

type userDeleteBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
}

type userSetScopeBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
	Body          UserSetScopeRequest
}

func registerUserListReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "userList",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List users",
		Description: "Lists users visible to the caller. Uses offset-style pagination; cursor decodes as the next offset.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *userListBearerInput) (*UserListOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		offset := 0
		if input.Cursor != "" {
			if n, err := parseIntSafe(input.Cursor); err == nil && n >= 0 {
				offset = n
			}
		}
		rows, total, err := svc.Repo().ListUsers(ctx, offset, limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("USER_LIST_FAILED")
		}
		records := make([]UserRecord, 0, len(rows))
		for i := range rows {
			records = append(records, userRowToRecord(&rows[i]))
		}
		next := offset + limit
		hasNext := next < total
		nextCursor := ""
		if hasNext {
			nextCursor = intToCursor(next)
		}
		return &UserListOutput{
			Body: UserListEnvelope{
				Data: records,
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

func registerUserCreateReal(api huma.API, svc *auth.Service, issuer CertIssuer) {
	huma.Register(api, huma.Operation{
		OperationID: "userCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/users",
		Summary:     "Create a user",
		Description: "Creates a Helling-managed account with an argon2id password hash. PAM integration lands in v0.1-beta.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{Description: "User creation payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized, http.StatusConflict, http.StatusBadRequest},
	}, func(ctx context.Context, input *userCreateBearerInput) (*UserCreateOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if input.Body.Username == "" || input.Body.Role == "" {
			return nil, huma.Error400BadRequest("USER_FIELDS_REQUIRED")
		}
		hash := ""
		if input.Body.Password != "" {
			h, err := auth.HashPassword(input.Body.Password, auth.DefaultArgon2idParams)
			if err != nil {
				return nil, huma.Error500InternalServerError("USER_HASH_FAILED")
			}
			hash = h
		}
		u, err := svc.Repo().CreateUser(ctx, input.Body.Username, input.Body.Role, hash)
		if errors.Is(err, authrepo.ErrDuplicate) {
			return nil, huma.Error409Conflict("USER_DUPLICATE_USERNAME")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("USER_CREATE_FAILED")
		}
		// Best-effort PKI issuance per ADR-024. We deliberately do not roll
		// back the user on issuance failure: the operator can re-issue from
		// the userSetScope path, and the user otherwise functions for
		// non-Incus operations.
		if issuer != nil {
			if err := issuer.IssueForUser(ctx, u.ID, u.Username); err != nil {
				return nil, huma.Error500InternalServerError("USER_CERT_ISSUE_FAILED")
			}
		}
		return &UserCreateOutput{
			Body: UserCreateEnvelope{
				Data: UserCreateData{
					ID: u.ID, Username: u.Username, Role: u.Role, Status: u.Status,
				},
				Meta: UserCreateMeta{RequestID: "req_user_create"},
			},
		}, nil
	})
}

func registerUserGetReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "userGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/{id}",
		Summary:     "Get a user",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *userGetBearerInput) (*UserGetOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		u, err := svc.Repo().GetUserByID(ctx, input.ID)
		if errors.Is(err, authrepo.ErrNotFound) {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("USER_GET_FAILED")
		}
		totp, terr := svc.Repo().GetTOTPSecret(ctx, u.ID)
		totpEnabled := terr == nil && totp.Enabled
		return &UserGetOutput{
			Body: UserGetEnvelope{
				Data: UserGetData{
					ID: u.ID, Username: u.Username, Role: u.Role, Status: u.Status,
					TotpEnabled: totpEnabled,
				},
				Meta: UserGetMeta{RequestID: "req_user_get"},
			},
		}, nil
	})
}

func registerUserDeleteReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "userDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/users/{id}",
		Summary:     "Delete a user",
		Description: "Removes the users row. ON DELETE CASCADE removes sessions, API tokens, TOTP secrets, and recovery codes per docs/spec/sqlite-schema.md §7.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *userDeleteBearerInput) (*UserDeleteOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if _, err := svc.Repo().GetUserByID(ctx, input.ID); errors.Is(err, authrepo.ErrNotFound) {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		if err := svc.Repo().DeleteUser(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("USER_DELETE_FAILED")
		}
		return &UserDeleteOutput{
			Body: UserDeleteEnvelope{
				Data: UserDeleteData{},
				Meta: UserDeleteMeta{RequestID: "req_user_delete"},
			},
		}, nil
	})
}

func registerUserSetScopeReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "userSetScope",
		Method:      http.MethodPut,
		Path:        "/api/v1/users/{id}/scope",
		Summary:     "Set a user's Incus trust scope",
		Description: "Records the requested Incus trust scope. Per-user certificate rotation lands alongside the internal CA in v0.1-beta.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{Description: "Scope assignment.", Required: true},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusBadRequest},
	}, func(ctx context.Context, input *userSetScopeBearerInput) (*UserSetScopeOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		switch input.Body.Scope {
		case "default", "restricted", "admin":
		default:
			return nil, huma.Error400BadRequest("USER_SCOPE_INVALID")
		}
		if _, err := svc.Repo().GetUserByID(ctx, input.ID); errors.Is(err, authrepo.ErrNotFound) {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		if err := svc.Repo().SetUserScope(ctx, input.ID, input.Body.Scope); err != nil {
			return nil, huma.Error500InternalServerError("USER_SET_SCOPE_FAILED")
		}
		return &UserSetScopeOutput{
			Body: UserSetScopeEnvelope{
				Data: UserSetScopeData{ID: input.ID, Scope: input.Body.Scope},
				Meta: UserSetScopeMeta{RequestID: "req_user_set_scope"},
			},
		}, nil
	})
}

func userRowToRecord(u *authrepo.User) UserRecord {
	return UserRecord{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		Status:   u.Status,
	}
}

func parseIntSafe(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not an integer")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func intToCursor(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
