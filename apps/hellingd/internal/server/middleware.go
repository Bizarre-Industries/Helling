package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

const ctxKeyUser ctxKey = "user"

// UserFromContext returns the authenticated user attached by authMiddleware,
// or (zero, false) if the context has none.
func UserFromContext(ctx context.Context) (store.User, bool) {
	v, ok := ctx.Value(ctxKeyUser).(store.User)
	return v, ok
}

// authMiddleware looks up the session cookie, validates it, and attaches the
// user to the request context. Rejects with 401 when the cookie is missing,
// the session has expired, or the user has been deleted.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "no_session", "authentication required")
			return
		}
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
	})
}
