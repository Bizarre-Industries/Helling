package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// CookieName is the name of the session cookie issued by /auth/login.
const CookieName = "helling_session"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authTokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
	MFARequired bool   `json:"mfa_required"`
	MFAToken    string `json:"mfa_token,omitempty"`
}

type mfaChallenge struct {
	UserID    int64
	ExpiresAt time.Time
}

type userResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

//nolint:gocyclo // Login intentionally spells out rate-limit, password, TOTP, and session branches.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "username and password required")
		return
	}

	ip := clientIP(r)
	if !s.userLimiter.Allow(req.Username) || !s.ipLimiter.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts; try again later")
		return
	}

	u, err := s.cfg.Store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
			return
		}
		s.cfg.Logger.Error("login: get user", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	ok, err := auth.Verify(u.PasswordHash, req.Password)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}

	totpSecret, err := s.cfg.Store.GetTOTPSecret(r.Context(), u.ID)
	switch {
	case err == nil && totpSecret.Enabled:
		raw, hash, err := auth.NewToken()
		if err != nil {
			s.cfg.Logger.Error("login: mfa token mint", slog.Any("err", err))
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		s.storeMFAChallenge(hash, mfaChallenge{
			UserID:    u.ID,
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		})
		writeAuthData(w, http.StatusAccepted, authTokenResponse{
			MFARequired: true,
			MFAToken:    raw,
		})
		return
	case err != nil && !errors.Is(err, store.ErrNotFound):
		s.cfg.Logger.Error("login: get totp", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	// Login succeeded: clear failure counters for this username.
	s.userLimiter.Reset(req.Username)

	accessToken, expiresIn, err := s.issueSession(w, r, u)
	if err != nil {
		s.cfg.Logger.Error("login: issue session", slog.Any("err", err))
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

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u store.User) (accessToken string, expiresIn int64, err error) {
	raw, hash, err := auth.NewToken()
	if err != nil {
		return "", 0, err
	}
	if _, err := s.cfg.Store.CreateSession(r.Context(), hash, u.ID, s.cfg.Auth.SessionTTL); err != nil {
		return "", 0, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    raw,
		Path:     "/",
		MaxAge:   int(s.cfg.Auth.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	if s.cfg.Auth.JWTSigner == nil {
		return "", 0, nil
	}
	ttl := s.cfg.Auth.AccessTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	accessToken, err = s.cfg.Auth.JWTSigner.NewAccessToken(u.ID, u.Username, u.IsAdmin, auth.ScopeWrite, ttl)
	if err != nil {
		return "", 0, err
	}
	return accessToken, int64(ttl.Seconds()), nil
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "no_session", "no session cookie")
		return
	}
	hash := auth.HashToken(cookie.Value)
	if err := s.cfg.Store.DeleteSession(r.Context(), hash); err != nil {
		s.cfg.Logger.Error("logout: delete session", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	writeJSON(w, http.StatusOK, userResponse{
		ID:        u.ID,
		Username:  u.Username,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
	})
}

func writeAuthData(w http.ResponseWriter, status int, data authTokenResponse) {
	writeJSON(w, status, map[string]any{"data": data})
}

func wantsAuthJSON(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/v1/")
}

func (s *Server) storeMFAChallenge(tokenHash string, challenge mfaChallenge) {
	s.mfaMu.Lock()
	defer s.mfaMu.Unlock()
	now := time.Now().UTC()
	for hash, existing := range s.mfaTokens {
		if now.After(existing.ExpiresAt) {
			delete(s.mfaTokens, hash)
		}
	}
	s.mfaTokens[tokenHash] = challenge
}

func (s *Server) getMFAChallenge(tokenHash string) (mfaChallenge, bool) {
	s.mfaMu.Lock()
	defer s.mfaMu.Unlock()
	challenge, ok := s.mfaTokens[tokenHash]
	if !ok {
		return mfaChallenge{}, false
	}
	if time.Now().UTC().After(challenge.ExpiresAt) {
		delete(s.mfaTokens, tokenHash)
		return mfaChallenge{}, false
	}
	return challenge, true
}

func (s *Server) deleteMFAChallenge(tokenHash string) {
	s.mfaMu.Lock()
	defer s.mfaMu.Unlock()
	delete(s.mfaTokens, tokenHash)
}

// clientIP returns a best-effort source IP. Strips port; honors no proxy
// headers (hellingd is socket-only; helling-proxy is the only client).
func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > -1 {
		host = host[:i]
	}
	if host == "" {
		return "unknown"
	}
	return host
}
