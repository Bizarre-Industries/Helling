package proxy

import (
	"context"
	"net/http"
	"testing"
)

func TestProxyDirectorStripsHellingCredentials(t *testing.T) {
	t.Parallel()
	proxy := newUnixReverseProxy("/tmp/missing.sock", "/api/incus", "incus", "incus_error", nil)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://helling.local/api/incus/1.0", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "helling_session=secret")
	req.Header.Set("X-Request-ID", "client-controlled")
	req.Header.Set("X-Helling-User", "alice")
	req.Header.Set("X-Helling-Project", "alice")
	proxy.Director(req)
	for _, name := range []string{"Authorization", "Cookie", "X-Request-ID", "X-Helling-User", "X-Helling-Project"} {
		if got := req.Header.Get(name); got != "" {
			t.Fatalf("%s header survived proxy director: %q", name, got)
		}
	}
}

func TestPodmanRequestAllowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		path    string
		allowed bool
	}{
		{name: "container list", method: http.MethodGet, path: "/api/podman/libpod/containers/json", allowed: true},
		{name: "container inspect", method: http.MethodHead, path: "/api/podman/libpod/containers/abc/json", allowed: true},
		{name: "ping", method: http.MethodGet, path: "/api/podman/_ping", allowed: true},
		{name: "delete blocked", method: http.MethodDelete, path: "/api/podman/libpod/containers/abc", allowed: false},
		{name: "exec blocked", method: http.MethodPost, path: "/api/podman/libpod/containers/abc/exec", allowed: false},
		{name: "unknown get blocked", method: http.MethodGet, path: "/api/podman/libpod/secrets/json", allowed: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequestWithContext(context.Background(), tt.method, "http://helling.local"+tt.path, http.NoBody)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if got := podmanRequestAllowed(req); got != tt.allowed {
				t.Fatalf("podmanRequestAllowed() = %v, want %v", got, tt.allowed)
			}
		})
	}
}
