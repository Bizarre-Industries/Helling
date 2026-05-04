// Package proxy implements httputil.ReverseProxy handlers that forward
// authenticated requests to Incus and Podman Unix sockets per ADR-014.
//
// The proxy validates the JWT/session, maps the Helling user to an Incus
// project, and forwards the request with the native upstream response
// format (no re-enveloping).
package proxy

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
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
			req.Header.Del("X-Helling-User")
			req.Header.Del("X-Helling-Project")
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
			http.Error(w, `{"code":"`+errorCode+`","message":"upstream unreachable"}`, http.StatusBadGateway)
		},
	}
}

// ServeHTTP implements http.Handler.
func (p *PodmanProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
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
