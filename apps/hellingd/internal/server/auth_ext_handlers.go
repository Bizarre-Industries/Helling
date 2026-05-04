package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// ---- setup (first admin) ----

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	n, err := s.cfg.Store.CountUsers(r.Context())
	if err != nil {
		s.cfg.Logger.Error("setup: count users", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if n > 0 {
		writeError(w, http.StatusConflict, "already_setup", "admin user already exists")
		return
	}

	var req setupRequest
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
		s.cfg.Logger.Error("setup: hash password", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	u, err := s.cfg.Store.CreateUser(r.Context(), req.Username, hash, true)
	if err != nil {
		s.cfg.Logger.Error("setup: create user", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID:        u.ID,
		Username:  u.Username,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
	})
}

// ---- API tokens ----

type createTokenRequest struct {
	Name   string `json:"name"`
	Scopes string `json:"scopes"`
}

type createTokenResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token"`
	Scopes    string     `json:"scopes"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	tokens, err := s.cfg.Store.ListAPITokensByUser(r.Context(), u.ID)
	if err != nil {
		s.cfg.Logger.Error("list tokens", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]map[string]any, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, map[string]any{
			"id":           t.ID,
			"name":         t.Name,
			"scopes":       t.Scopes,
			"created_at":   t.CreatedAt,
			"expires_at":   t.ExpiresAt,
			"last_used_at": t.LastUsedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	var req createTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	if req.Scopes == "" {
		req.Scopes = auth.ScopeRead
	}
	if !auth.ValidScopes(req.Scopes) {
		writeError(w, http.StatusBadRequest, "bad_request", "scopes must be read, write, or admin")
		return
	}

	raw, hash, err := auth.NewAPIToken()
	if err != nil {
		s.cfg.Logger.Error("create token: mint", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	t, err := s.cfg.Store.CreateAPIToken(r.Context(), u.ID, req.Name, hash, req.Scopes, nil)
	if err != nil {
		s.cfg.Logger.Error("create token: store", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, createTokenResponse{
		ID:        t.ID,
		Name:      t.Name,
		Token:     raw,
		Scopes:    t.Scopes,
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
	})
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "token id required")
		return
	}
	// Verify ownership before deleting.
	tokens, err := s.cfg.Store.ListAPITokensByUser(r.Context(), u.ID)
	if err != nil {
		s.cfg.Logger.Error("revoke token: list", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	found := false
	for _, t := range tokens {
		if t.ID == id {
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, "not_found", "token not found")
		return
	}
	if err := s.cfg.Store.DeleteAPIToken(r.Context(), id); err != nil {
		s.cfg.Logger.Error("revoke token: delete", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- TOTP ----

type totpSetupResponse struct {
	Secret        string   `json:"secret"`
	KeyURI        string   `json:"key_uri"`
	RecoveryCodes []string `json:"recovery_codes"`
}

func (s *Server) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}

	// Check if TOTP is already enabled.
	if _, err := s.cfg.Store.GetTOTPSecret(r.Context(), u.ID); err == nil {
		writeError(w, http.StatusConflict, "conflict", "TOTP already configured; disable first")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		s.cfg.Logger.Error("totp setup: get secret", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	secret, err := auth.NewTOTPSecret()
	if err != nil {
		s.cfg.Logger.Error("totp setup: generate secret", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	// Store as not-yet-enabled; user must verify before it activates.
	if err := s.cfg.Store.SetTOTPSecret(r.Context(), u.ID, secret, false); err != nil {
		s.cfg.Logger.Error("totp setup: store secret", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	rawCodes, codeHashes, err := auth.NewRecoveryCodesWithParams(auth.RecoveryCodeCount, s.cfg.Auth.Argon2)
	if err != nil {
		s.cfg.Logger.Error("totp setup: recovery codes", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := s.cfg.Store.SaveRecoveryCodes(r.Context(), u.ID, codeHashes); err != nil {
		s.cfg.Logger.Error("totp setup: save recovery codes", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	keyURI := auth.TOTPKeyURI("Helling", u.Username, secret)
	writeJSON(w, http.StatusOK, totpSetupResponse{
		Secret:        secret,
		KeyURI:        keyURI,
		RecoveryCodes: rawCodes,
	})
}

type totpVerifyRequest struct {
	Code string `json:"code"`
}

func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}

	var req totpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "code is required")
		return
	}

	ts, err := s.cfg.Store.GetTOTPSecret(r.Context(), u.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "TOTP not configured; run setup first")
			return
		}
		s.cfg.Logger.Error("totp verify: get secret", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	if !auth.ValidateTOTP(ts.Secret, req.Code) {
		writeError(w, http.StatusBadRequest, "invalid_code", "invalid TOTP code")
		return
	}

	if err := s.cfg.Store.SetTOTPSecret(r.Context(), u.ID, ts.Secret, true); err != nil {
		s.cfg.Logger.Error("totp verify: enable", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTOTPDelete(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	if err := s.cfg.Store.DeleteTOTPSecret(r.Context(), u.ID); err != nil {
		s.cfg.Logger.Error("totp delete: secret", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := s.cfg.Store.DeleteRecoveryCodes(r.Context(), u.ID); err != nil {
		s.cfg.Logger.Error("totp delete: recovery codes", slog.Any("err", err))
	}
	w.WriteHeader(http.StatusNoContent)
}

type mfaCompleteRequest struct {
	MFAToken     string `json:"mfa_token"`
	Code         string `json:"code"`
	TOTPCode     string `json:"totp_code"`
	RecoveryCode string `json:"recovery_code"`
}

//nolint:gocyclo // This handler is a single auth state transition with explicit failure branches.
func (s *Server) handleMFAComplete(w http.ResponseWriter, r *http.Request) {
	var req mfaCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.MFAToken == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "mfa_token required")
		return
	}
	code := req.Code
	if code == "" {
		code = req.TOTPCode
	}
	if code == "" && req.RecoveryCode == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "code or recovery_code required")
		return
	}
	tokenHash := auth.HashToken(req.MFAToken)
	challenge, ok := s.getMFAChallenge(tokenHash)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_mfa_token", "MFA challenge expired or unknown")
		return
	}
	u, err := s.cfg.Store.GetUserByID(r.Context(), challenge.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.deleteMFAChallenge(tokenHash)
			writeError(w, http.StatusUnauthorized, "invalid_mfa_token", "user no longer exists")
			return
		}
		s.cfg.Logger.Error("mfa complete: get user", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	ts, err := s.cfg.Store.GetTOTPSecret(r.Context(), u.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.deleteMFAChallenge(tokenHash)
			writeError(w, http.StatusUnauthorized, "mfa_not_configured", "MFA is not configured")
			return
		}
		s.cfg.Logger.Error("mfa complete: get totp", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if !ts.Enabled {
		s.deleteMFAChallenge(tokenHash)
		writeError(w, http.StatusUnauthorized, "mfa_not_enabled", "MFA is not enabled")
		return
	}

	valid := false
	if code != "" {
		valid = auth.ValidateTOTP(ts.Secret, code)
	}
	if !valid && req.RecoveryCode != "" {
		consumed, err := s.cfg.Store.ConsumeRecoveryCode(r.Context(), u.ID, strings.TrimSpace(req.RecoveryCode))
		if err != nil {
			s.cfg.Logger.Error("mfa complete: consume recovery code", slog.Any("err", err))
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		valid = consumed
	}
	if !valid {
		s.deleteMFAChallenge(tokenHash)
		writeError(w, http.StatusUnauthorized, "invalid_mfa_code", "invalid MFA code")
		return
	}

	s.deleteMFAChallenge(tokenHash)
	accessToken, expiresIn, err := s.issueSession(w, r, u)
	if err != nil {
		s.cfg.Logger.Error("mfa complete: issue session", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if wantsAuthJSON(r) {
		writeAuthData(w, http.StatusOK, authTokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			MFARequired: false,
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
