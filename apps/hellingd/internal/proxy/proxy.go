// Package proxy implements the authenticated reverse-proxy middleware that
// forwards /api/incus/* and /api/podman/* requests to the upstream runtimes
// per docs/spec/proxies.md and ADR-014.
//
// v0.1 scope notes:
//   - Per-user Incus mTLS certificates are not yet provisioned (requires the
//     internal CA from docs/spec/internal-ca.md). Today the proxy shares a
//     single client certificate loaded from HELLING_INCUS_CLIENT_CERT/KEY.
//     Per-user certs are tracked as a v0.1-beta gate.
//   - WebSocket upgrade handling is deferred to v0.1-beta; upgrade requests
//     return 501 with PROXY_WEBSOCKET_UNSUPPORTED_V01_ALPHA.
package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

// Config carries the upstream endpoints for Incus and Podman.
type Config struct {
	// IncusURL is the upstream Incus HTTPS endpoint, e.g. https://127.0.0.1:8443.
	// Empty disables the Incus route (returns 503).
	IncusURL string
	// IncusClientCertPath is a PEM file for mTLS. Empty disables mTLS (lab only).
	IncusClientCertPath string
	// IncusClientKeyPath is the matching key file.
	IncusClientKeyPath string
	// IncusCAPath is the Incus CA bundle (optional).
	IncusCAPath string
	// PodmanSocket is the Podman Unix socket path (e.g. /run/podman/podman.sock).
	// Empty disables the Podman route (returns 503).
	PodmanSocket string
	// InsecureSkipVerify allows self-signed upstream certs in dev only.
	InsecureSkipVerify bool
}

// ConfigFromEnv reads proxy configuration from the standard environment
// variables. Missing values leave the matching route disabled.
func ConfigFromEnv() Config {
	return Config{
		IncusURL:            os.Getenv("HELLING_INCUS_URL"),
		IncusClientCertPath: os.Getenv("HELLING_INCUS_CLIENT_CERT"),
		IncusClientKeyPath:  os.Getenv("HELLING_INCUS_CLIENT_KEY"),
		IncusCAPath:         os.Getenv("HELLING_INCUS_CA"),
		PodmanSocket:        os.Getenv("HELLING_PODMAN_SOCKET"),
		InsecureSkipVerify:  os.Getenv("HELLING_PROXY_INSECURE_SKIP_VERIFY") == "1",
	}
}

// Proxy wires auth, logging, and upstream dispatch.
type Proxy struct {
	cfg    Config
	svc    *auth.Service
	logger *slog.Logger
	incus  http.Handler // nil when disabled
	podman http.Handler // nil when disabled
}

// New constructs a Proxy. Missing envs yield functional 503 routes so that
// the Helling-owned endpoints continue to work in dev where upstreams are not
// running yet. cfg is passed by pointer to avoid the large-value copy.
func New(cfg *Config, svc *auth.Service, logger *slog.Logger) (*Proxy, error) {
	if logger == nil {
		logger = slog.Default()
	}
	p := &Proxy{cfg: *cfg, svc: svc, logger: logger}

	if cfg.IncusURL != "" {
		h, err := buildIncusHandler(cfg)
		if err != nil {
			return nil, err
		}
		p.incus = h
	}
	if cfg.PodmanSocket != "" {
		h, err := buildPodmanHandler(cfg.PodmanSocket)
		if err != nil {
			return nil, err
		}
		p.podman = h
	}
	return p, nil
}

// IncusHandler returns the bearer-auth wrapped handler for /api/incus/.
func (p *Proxy) IncusHandler() http.Handler {
	return p.bearerAuth("incus", func(w http.ResponseWriter, r *http.Request) {
		if p.incus == nil {
			writeErr(w, "incus", http.StatusServiceUnavailable, "INCUS_UPSTREAM_NOT_CONFIGURED")
			return
		}
		if isWebSocketUpgrade(r) {
			writeErr(w, "incus", http.StatusNotImplemented, "PROXY_WEBSOCKET_UNSUPPORTED_V01_ALPHA")
			return
		}
		r.URL.Path = stripPrefix(r.URL.Path, "/api/incus")
		p.incus.ServeHTTP(w, r)
	})
}

// PodmanHandler returns the bearer-auth wrapped handler for /api/podman/.
func (p *Proxy) PodmanHandler() http.Handler {
	return p.bearerAuth("podman", func(w http.ResponseWriter, r *http.Request) {
		if p.podman == nil {
			writeErr(w, "podman", http.StatusServiceUnavailable, "PODMAN_UPSTREAM_NOT_CONFIGURED")
			return
		}
		if isWebSocketUpgrade(r) {
			writeErr(w, "podman", http.StatusNotImplemented, "PROXY_WEBSOCKET_UNSUPPORTED_V01_ALPHA")
			return
		}
		r.URL.Path = stripPrefix(r.URL.Path, "/api/podman")
		p.podman.ServeHTTP(w, r)
	})
}

// bearerAuth verifies the Authorization Bearer token (JWT or API token),
// strips Helling-internal headers, and passes through. Failures return 401.
func (p *Proxy) bearerAuth(source string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.resolveBearer(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			writeErr(w, source, http.StatusUnauthorized, "PROXY_UNAUTHENTICATED")
			return
		}

		for h := range r.Header {
			if strings.HasPrefix(strings.ToLower(h), "x-helling-") {
				r.Header.Del(h)
			}
		}
		r.Header.Del("Authorization")
		r.Header.Set("X-Forwarded-User", userID)

		start := time.Now()
		sw := &statusRecorder{ResponseWriter: w, status: 200}
		next(sw, r)

		// Async audit (non-blocking). Detach from request cancellation but
		// keep trace values via context.WithoutCancel.
		if p.svc != nil {
			go p.audit(context.WithoutCancel(r.Context()), userID, r.Method, r.URL.Path, source, sw.status, time.Since(start))
		}
	})
}

func (p *Proxy) audit(base context.Context, userID, method, path, source string, status int, dur time.Duration) {
	ctx, cancel := context.WithTimeout(base, 2*time.Second)
	defer cancel()
	meta, _ := json.Marshal(map[string]any{
		"method":      method,
		"path":        path,
		"status":      status,
		"duration_ms": dur.Milliseconds(),
		"source":      source,
	})
	_ = p.svc.Repo().RecordEvent(ctx, userID, "proxy.forward", "", "", string(meta))
}

// resolveBearer returns the owning user id for a JWT access or API token.
func (p *Proxy) resolveBearer(ctx context.Context, authz string) (string, error) {
	if p.svc == nil {
		return "", auth.ErrInvalidToken
	}
	tok := strings.TrimPrefix(authz, "Bearer ")
	tok = strings.TrimSpace(tok)
	if tok == "" || authz == tok {
		return "", auth.ErrInvalidToken
	}
	if strings.HasPrefix(tok, auth.APITokenPrefix) {
		u, _, err := p.svc.VerifyAPIToken(ctx, tok)
		if err != nil {
			return "", auth.ErrInvalidToken
		}
		return u.ID, nil
	}
	claims, err := p.svc.Signer().Verify(tok)
	if err != nil {
		return "", err
	}
	return claims.Subject, nil
}

// buildIncusHandler wires a ReverseProxy + optional mTLS for the Incus loopback.
func buildIncusHandler(cfg *Config) (http.Handler, error) {
	target, err := url.Parse(cfg.IncusURL)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // dev-only switch
	}
	if cfg.IncusClientCertPath != "" && cfg.IncusClientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.IncusClientCertPath, cfg.IncusClientKeyPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	if cfg.IncusCAPath != "" {
		caPEM, err := os.ReadFile(cfg.IncusCAPath)
		if err != nil {
			return nil, err
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		pool.AppendCertsFromPEM(caPEM)
		tlsCfg.RootCAs = pool
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsCfg,
		MaxIdleConns:        16,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     5 * time.Minute,
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = transport
	rp.ErrorHandler = incusErrorHandler
	rp.ModifyResponse = tagIncusResponse
	return rp, nil
}

// buildPodmanHandler wires a ReverseProxy over a Unix-socket dialer.
func buildPodmanHandler(socketPath string) (http.Handler, error) {
	if _, err := os.Stat(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	target, _ := url.Parse("http://podman.local")
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
		IdleConnTimeout: 5 * time.Minute,
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = transport
	rp.ErrorHandler = podmanErrorHandler
	rp.ModifyResponse = tagPodmanResponse
	return rp, nil
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func stripPrefix(path, prefix string) string {
	out := strings.TrimPrefix(path, prefix)
	if !strings.HasPrefix(out, "/") {
		return "/" + out
	}
	return out
}

// statusRecorder captures the status code for audit logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func writeErr(w http.ResponseWriter, source string, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":   nil,
		"error":  detail,
		"code":   status,
		"source": source,
		"meta":   map[string]string{},
	})
}

func incusErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	writeErr(w, "incus", http.StatusBadGateway, "INCUS_UPSTREAM_ERROR: "+err.Error())
}

func podmanErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	writeErr(w, "podman", http.StatusBadGateway, "PODMAN_UPSTREAM_ERROR: "+err.Error())
}

// tagIncusResponse / tagPodmanResponse are ModifyResponse hooks. They tag the
// upstream source and strip any auth cookies. Per docs/spec/proxies.md
// §Error Normalization the alpha keeps upstream bodies transparent;
// body-level normalization lands in a follow-up PR. httputil.ReverseProxy
// owns the response body lifetime; these hooks only adjust headers.
func tagIncusResponse(resp *http.Response) error {
	resp.Header.Set("X-Proxy-Source", "incus")
	resp.Header.Del("Set-Cookie")
	return nil
}

func tagPodmanResponse(resp *http.Response) error {
	resp.Header.Set("X-Proxy-Source", "podman")
	resp.Header.Del("Set-Cookie")
	return nil
}
