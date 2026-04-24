package api

import (
	"net/http"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

// Deps carries optional runtime dependencies for Helling API handlers.
// A zero Deps keeps the stubbed handlers, which is what the unit tests for
// the Huma spike rely on. Production wiring passes the real services.
type Deps struct {
	// Auth is the auth service powering setup/login/logout/refresh.
	// When nil, stub handlers are registered instead.
	Auth *auth.Service
	// IncusProxy handles /api/incus/* when non-nil (ADR-014).
	IncusProxy http.Handler
	// PodmanProxy handles /api/podman/* when non-nil.
	PodmanProxy http.Handler
}

// HasAuth reports whether a real auth service is wired in.
func (d Deps) HasAuth() bool { return d.Auth != nil }

// HasIncusProxy reports whether the Incus reverse-proxy is wired.
func (d Deps) HasIncusProxy() bool { return d.IncusProxy != nil }

// HasPodmanProxy reports whether the Podman reverse-proxy is wired.
func (d Deps) HasPodmanProxy() bool { return d.PodmanProxy != nil }
