package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

type createUserRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	IsAdmin      bool   `json:"is_admin"`
	IncusProject string `json:"incus_project"`
}

type updateUserRequest struct {
	Password *string `json:"password"`
	IsAdmin  *bool   `json:"is_admin"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.cfg.Store.ListUsers(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list users", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, userResponse{
			ID:           u.ID,
			Username:     u.Username,
			IsAdmin:      u.IsAdmin,
			IncusProject: u.IncusProject,
			CreatedAt:    u.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "username and password required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "bad_request", "password must be at least 8 characters")
		return
	}

	hash, err := auth.Hash(req.Password, s.cfg.Auth.Argon2)
	if err != nil {
		s.cfg.Logger.Error("create user: hash", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	u, err := s.cfg.Store.CreateUser(r.Context(), req.Username, hash, req.IsAdmin)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "conflict", "username already exists")
			return
		}
		s.cfg.Logger.Error("create user: store", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	if req.IncusProject != "" {
		project := normalizeIncusProject(req.IncusProject)
		if !incus.ValidTrustProjectName(project) {
			_ = s.cfg.Store.DeleteUser(context.WithoutCancel(r.Context()), u.ID)
			writeError(w, http.StatusBadRequest, "bad_request", "invalid incus_project")
			return
		}
		if err := s.cfg.Store.SetUserIncusProject(r.Context(), u.ID, project); err != nil {
			s.cfg.Logger.Error("create user: set incus project", slog.Any("err", err))
			_ = s.cfg.Store.DeleteUser(context.WithoutCancel(r.Context()), u.ID)
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		u.IncusProject = project
	}
	if err := s.provisionIncusTrust(r.Context(), &u); err != nil {
		_ = s.cfg.Store.DeleteUser(context.WithoutCancel(r.Context()), u.ID)
		s.cfg.Logger.Error("create user: provision incus trust", slog.Any("err", err), slog.Int64("user", u.ID))
		writeError(w, http.StatusInternalServerError, "internal", "could not provision Incus project scope")
		return
	}

	s.audit(r, "user.create", outcomeSuccess, "user", strconv.FormatInt(u.ID, 10), "user created")
	writeJSON(w, http.StatusCreated, map[string]any{"data": userResponse{
		ID:           u.ID,
		Username:     u.Username,
		IsAdmin:      u.IsAdmin,
		IncusProject: u.IncusProject,
		CreatedAt:    u.CreatedAt,
	}})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	u, err := s.getUserFromIDParam(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		s.cfg.Logger.Error("get user", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": userResponse{
		ID:           u.ID,
		Username:     u.Username,
		IsAdmin:      u.IsAdmin,
		IncusProject: u.IncusProject,
		CreatedAt:    u.CreatedAt,
	}})
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.getUserFromIDParam(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		s.cfg.Logger.Error("update user: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	passwordHash := u.PasswordHash
	if req.Password != nil {
		if len(*req.Password) < 8 {
			writeError(w, http.StatusBadRequest, "bad_request", "password must be at least 8 characters")
			return
		}
		hash, hashErr := auth.Hash(*req.Password, s.cfg.Auth.Argon2)
		if hashErr != nil {
			s.cfg.Logger.Error("update user: hash", slog.Any("err", hashErr))
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		passwordHash = hash
	}

	isAdmin := u.IsAdmin
	if req.IsAdmin != nil {
		isAdmin = *req.IsAdmin
	}

	if err := s.cfg.Store.UpdateUser(r.Context(), u.ID, passwordHash, isAdmin); err != nil {
		s.cfg.Logger.Error("update user: store", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	s.audit(r, "user.update", outcomeSuccess, "user", strconv.FormatInt(u.ID, 10), "user updated")
	writeJSON(w, http.StatusOK, map[string]any{"data": userResponse{
		ID:           u.ID,
		Username:     u.Username,
		IsAdmin:      isAdmin,
		IncusProject: u.IncusProject,
		CreatedAt:    u.CreatedAt,
	}})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.getUserFromIDParam(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		s.cfg.Logger.Error("delete user: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := s.revokeIncusTrust(r.Context(), &u); err != nil {
		s.cfg.Logger.Error("delete user: revoke incus trust", slog.Any("err", err), slog.Int64("user", u.ID))
		writeError(w, http.StatusInternalServerError, "internal", "could not revoke Incus project scope")
		return
	}
	if err := s.cfg.Store.DeleteUser(r.Context(), u.ID); err != nil {
		s.cfg.Logger.Error("delete user: store", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	s.audit(r, "user.delete", outcomeSuccess, "user", strconv.FormatInt(u.ID, 10), "user deleted")
	w.WriteHeader(http.StatusNoContent)
}

type userScopeRequest struct {
	Scope string `json:"scope"`
}

func (s *Server) handleUpdateUserScope(w http.ResponseWriter, r *http.Request) {
	u, err := s.getUserFromIDParam(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		s.cfg.Logger.Error("update user scope: get", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	var req userScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	project := normalizeIncusProject(req.Scope)
	if project == "" || !incus.ValidTrustProjectName(project) {
		writeError(w, http.StatusBadRequest, "bad_request", "valid scope is required")
		return
	}
	u.IncusProject = project
	if err := s.provisionIncusTrust(r.Context(), &u); err != nil {
		s.cfg.Logger.Error("update user scope: provision incus trust", slog.Any("err", err), slog.Int64("user", u.ID))
		writeError(w, http.StatusInternalServerError, "internal", "could not provision Incus project scope")
		return
	}
	s.audit(r, "user.scope.update", outcomeSuccess, "user", strconv.FormatInt(u.ID, 10), "user Incus project scope updated")
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"scope": "incus:project:" + project}})
}

func (s *Server) getUserFromIDParam(ctx context.Context, id string) (store.User, error) {
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return s.cfg.Store.GetUserByID(ctx, n)
	}
	return s.cfg.Store.GetUserByUsername(ctx, id)
}

func normalizeIncusProject(scope string) string {
	scope = strings.TrimSpace(scope)
	scope = strings.TrimPrefix(scope, "incus:project:")
	scope = strings.TrimPrefix(scope, "incus:")
	return scope
}

func (s *Server) provisionIncusTrust(ctx context.Context, u *store.User) error {
	if s.cfg.IncusTrust == nil {
		return nil
	}
	return s.cfg.IncusTrust.Provision(ctx, u)
}

func (s *Server) revokeIncusTrust(ctx context.Context, u *store.User) error {
	if s.cfg.IncusTrust == nil {
		return nil
	}
	return s.cfg.IncusTrust.Revoke(ctx, u)
}
