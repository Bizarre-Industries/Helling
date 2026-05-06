// Package proxy implements httputil.ReverseProxy handlers that forward
// authenticated requests to Incus and Podman Unix sockets per ADR-014.
//
// Auth and role checks are enforced by the server route that mounts these
// handlers. The proxy strips Helling credentials and internal headers, then
// forwards native upstream responses without re-enveloping.
package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// IncusProxy forwards /api/incus/* to the Incus Unix socket.
type IncusProxy struct {
	proxy  *httputil.ReverseProxy
	logger *slog.Logger
}

// NewIncusProxy creates a reverse proxy to the Incus Unix socket at socketPath.
func NewIncusProxy(socketPath string, logger *slog.Logger) *IncusProxy {
	p := &IncusProxy{logger: logger}
	p.proxy = newUnixReverseProxy(socketPath, "/api/incus", "incus", "incus_error", logger)
	return p
}

// ServeHTTP implements http.Handler.
func (p *IncusProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}

// UserCertificateProvider returns the client TLS identity for a Helling user.
type UserCertificateProvider func(context.Context, int64) (tls.Certificate, error)

// DelegatedIncusProxy forwards non-admin /api/incus/* requests over loopback
// HTTPS with the caller's per-user Incus client certificate.
type DelegatedIncusProxy struct {
	endpoint     *url.URL
	rootCAs      *x509.CertPool
	serverName   string
	certProvider UserCertificateProvider
	logger       *slog.Logger
}

// DelegatedIncusProxyConfig configures the per-user Incus proxy path.
type DelegatedIncusProxyConfig struct {
	Endpoint     string
	CACertPath   string
	RootCAs      *x509.CertPool
	ServerName   string
	CertProvider UserCertificateProvider
	Logger       *slog.Logger
}

// NewDelegatedIncusProxy creates a delegated-user Incus HTTPS proxy.
func NewDelegatedIncusProxy(cfg DelegatedIncusProxyConfig) (*DelegatedIncusProxy, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://127.0.0.1:8443"
	}
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing delegated Incus endpoint: %w", err)
	}
	if endpoint.Scheme != "https" || endpoint.Host == "" {
		return nil, errors.New("delegated Incus endpoint must be an https URL")
	}
	if cfg.CertProvider == nil {
		return nil, errors.New("delegated Incus certificate provider is required")
	}
	rootCAs := cfg.RootCAs
	if rootCAs == nil && cfg.CACertPath != "" {
		rootCAs, err = loadCertPool(cfg.CACertPath)
		if err != nil {
			return nil, err
		}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &DelegatedIncusProxy{
		endpoint:     endpoint,
		rootCAs:      rootCAs,
		serverName:   cfg.ServerName,
		certProvider: cfg.CertProvider,
		logger:       cfg.Logger,
	}, nil
}

// StoreUserCertificateProvider decrypts per-user Incus certificates from the store.
func StoreUserCertificateProvider(st *store.Store) UserCertificateProvider {
	return func(ctx context.Context, userID int64) (tls.Certificate, error) {
		row, err := st.GetIncusUserCert(ctx, userID)
		if err != nil {
			return tls.Certificate{}, err
		}
		if row.RevokedAt != nil {
			return tls.Certificate{}, errors.New("Incus user certificate revoked")
		}
		if row.ExpiresAt != nil && !row.ExpiresAt.After(time.Now().UTC()) {
			return tls.Certificate{}, errors.New("Incus user certificate expired")
		}
		keyPEM, err := st.DecryptSecret(row.EncryptedKeyPEM)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("decrypting Incus user certificate key: %w", err)
		}
		cert, err := tls.X509KeyPair([]byte(row.CertPEM), []byte(keyPEM))
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("parsing Incus user certificate: %w", err)
		}
		return cert, nil
	}
}

// ServeHTTPForUser implements delegated proxying for a specific Helling user.
func (p *DelegatedIncusProxy) ServeHTTPForUser(w http.ResponseWriter, r *http.Request, userID int64) {
	cert, err := p.certProvider(r.Context(), userID)
	if err != nil {
		p.logger.Warn("delegated incus proxy certificate unavailable", slog.Int64("user_id", userID), slog.Any("err", err))
		writeProxyError(w, http.StatusForbidden, "incus_proxy_forbidden", "Incus proxy certificate is unavailable")
		return
	}
	p.newReverseProxy(&cert).ServeHTTP(w, r)
}

func (p *DelegatedIncusProxy) newReverseProxy(cert *tls.Certificate) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = p.endpoint.Scheme
			req.URL.Host = p.endpoint.Host
			req.URL.Path = singleJoiningSlash(p.endpoint.Path, strings.TrimPrefix(req.URL.Path, "/api/incus"))
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.Host = p.endpoint.Host
			stripHellingProxyHeaders(req.Header)
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{*cert},
				MinVersion:   tls.VersionTLS12,
				RootCAs:      p.rootCAs,
				ServerName:   p.serverName,
			},
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.logger.Error("delegated incus proxy error", slog.Any("err", err))
			writeProxyError(w, http.StatusBadGateway, "incus_error", "upstream unreachable")
		},
	}
}

// PodmanProxy forwards /api/podman/* to the Podman Unix socket.
type PodmanProxy struct {
	proxy  *httputil.ReverseProxy
	logger *slog.Logger
}

// NewPodmanProxy creates a reverse proxy to the Podman Unix socket at socketPath.
func NewPodmanProxy(socketPath string, logger *slog.Logger) *PodmanProxy {
	p := &PodmanProxy{logger: logger}
	p.proxy = newUnixReverseProxy(socketPath, "/api/podman", "podman", "podman_error", logger)
	return p
}

func newUnixReverseProxy(socketPath, prefix, upstreamHost, errorCode string, logger *slog.Logger) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = upstreamHost
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			stripHellingProxyHeaders(req.Header)
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 5*time.Second)
			},
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error(upstreamHost+" proxy error", slog.Any("err", err))
			writeProxyError(w, http.StatusBadGateway, errorCode, "upstream unreachable")
		},
	}
}

func stripHellingProxyHeaders(header http.Header) {
	header.Del("Authorization")
	header.Del("Cookie")
	header.Del("X-Request-ID")
	for name := range header {
		if strings.HasPrefix(http.CanonicalHeaderKey(name), "X-Helling-") {
			header.Del(name)
		}
	}
}

// ServeHTTP implements http.Handler.
func (p *PodmanProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !podmanRequestAllowed(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "podman_proxy_forbidden",
			"message": "podman proxy path or method is not allowed",
		})
		return
	}
	p.proxy.ServeHTTP(w, r)
}

func podmanRequestAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
	default:
		return false
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/podman")
	if path == "" {
		path = "/"
	}
	if path == "/_ping" || path == "/version" || path == "/libpod/info" || path == "/libpod/containers/json" {
		return true
	}
	return strings.HasPrefix(path, "/libpod/containers/") && strings.HasSuffix(path, "/json")
}

// UnixTransport returns an http.RoundTripper that dials a Unix socket.
// Useful for testing or when you need a standalone transport.
func UnixTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, 5*time.Second)
		},
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

// MustParseURL parses a URL or panics. For use in tests and init.
func MustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func loadCertPool(path string) (*x509.CertPool, error) {
	// #nosec G304 -- path is operator-controlled helling.yaml config, not request input.
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading delegated Incus CA certificate: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("delegated Incus CA certificate %s has no PEM certificates", path)
	}
	return pool, nil
}

func singleJoiningSlash(a, b string) string {
	switch {
	case a == "":
		if b == "" {
			return "/"
		}
		return b
	case b == "":
		return a
	case strings.HasSuffix(a, "/") && strings.HasPrefix(b, "/"):
		return a + b[1:]
	case !strings.HasSuffix(a, "/") && !strings.HasPrefix(b, "/"):
		return a + "/" + b
	default:
		return a + b
	}
}

func writeProxyError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": message,
	})
}
