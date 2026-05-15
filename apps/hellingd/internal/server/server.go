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

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/audit"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/proxy"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/systemd"
)

// VersionInfo is build metadata exposed via /v1/version.
type VersionInfo struct {
	Version   string
	Commit    string
	BuildTime string
}

// AuthSettings groups runtime knobs for the auth surface.
type AuthSettings struct {
	SessionTTL        time.Duration
	AccessTTL         time.Duration
	UsernameLimit     int
	UsernameWindow    time.Duration
	IPLimit           int
	IPWindow          time.Duration
	SetupTokenPath    string
	ScheduleTokenPath string
	Argon2            auth.Argon2Params
	JWTSigner         *auth.JWTSigner
}

// IncusProber returns whether the Incus daemon is reachable. Injected so
// tests can stub the probe without hitting a real socket.
type IncusProber func(context.Context) bool

// IncusMetricsScraper scrapes Incus' native Prometheus metrics surface.
type IncusMetricsScraper func(context.Context) (string, error)

// WebhookDeliveryFunc sends one signed webhook HTTP request.
type WebhookDeliveryFunc func(context.Context, string, string, []byte) (*int, *string, error)

// AuditEmitFunc writes one audit record.
type AuditEmitFunc func(*slog.Logger, *audit.Record)

// ScheduleUnitManager owns the narrow systemd helper handoff for schedules.
type ScheduleUnitManager interface {
	Install(context.Context, systemd.ScheduleSpec) error
	Remove(context.Context, string) error
}

// IncusTrustManager owns per-user Incus certificate trust lifecycle.
type IncusTrustManager interface {
	Provision(context.Context, *store.User) error
	Revoke(context.Context, *store.User) error
}

// Config wires the server's collaborators.
type Config struct {
	Store               *store.Store
	Logger              *slog.Logger
	Version             VersionInfo
	Auth                AuthSettings
	IncusProber         IncusProber
	IncusMetrics        IncusMetricsScraper
	Incus               incus.Client
	IncusProxy          *proxy.IncusProxy
	DelegatedIncusProxy *proxy.DelegatedIncusProxy
	PodmanProxy         *proxy.PodmanProxy
	ScheduleUnits       ScheduleUnitManager
	IncusTrust          IncusTrustManager
	AuditEmit           AuditEmitFunc
	WebhookDelivery     WebhookDeliveryFunc
	WebhookRetryDelays  []time.Duration
	WebhookWorkers      int
	EventRetentionRows  int
	EventRetentionAge   time.Duration
}

// Server is the top-level HTTP server.
type Server struct {
	cfg          Config
	router       chi.Router
	userLimiter  *auth.RateLimiter
	ipLimiter    *auth.RateLimiter
	setup        firstAdminSetupService
	mfaMu        sync.Mutex
	mfaTokens    map[string]mfaChallenge
	events       *eventHub
	metrics      *metricsRegistry
	webhookQueue chan store.OutboxEvent
	webhookWake  chan struct{}
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

	cfgCopy := *cfg
	if cfgCopy.AuditEmit == nil {
		cfgCopy.AuditEmit = defaultAuditEmit
	}
	if cfgCopy.WebhookDelivery == nil {
		cfgCopy.WebhookDelivery = deliverWebhookOnce
	}
	if cfgCopy.WebhookRetryDelays == nil {
		cfgCopy.WebhookRetryDelays = []time.Duration{time.Second, 10 * time.Second, 60 * time.Second}
	}
	if cfgCopy.WebhookWorkers <= 0 {
		cfgCopy.WebhookWorkers = 4
	}
	if cfgCopy.EventRetentionRows <= 0 {
		cfgCopy.EventRetentionRows = 150000
	}
	if cfgCopy.EventRetentionAge <= 0 {
		cfgCopy.EventRetentionAge = 7 * 24 * time.Hour
	}

	s := &Server{
		cfg:          cfgCopy,
		userLimiter:  auth.NewRateLimiter(cfgCopy.Auth.UsernameLimit, cfgCopy.Auth.UsernameWindow),
		ipLimiter:    auth.NewRateLimiter(cfgCopy.Auth.IPLimit, cfgCopy.Auth.IPWindow),
		setup:        newFirstAdminSetupService(cfgCopy.Store, cfgCopy.Auth.Argon2, cfgCopy.Auth.SetupTokenPath),
		mfaTokens:    make(map[string]mfaChallenge),
		events:       newEventHub(),
		metrics:      newMetricsRegistry(),
		webhookQueue: make(chan store.OutboxEvent, 8192),
		webhookWake:  make(chan struct{}, 1),
	}
	s.router = s.routes()
	return s, nil
}

// Handler returns the chi router as an http.Handler suitable for http.Server.
func (s *Server) Handler() http.Handler {
	return s.router
}

// StartBackground starts daemon-scoped workers owned by the caller's context.
func (s *Server) StartBackground(ctx context.Context) {
	for i := 0; i < s.cfg.WebhookWorkers; i++ {
		go s.webhookWorker(ctx)
	}
	go s.outboxDrainer(ctx)
	go s.eventRetentionWorker(ctx)
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()

	r.Use(requestIDMiddleware)
	r.Use(loggerMiddleware(s.cfg.Logger))
	r.Use(middleware.Recoverer)
	r.Use(timeoutExceptSSE(60 * time.Second))
	r.Use(s.metricsMiddleware)

	// Public, unauthenticated.
	r.Get("/healthz", s.handleHealth)
	r.Get("/metrics", s.handleMetrics)
	r.Get("/v1/version", s.handleVersion)

	r.Route("/v1", s.registerV1Routes)
	r.Route("/api/v1", s.registerV1Routes)

	// Proxy pass-through to Incus and Podman (ADR-014/024/036).
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.HandleFunc("/api/incus/*", s.handleIncusProxy)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.adminMiddleware)
		if s.cfg.PodmanProxy != nil {
			r.Handle("/api/podman/*", s.cfg.PodmanProxy)
		}
	})

	r.NotFound(s.handleNotFound)
	r.MethodNotAllowed(s.handleMethodNotAllowed)

	return r
}

func (s *Server) handleIncusProxy(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	mutating := incusProxyMutation(r.Method)
	if mutating {
		if scopes, fromToken := apiTokenScopesFromContext(r.Context()); fromToken && !scopeAllows(scopes, auth.ScopeWrite) {
			s.auditForUserWithPolicyReason(r, &user, "policy.deny", outcomeDenied, "incus", r.URL.RequestURI(), "rbac.insufficient_scope", "Incus proxy mutation denied")
			writeError(w, http.StatusForbidden, "forbidden", "write API token scope required")
			return
		}
	}
	if user.IsAdmin {
		if scopes, fromToken := apiTokenScopesFromContext(r.Context()); fromToken && scopes != auth.ScopeAdmin {
			s.auditForUserWithPolicyReason(r, &user, "policy.deny", outcomeDenied, "incus", r.URL.RequestURI(), "rbac.insufficient_scope", "admin API token scope required for raw Incus proxy")
			writeError(w, http.StatusForbidden, "forbidden", "admin API token scope required")
			return
		}
		if s.cfg.IncusProxy == nil {
			writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "Incus proxy not configured")
			if mutating {
				s.auditIncusProxyMutation(r, &user, http.StatusServiceUnavailable)
			}
			return
		}
		if mutating {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			s.cfg.IncusProxy.ServeHTTP(ww, r)
			s.auditIncusProxyMutation(r, &user, ww.Status())
			return
		}
		s.cfg.IncusProxy.ServeHTTP(w, r)
		return
	}
	if s.cfg.DelegatedIncusProxy == nil {
		writeError(w, http.StatusForbidden, "forbidden", "admin role required")
		if mutating {
			s.auditIncusProxyMutation(r, &user, http.StatusForbidden)
		}
		return
	}
	if mutating {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		s.cfg.DelegatedIncusProxy.ServeHTTPForUser(ww, r, user.ID)
		s.auditIncusProxyMutation(r, &user, ww.Status())
		return
	}
	s.cfg.DelegatedIncusProxy.ServeHTTPForUser(w, r, user.ID)
}

func incusProxyMutation(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func (s *Server) registerV1Routes(r chi.Router) {
	// Public auth endpoints.
	r.Get("/healthz", s.handleHealth)
	r.Get("/version", s.handleVersion)
	r.Post("/auth/login", s.handleLogin)
	r.Get("/auth/setup/status", s.handleSetupStatus)
	r.Post("/auth/setup", s.handleSetup)
	r.Post("/auth/mfa/complete", s.handleMFAComplete)
	r.With(s.scheduleRunnerMiddleware).Post("/internal/schedules/{id}/run", s.handleRunSchedule)

	// Authenticated surface.
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Auth.
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)
		r.With(s.writeScopeMiddleware).Post("/auth/totp/setup", s.handleTOTPSetup)
		r.With(s.writeScopeMiddleware).Post("/auth/totp/verify", s.handleTOTPVerify)
		r.With(s.writeScopeMiddleware).Delete("/auth/totp", s.handleTOTPDelete)
		r.Get("/auth/tokens", s.handleListTokens)
		r.With(s.writeScopeMiddleware).Post("/auth/tokens", s.handleCreateToken)
		r.With(s.writeScopeMiddleware).Delete("/auth/tokens/{id}", s.handleRevokeToken)

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
		r.Patch("/users/{id}", s.handleUpdateUser)
		r.Put("/users/{id}/scope", s.handleUpdateUserScope)
		r.Delete("/users/{id}", s.handleDeleteUser)

		// Privileged system config and upgrade.
		r.Get("/system/config", s.handleSystemConfig)
		r.Put("/system/config", s.handleSystemConfigUpdate)
		r.Post("/system/upgrade", s.handleSystemUpgrade)

		// Deferred privileged surfaces.
		r.Get("/schedules", s.handleListSchedules)
		r.Post("/schedules", s.handleCreateSchedule)
		r.Get("/schedules/{id}", s.handleGetSchedule)
		r.Patch("/schedules/{id}", s.handleUpdateSchedule)
		r.Delete("/schedules/{id}", s.handleDeleteSchedule)
		r.Post("/schedules/{id}/run", s.handleRunSchedule)

		r.Get("/webhooks", s.handleListWebhooks)
		r.Post("/webhooks", s.handleCreateWebhook)
		r.Get("/webhooks/{id}", s.handleGetWebhook)
		r.Patch("/webhooks/{id}", s.handleUpdateWebhook)
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

const (
	ctxKeyRequestID      ctxKey = "request_id"
	ctxKeyRequestStarted ctxKey = "request_started"
	ctxKeyJWTID          ctxKey = "jwt_id"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		ctx = context.WithValue(ctx, ctxKeyRequestStarted, time.Now())
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

func requestStartedFromContext(ctx context.Context) (time.Time, bool) {
	v, ok := ctx.Value(ctxKeyRequestStarted).(time.Time)
	return v, ok
}

func jwtIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyJWTID).(string); ok {
		return v
	}
	return ""
}

func timeoutExceptSSE(timeout time.Duration) func(http.Handler) http.Handler {
	timeoutMiddleware := middleware.Timeout(timeout)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wantsSSE(r) {
				next.ServeHTTP(w, r)
				return
			}
			timeoutMiddleware(next).ServeHTTP(w, r)
		})
	}
}
