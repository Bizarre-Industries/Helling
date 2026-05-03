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

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// VersionInfo is build metadata exposed via /v1/version.
type VersionInfo struct {
	Version   string
	Commit    string
	BuildTime string
}

// Config wires the server's collaborators.
type Config struct {
	Store   *store.Store
	Logger  *slog.Logger
	Version VersionInfo
}

// Server is the top-level HTTP server.
type Server struct {
	cfg    Config
	router chi.Router
}

// New constructs the server and registers routes.
func New(cfg Config) (*Server, error) {
	if cfg.Store == nil {
		return nil, errors.New("server.New: Store is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("server.New: Logger is required")
	}

	s := &Server{cfg: cfg}
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
		r.Post("/auth/login", s.handleNotImplemented)
		r.Post("/auth/logout", s.handleNotImplemented)

		r.Group(func(r chi.Router) {
			// TODO(suhail, 2026-05-02): wire authMiddleware here once sessions exist.
			r.Get("/auth/me", s.handleNotImplemented)
			r.Get("/instances", s.handleNotImplemented)
			r.Post("/instances", s.handleNotImplemented)
			r.Get("/instances/{name}", s.handleNotImplemented)
			r.Delete("/instances/{name}", s.handleNotImplemented)
			r.Post("/instances/{name}/start", s.handleNotImplemented)
			r.Post("/instances/{name}/stop", s.handleNotImplemented)
			r.Get("/operations", s.handleNotImplemented)
			r.Get("/operations/{id}", s.handleNotImplemented)
		})
	})

	r.NotFound(s.handleNotFound)
	r.MethodNotAllowed(s.handleMethodNotAllowed)

	return r
}

// ---- handlers (skeletons; real implementations land in stage 2) ----

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":     s.cfg.Version.Version,
		"api_version": "v1",
		"commit":      s.cfg.Version.Commit,
		"build_time":  s.cfg.Version.BuildTime,
	})
}

func (s *Server) handleNotImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "endpoint pending implementation")
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
