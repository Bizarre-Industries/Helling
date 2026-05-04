package proxy

import (
	"context"
	"net/http"
	"testing"
)

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
