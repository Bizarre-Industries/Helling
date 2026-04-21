package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestNewConfigProvidesRequiredMetadata(t *testing.T) {
	config := NewConfig()
	if config.Info == nil || config.Info.Description == "" {
		t.Fatal("expected info description in config")
	}
	if len(config.Servers) == 0 {
		t.Fatal("expected at least one server in config")
	}
	if len(config.Tags) == 0 || config.Tags[0].Name != "System" {
		t.Fatal("expected System global tag in config")
	}
}

func TestRegisterOperationsHealthRoute(t *testing.T) {
	mux := http.NewServeMux()
	api := humago.New(mux, NewConfig())
	RegisterOperations(api)
	EnrichOpenAPI(api.OpenAPI())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("expected status payload in response body, got: %s", body)
	}

	if !strings.Contains(body, `"request_id":"req_huma_spike"`) {
		t.Fatalf("expected request_id in response body, got: %s", body)
	}
}

func TestEnrichOpenAPIPatchesSchemaMetadata(t *testing.T) {
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig(apiTitle, apiVersion))
	RegisterOperations(api)
	EnrichOpenAPI(api.OpenAPI())

	doc := api.OpenAPI()
	if doc.Info == nil || doc.Info.Description == "" {
		t.Fatal("expected info description after enrichment")
	}
	if len(doc.Servers) == 0 {
		t.Fatal("expected servers after enrichment")
	}

	schemas := doc.Components.Schemas.Map()
	required := []string{"ErrorDetail", "ErrorModel", "HealthData", "HealthEnvelope", "HealthMeta"}
	for _, name := range required {
		s := schemas[name]
		if s == nil || s.Description == "" {
			t.Fatalf("expected description for schema %s", name)
		}
	}

	errorDetail := schemas["ErrorDetail"]
	if errorDetail.Properties["value"].Type == "" {
		t.Fatal("expected type for ErrorDetail.value")
	}

	healthPath := doc.Paths["/api/v1/health"]
	if healthPath == nil || healthPath.Get == nil {
		t.Fatal("expected health path operation")
	}
	if healthPath.Get.Responses["200"].Content["application/json"].Example == nil {
		t.Fatal("expected success example on health response")
	}
	if healthPath.Get.Responses["default"].Content["application/problem+json"].Example == nil {
		t.Fatal("expected default problem response example")
	}
}
