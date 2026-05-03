// Package server hosts the HTTP layer of hellingd.
//
// The server wires the chi router, middleware, and generated OpenAPI
// handlers together. Business logic lives behind service interfaces;
// handlers are thin adapters between HTTP and services.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// VersionInfo is build metadata exposed via /v1/version.
type VersionInfo struct {
	Version   string
	Commit    string
	BuildTime string
}

// AuthSettings groups runtime knobs for the auth surface.
type AuthSettings struct {
	SessionTTL     time.Duration
	UsernameLimit  int
	UsernameWindow time.Duration
	IPLimit        int
	IPWindow       time.Duration
	Argon2         auth.Argon2Params
}

// IncusProber returns whether the Incus daemon is reachable. Injected so
// tests can stub the probe without hitting a real socket.
type IncusProber func(context.Context) bool

// Config wires the server's collaborators.
type Config struct {
	Store       *store.Store
	Logger      *slog.Logger
	Version     VersionInfo
	Auth        AuthSettings
	IncusProber IncusProber
	Incus       incus.Client
}

// Server is the top-level HTTP server.
type Server struct {
	cfg         Config
	router      chi.Router
	userLimiter *auth.RateLimiter
	ipLimiter   *auth.RateLimiter
}

// New constructs the server and registers routes.
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("server.New: cfg is nil")
	}
	if cfg.Store == nil {
		return nil, errors.New("server.New: Store is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("server.New: Logger is required")
	}
	if cfg.Auth.SessionTTL <= 0 {
		return nil, errors.New("server.New: Auth.SessionTTL must be > 0")
	}
	if cfg.Auth.UsernameLimit <= 0 || cfg.Auth.UsernameWindow <= 0 {
		return nil, errors.New("server.New: Auth.UsernameLimit/Window must be > 0")
	}
	if cfg.Auth.IPLimit <= 0 || cfg.Auth.IPWindow <= 0 {
		return nil, errors.New("server.New: Auth.IPLimit/Window must be > 0")
	}

	s := &Server{
		cfg:         *cfg,
		userLimiter: auth.NewRateLimiter(cfg.Auth.UsernameLimit, cfg.Auth.UsernameWindow),
		ipLimiter:   auth.NewRateLimiter(cfg.Auth.IPLimit, cfg.Auth.IPWindow),
	}
	s.router = s.routes()
	return s, nil
}

// Handler returns the chi router as an http.Handler suitable for http.Server.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()

	r.Use(requestIDMiddleware)
	r.Use(loggerMiddleware(s.cfg.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Public, unauthenticated.
	r.Get("/healthz", s.handleHealth)
	r.Get("/v1/version", s.handleVersion)

	// v1 group; auth applied where appropriate inside subroutes.
	r.Route("/v1", func(r chi.Router) {
		// Public auth endpoints.
		r.Post("/auth/login", s.handleLogin)

		// Authenticated surface.
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Post("/auth/logout", s.handleLogout)
			r.Get("/auth/me", s.handleMe)
			r.Get("/instances", s.handleListInstances)
			r.Post("/instances", s.handleCreateInstance)
			r.Get("/instances/{name}", s.handleGetInstance)
			r.Delete("/instances/{name}", s.handleDeleteInstance)
			r.Post("/instances/{name}/start", s.handleStartInstance)
			r.Post("/instances/{name}/stop", s.handleStopInstance)
			r.Get("/operations", s.handleListOperations)
			r.Get("/operations/{id}", s.handleGetOperation)
		})
	})

	r.NotFound(s.handleNotFound)
	r.MethodNotAllowed(s.handleMethodNotAllowed)

	return r
}

// ---- handlers (skeletons; real implementations land in stage 2) ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	incusReachable := false
	if s.cfg.IncusProber != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		incusReachable = s.cfg.IncusProber(ctx)
	}
	status := "ok"
	if !incusReachable {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          status,
		"incus_reachable": incusReachable,
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":     s.cfg.Version.Version,
		"api_version": "v1",
		"commit":      s.cfg.Version.Commit,
		"build_time":  s.cfg.Version.BuildTime,
	})
}

func (s *Server) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "route does not exist")
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed for this route")
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}

// ---- middleware ----

type ctxKey string

const ctxKeyRequestID ctxKey = "request_id"

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loggerMiddleware(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			base.LogAttrs(r.Context(), slog.LevelInfo, "http_request",
				slog.String("request_id", RequestIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes_out", ww.BytesWritten()),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}

// RequestIDFromContext returns the request ID set by requestIDMiddleware,
// or empty string if the context has none.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}
