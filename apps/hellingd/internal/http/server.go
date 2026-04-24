// Package httpserver wires net/http ServeMux with Huma operations.
package httpserver

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	hellingapi "github.com/Bizarre-Industries/Helling/apps/hellingd/api"
)

// NewAPI mounts Helling-owned API operations on top of the provided ServeMux
// with stubbed handlers. Kept for the Huma-spike test coverage.
func NewAPI(mux *http.ServeMux) huma.API {
	return NewAPIWith(mux, hellingapi.Deps{})
}

// NewAPIWith mounts Helling-owned API operations using the supplied Deps.
// Passing a zero Deps is equivalent to NewAPI.
func NewAPIWith(mux *http.ServeMux, deps hellingapi.Deps) huma.API {
	config := hellingapi.NewConfig()
	api := humago.New(mux, config)
	hellingapi.RegisterOperationsWith(api, deps)
	hellingapi.EnrichOpenAPI(api.OpenAPI())
	return api
}

// NewMux builds the daemon's top-level net/http router with stub handlers.
func NewMux() *http.ServeMux {
	return NewMuxWith(hellingapi.Deps{})
}

// NewMuxWith builds the daemon's top-level router using the supplied Deps.
// When Deps.IncusProxy or Deps.PodmanProxy are non-nil the proxy routes are
// mounted alongside the Huma API routes (ADR-014).
func NewMuxWith(deps hellingapi.Deps) *http.ServeMux {
	mux := http.NewServeMux()
	_ = NewAPIWith(mux, deps)
	if deps.HasIncusProxy() {
		mux.Handle("/api/incus/", deps.IncusProxy)
	}
	if deps.HasPodmanProxy() {
		mux.Handle("/api/podman/", deps.PodmanProxy)
	}
	return mux
}
