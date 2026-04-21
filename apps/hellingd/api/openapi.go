package api

import "github.com/danielgtaylor/huma/v2"

const (
	apiTitle   = "Helling API"
	apiVersion = "0.1.0"
)

// NewConfig returns the default Huma configuration for Helling-owned APIs.
func NewConfig() huma.Config {
	config := huma.DefaultConfig(apiTitle, apiVersion)
	config.Info.Description = "Generated contract for Helling-owned endpoints under /api/v1."
	config.Servers = []*huma.Server{
		{
			URL:         "https://localhost:8006",
			Description: "Local Helling API endpoint.",
		},
	}
	config.Tags = []*huma.Tag{
		{
			Name:        "System",
			Description: "System and service health endpoints.",
		},
	}

	return config
}

// EnrichOpenAPI patches generated fields to satisfy repository OpenAPI rules.
func EnrichOpenAPI(doc *huma.OpenAPI) {
	if doc == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return
	}

	if doc.Info != nil && doc.Info.Description == "" {
		doc.Info.Description = "Generated contract for Helling-owned endpoints under /api/v1."
	}

	if len(doc.Servers) == 0 {
		doc.Servers = []*huma.Server{{URL: "https://localhost:8006", Description: "Local Helling API endpoint."}}
	}

	if len(doc.Tags) == 0 {
		doc.Tags = []*huma.Tag{{Name: "System", Description: "System and service health endpoints."}}
	}

	schemas := doc.Components.Schemas.Map()
	setSchemaMetadata(schemas)
	setResponseExamples(doc)
}

func setSchemaMetadata(schemas map[string]*huma.Schema) {
	enrichErrorDetailSchema(schemas["ErrorDetail"])
	enrichErrorModelSchema(schemas["ErrorModel"])
	enrichHealthDataSchema(schemas["HealthData"])
	enrichHealthEnvelopeSchema(schemas["HealthEnvelope"])
	enrichHealthMetaSchema(schemas["HealthMeta"])
}

func enrichErrorDetailSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Detailed validation issue information."
	schema.Examples = []any{map[string]any{
		"location": "body.username",
		"message":  "must not be empty",
		"value":    "",
	}}

	if location := schema.Properties["location"]; location != nil {
		location.Examples = []any{"body.username"}
	}
	if message := schema.Properties["message"]; message != nil {
		message.Examples = []any{"must not be empty"}
	}
	if value := schema.Properties["value"]; value != nil {
		if value.Type == "" {
			value.Type = "string"
		}
		if len(value.Examples) == 0 {
			value.Examples = []any{""}
		}
	}
}

func enrichErrorModelSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Problem-details error response model."
	schema.Examples = []any{map[string]any{
		"title":  "Bad Request",
		"status": 400,
		"detail": "validation failed",
		"errors": []map[string]any{{"location": "body.username", "message": "must not be empty", "value": ""}},
	}}
	if errorsProp := schema.Properties["errors"]; errorsProp != nil && len(errorsProp.Examples) == 0 {
		errorsProp.Examples = []any{[]map[string]any{{"location": "body.username", "message": "must not be empty", "value": ""}}}
	}
}

func enrichHealthDataSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Health endpoint payload."
	schema.Examples = []any{map[string]any{"status": "ok"}}
}

func enrichHealthEnvelopeSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Success envelope for health responses."
	schema.Examples = []any{map[string]any{
		"data": map[string]any{"status": "ok"},
		"meta": map[string]any{"request_id": "req_huma_spike"},
	}}
	if data := schema.Properties["data"]; data != nil {
		if data.Description == "" {
			data.Description = "Health payload."
		}
		if len(data.Examples) == 0 {
			data.Examples = []any{map[string]any{"status": "ok"}}
		}
	}
	if meta := schema.Properties["meta"]; meta != nil && meta.Description == "" {
		meta.Description = "Request metadata envelope."
	}
}

func enrichHealthMetaSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Metadata included in health responses."
	schema.Examples = []any{map[string]any{"request_id": "req_huma_spike"}}
}

func setResponseExamples(doc *huma.OpenAPI) {
	path := doc.Paths["/api/v1/health"]
	if path == nil || path.Get == nil {
		return
	}

	if okResp := path.Get.Responses["200"]; okResp != nil {
		if media := okResp.Content["application/json"]; media != nil && media.Example == nil {
			media.Example = map[string]any{
				"data": map[string]any{"status": "ok"},
				"meta": map[string]any{"request_id": "req_huma_spike"},
			}
		}
	}

	if defaultResp := path.Get.Responses["default"]; defaultResp != nil {
		if media := defaultResp.Content["application/problem+json"]; media != nil && media.Example == nil {
			media.Example = map[string]any{
				"title":  "Bad Request",
				"status": 400,
				"detail": "validation failed",
				"errors": []map[string]any{{"location": "body.username", "message": "must not be empty", "value": ""}},
			}
		}
	}
}
