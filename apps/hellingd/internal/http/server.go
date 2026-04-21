// Package httpserver wires net/http ServeMux with Huma operations.
package httpserver

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	hellingapi "github.com/Bizarre-Industries/Helling/apps/hellingd/api"
)

// NewAPI mounts Helling-owned API operations on top of the provided ServeMux.
func NewAPI(mux *http.ServeMux) huma.API {
	config := hellingapi.NewConfig()
	api := humago.New(mux, config)
	hellingapi.RegisterOperations(api)
	hellingapi.EnrichOpenAPI(api.OpenAPI())
	return api
}

// NewMux builds the daemon's top-level net/http router.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	_ = NewAPI(mux)
	return mux
}
