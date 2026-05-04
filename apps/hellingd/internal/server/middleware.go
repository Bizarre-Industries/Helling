package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

const (
	ctxKeyUser           ctxKey = "user"
	ctxKeyAPITokenScopes ctxKey = "api_token_scopes"
)

// UserFromContext returns the authenticated user attached by authMiddleware,
// or (zero, false) if the context has none.
func UserFromContext(ctx context.Context) (store.User, bool) {
	v, ok := ctx.Value(ctxKeyUser).(store.User)
	return v, ok
}

func apiTokenScopesFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyAPITokenScopes).(string)
	return v, ok
}

// authMiddleware looks up the session cookie, validates it, and attaches the
// user to the request context. Rejects with 401 when the cookie is missing,
// the session has expired, or the user has been deleted.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err == nil && cookie.Value != "" {
			hash := auth.HashToken(cookie.Value)
			sess, err := s.cfg.Store.GetSessionByTokenHash(r.Context(), hash)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeError(w, http.StatusUnauthorized, "no_session", "session expired or unknown")
					return
				}
				s.cfg.Logger.Error("authMiddleware: get session", slog.Any("err", err))
				writeError(w, http.StatusInternalServerError, "internal", "internal error")
				return
			}
			u, err := s.cfg.Store.GetUserByID(r.Context(), sess.UserID)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					_ = s.cfg.Store.DeleteSession(r.Context(), hash)
					writeError(w, http.StatusUnauthorized, "no_session", "user no longer exists")
					return
				}
				s.cfg.Logger.Error("authMiddleware: get user", slog.Any("err", err))
				writeError(w, http.StatusInternalServerError, "internal", "internal error")
				return
			}

			// Best-effort sliding-session refresh. Failure does not block the request.
			if err := s.cfg.Store.TouchSession(r.Context(), hash); err != nil {
				s.cfg.Logger.Warn("authMiddleware: touch session", slog.Any("err", err))
			}

			ctx := context.WithValue(r.Context(), ctxKeyUser, u)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if u, scopes, attempted, ok := s.authenticateBearer(w, r); attempted {
			if !ok {
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyUser, u)
			if scopes != "" {
				ctx = context.WithValue(ctx, ctxKeyAPITokenScopes, scopes)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		writeError(w, http.StatusUnauthorized, "no_session", "authentication required")
	})
}

func (s *Server) authenticateBearer(w http.ResponseWriter, r *http.Request) (u store.User, scopes string, attempted bool, ok bool) {
	header := r.Header.Get("Authorization")
	raw, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || strings.TrimSpace(raw) == "" {
		return store.User{}, "", false, false
	}
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, auth.APITokenPrefix) {
		return s.authenticateAPIToken(w, r, raw)
	}
	if s.cfg.Auth.JWTSigner == nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", "bearer token rejected")
		return store.User{}, "", true, false
	}
	claims, err := s.cfg.Auth.JWTSigner.Verify(raw)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", "bearer token rejected")
		return store.User{}, "", true, false
	}
	u, err = s.cfg.Store.GetUserByID(r.Context(), claims.UserID())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid_token", "user no longer exists")
			return store.User{}, "", true, false
		}
		s.cfg.Logger.Error("authMiddleware: get bearer user", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return store.User{}, "", true, false
	}
	return u, claims.Scopes, true, true
}

func (s *Server) authenticateAPIToken(w http.ResponseWriter, r *http.Request, raw string) (u store.User, scopes string, attempted bool, ok bool) {
	hash := auth.HashAPIToken(raw)
	tok, err := s.cfg.Store.GetAPITokenByHash(r.Context(), hash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid_token", "API token rejected")
			return store.User{}, "", true, false
		}
		s.cfg.Logger.Error("authMiddleware: get api token", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return store.User{}, "", true, false
	}
	if !auth.ValidScopes(tok.Scopes) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "API token rejected")
		return store.User{}, "", true, false
	}
	if tok.ExpiresAt != nil && !time.Now().UTC().Before(*tok.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "invalid_token", "API token expired")
		return store.User{}, "", true, false
	}
	u, err = s.cfg.Store.GetUserByID(r.Context(), tok.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid_token", "user no longer exists")
			return store.User{}, "", true, false
		}
		s.cfg.Logger.Error("authMiddleware: get api token user", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return store.User{}, "", true, false
	}
	if err := s.cfg.Store.TouchAPIToken(r.Context(), hash); err != nil {
		s.cfg.Logger.Warn("authMiddleware: touch api token", slog.Any("err", err))
	}
	return u, tok.Scopes, true, true
}

func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "no_session", "authentication required")
			return
		}
		if scopes, fromToken := apiTokenScopesFromContext(r.Context()); fromToken && scopes != auth.ScopeAdmin {
			writeError(w, http.StatusForbidden, "forbidden", "admin API token scope required")
			return
		}
		if !u.IsAdmin {
			writeError(w, http.StatusForbidden, "forbidden", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeScopeMiddleware(next http.Handler) http.Handler {
	return s.tokenScopeMiddleware(auth.ScopeWrite)(next)
}

func (s *Server) tokenScopeMiddleware(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, ok := apiTokenScopesFromContext(r.Context())
			if ok && !scopeAllows(scopes, required) {
				writeError(w, http.StatusForbidden, "forbidden", required+" API token scope required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func scopeAllows(have, required string) bool {
	switch have {
	case auth.ScopeAdmin:
		return true
	case auth.ScopeWrite:
		return required == auth.ScopeRead || required == auth.ScopeWrite
	case auth.ScopeRead:
		return required == auth.ScopeRead
	default:
		return false
	}
}
