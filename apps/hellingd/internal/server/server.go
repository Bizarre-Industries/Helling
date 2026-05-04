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
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/proxy"
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
	AccessTTL      time.Duration
	UsernameLimit  int
	UsernameWindow time.Duration
	IPLimit        int
	IPWindow       time.Duration
	SetupTokenPath string
	Argon2         auth.Argon2Params
	JWTSigner      *auth.JWTSigner
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
	IncusProxy  *proxy.IncusProxy
	PodmanProxy *proxy.PodmanProxy
}

// Server is the top-level HTTP server.
type Server struct {
	cfg         Config
	router      chi.Router
	userLimiter *auth.RateLimiter
	ipLimiter   *auth.RateLimiter
	setup       firstAdminSetupService
	mfaMu       sync.Mutex
	mfaTokens   map[string]mfaChallenge
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
		setup:       newFirstAdminSetupService(cfg.Store, cfg.Auth.Argon2, cfg.Auth.SetupTokenPath),
		mfaTokens:   make(map[string]mfaChallenge),
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

	r.Route("/v1", s.registerV1Routes)
	r.Route("/api/v1", s.registerV1Routes)

	// Proxy pass-through to Incus and Podman Unix sockets (ADR-014).
	// Admin-only until ADR-024 per-user Incus mTLS is available in this daemon.
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.adminMiddleware)
		if s.cfg.IncusProxy != nil {
			r.Handle("/api/incus/*", s.cfg.IncusProxy)
		}
		if s.cfg.PodmanProxy != nil {
			r.Handle("/api/podman/*", s.cfg.PodmanProxy)
		}
	})

	r.NotFound(s.handleNotFound)
	r.MethodNotAllowed(s.handleMethodNotAllowed)

	return r
}

func (s *Server) registerV1Routes(r chi.Router) {
	// Public auth endpoints.
	r.Post("/auth/login", s.handleLogin)
	r.Get("/auth/setup/status", s.handleSetupStatus)
	r.Post("/auth/setup", s.handleSetup)
	r.Post("/auth/mfa/complete", s.handleMFAComplete)

	// Authenticated surface.
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Auth.
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)
		r.Post("/auth/totp/setup", s.handleTOTPSetup)
		r.Post("/auth/totp/verify", s.handleTOTPVerify)
		r.Delete("/auth/totp", s.handleTOTPDelete)
		r.Get("/auth/tokens", s.handleListTokens)
		r.Post("/auth/tokens", s.handleCreateToken)
		r.Delete("/auth/tokens/{id}", s.handleRevokeToken)

		// Instances.
		r.Get("/instances", s.handleListInstances)
		r.With(s.writeScopeMiddleware).Post("/instances", s.handleCreateInstance)
		r.Get("/instances/{name}", s.handleGetInstance)
		r.With(s.writeScopeMiddleware).Delete("/instances/{name}", s.handleDeleteInstance)
		r.With(s.writeScopeMiddleware).Post("/instances/{name}/start", s.handleStartInstance)
		r.With(s.writeScopeMiddleware).Post("/instances/{name}/stop", s.handleStopInstance)

		// Operations.
		r.Get("/operations", s.handleListOperations)
		r.Get("/operations/{id}", s.handleGetOperation)

		// System info and diagnostics are read-only.
		r.Get("/system/info", s.handleSystemInfo)
		r.Get("/system/hardware", s.handleSystemHardware)
		r.Get("/system/diagnostics", s.handleSystemDiagnostics)

		// Events.
		r.Get("/events", s.handleEvents)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.adminMiddleware)

		// Users.
		r.Get("/users", s.handleListUsers)
		r.Post("/users", s.handleCreateUser)
		r.Get("/users/{id}", s.handleGetUser)
		r.Put("/users/{id}", s.handleUpdateUser)
		r.Delete("/users/{id}", s.handleDeleteUser)

		// Privileged system config and upgrade.
		r.Get("/system/config", s.handleSystemConfig)
		r.Put("/system/config", s.handleSystemConfigUpdate)
		r.Post("/system/upgrade", s.handleSystemUpgrade)

		// Deferred privileged surfaces.
		r.Get("/schedules", s.handleListSchedules)
		r.Post("/schedules", s.handleCreateSchedule)
		r.Get("/schedules/{id}", s.handleGetSchedule)
		r.Put("/schedules/{id}", s.handleUpdateSchedule)
		r.Delete("/schedules/{id}", s.handleDeleteSchedule)
		r.Post("/schedules/{id}/run", s.handleRunSchedule)

		r.Get("/webhooks", s.handleListWebhooks)
		r.Post("/webhooks", s.handleCreateWebhook)
		r.Get("/webhooks/{id}", s.handleGetWebhook)
		r.Put("/webhooks/{id}", s.handleUpdateWebhook)
		r.Delete("/webhooks/{id}", s.handleDeleteWebhook)
		r.Post("/webhooks/{id}/test", s.handleTestWebhook)

		r.Get("/bmc", s.handleListBMC)
		r.Post("/bmc", s.handleCreateBMC)
		r.Get("/bmc/{id}", s.handleGetBMC)
		r.Delete("/bmc/{id}", s.handleDeleteBMC)
		r.Post("/bmc/{id}/power", s.handleBMCPower)
		r.Get("/bmc/{id}/sensors", s.handleBMCSensors)
		r.Get("/bmc/{id}/sel", s.handleBMCSEL)

		r.Get("/kubernetes", s.handleListK8s)
		r.Post("/kubernetes", s.handleCreateK8s)
		r.Get("/kubernetes/{name}", s.handleGetK8s)
		r.Delete("/kubernetes/{name}", s.handleDeleteK8s)
		r.Post("/kubernetes/{name}/scale", s.handleScaleK8s)
		r.Post("/kubernetes/{name}/upgrade", s.handleUpgradeK8s)
		r.Get("/kubernetes/{name}/kubeconfig", s.handleK8sKubeconfig)

		r.Get("/firewall/host", s.handleListFirewallRules)
		r.Post("/firewall/host", s.handleCreateFirewallRule)
		r.Delete("/firewall/host/{id}", s.handleDeleteFirewallRule)

		r.Get("/audit", s.handleAuditQuery)
		r.Get("/audit/export", s.handleAuditExport)

		r.Get("/notifications/channels", s.handleListNotificationChannels)
		r.Post("/notifications/channels", s.handleCreateNotificationChannel)
		r.Delete("/notifications/channels/{id}", s.handleDeleteNotificationChannel)
		r.Post("/notifications/channels/{id}/test", s.handleTestNotificationChannel)
	})
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
			base.LogAttrs(
				r.Context(), slog.LevelInfo, "http_request",
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
