package api

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const (
	apiTitle         = "Helling API"
	apiVersion       = "0.1.0"
	schemaTypeString = "string"
)

var defaultTags = []huma.Tag{
	{Name: "Auth", Description: "Authentication and token lifecycle endpoints."},
	{Name: "Users", Description: "User account and access management endpoints."},
	{Name: "System", Description: "System and service health endpoints."},
}

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
	config.Tags = make([]*huma.Tag, 0, len(defaultTags))
	for _, tag := range defaultTags {
		config.Tags = append(config.Tags, &huma.Tag{Name: tag.Name, Description: tag.Description})
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

	ensureDefaultTags(doc)

	schemas := doc.Components.Schemas.Map()
	setSchemaMetadata(schemas)
	setResponseExamples(doc)
	ensureSchemaFallbackMetadata(schemas)
	ensureMediaExamples(doc, schemas)
}

func ensureDefaultTags(doc *huma.OpenAPI) {
	if len(doc.Tags) > 0 {
		return
	}

	doc.Tags = make([]*huma.Tag, 0, len(defaultTags))
	for _, tag := range defaultTags {
		doc.Tags = append(doc.Tags, &huma.Tag{Name: tag.Name, Description: tag.Description})
	}
}

func setSchemaMetadata(schemas map[string]*huma.Schema) {
	enrichErrorDetailSchema(schemas["ErrorDetail"])
	enrichErrorModelSchema(schemas["ErrorModel"])
	enrichHealthDataSchema(schemas["HealthData"])
	enrichHealthEnvelopeSchema(schemas["HealthEnvelope"])
	enrichHealthMetaSchema(schemas["HealthMeta"])
	enrichAuthLogoutEnvelopeSchema(schemas["AuthLogoutEnvelope"])
	enrichAuthRefreshEnvelopeSchema(schemas["AuthRefreshEnvelope"])
}

func enrichAuthLogoutEnvelopeSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Success envelope for auth logout responses."
	schema.Examples = []any{map[string]any{
		"data": map[string]any{},
		"meta": map[string]any{"request_id": "req_auth_logout"},
	}}
}

func enrichAuthRefreshEnvelopeSchema(schema *huma.Schema) {
	if schema == nil {
		return
	}

	schema.Description = "Success envelope for auth refresh responses."
	schema.Examples = []any{map[string]any{
		"data": map[string]any{
			"access_token": "eyJhbGciOiJFZERTQSJ9.refresh.stub",
			"token_type":   "Bearer",
			"expires_in":   900,
		},
		"meta": map[string]any{"request_id": "req_auth_refresh"},
	}}
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
			value.Type = schemaTypeString
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

func ensureSchemaFallbackMetadata(schemas map[string]*huma.Schema) {
	for name, schema := range schemas {
		ensureSchemaDescription(name, schema)
		ensureSchemaExample(name, schema, schemas)
		ensurePropertyMetadata(schema, schemas)
	}
}

func ensureSchemaDescription(name string, schema *huma.Schema) {
	if schema == nil || schema.Description != "" {
		return
	}

	schema.Description = "Schema for " + strings.ToLower(name) + "."
}

func ensureSchemaExample(name string, schema *huma.Schema, schemas map[string]*huma.Schema) {
	if schema == nil || len(schema.Examples) > 0 {
		return
	}

	schema.Examples = []any{exampleForSchema(name, schema, schemas)}
}

func ensurePropertyMetadata(schema *huma.Schema, schemas map[string]*huma.Schema) {
	if schema == nil || len(schema.Properties) == 0 {
		return
	}

	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		if propSchema.Description == "" {
			propSchema.Description = "Property " + propName + "."
		}
		if len(propSchema.Examples) == 0 {
			propSchema.Examples = []any{exampleForSchema(propName, propSchema, schemas)}
		}
		if propSchema.Type == "" && propSchema.Ref == "" {
			propSchema.Type = schemaTypeString
		}
	}
}

func ensureMediaExamples(doc *huma.OpenAPI, schemas map[string]*huma.Schema) {
	for _, pathItem := range doc.Paths {
		for _, operation := range operationsFromPath(pathItem) {
			if operation == nil {
				continue
			}
			if operation.RequestBody != nil {
				ensureContentExamples(operation.RequestBody.Content, schemas)
			}
			for _, response := range operation.Responses {
				if response == nil {
					continue
				}
				ensureContentExamples(response.Content, schemas)
			}
		}
	}
}

func operationsFromPath(pathItem *huma.PathItem) []*huma.Operation {
	if pathItem == nil {
		return nil
	}

	return []*huma.Operation{
		pathItem.Get,
		pathItem.Post,
		pathItem.Put,
		pathItem.Patch,
		pathItem.Delete,
		pathItem.Head,
		pathItem.Options,
		pathItem.Trace,
	}
}

func ensureContentExamples(content map[string]*huma.MediaType, schemas map[string]*huma.Schema) {
	for _, media := range content {
		if media == nil {
			continue
		}
		if media.Example != nil || len(media.Examples) > 0 {
			continue
		}
		media.Example = exampleForSchema("response", media.Schema, schemas)
	}
}

func exampleForSchema(name string, schema *huma.Schema, schemas map[string]*huma.Schema) any {
	if schema == nil {
		return map[string]any{}
	}

	if len(schema.Examples) > 0 {
		return schema.Examples[0]
	}

	if schema.Ref != "" {
		resolved := schemas[schemaNameFromRef(schema.Ref)]
		if resolved != nil && len(resolved.Examples) > 0 {
			return resolved.Examples[0]
		}
		return map[string]any{}
	}

	return scalarOrStructuredExample(name, schema)
}

func scalarOrStructuredExample(name string, schema *huma.Schema) any {
	switch schema.Type {
	case "boolean":
		return true
	case "integer", "number":
		return 1
	case "array":
		return []any{scalarOrStructuredExample(name, schema.Items)}
	case schemaTypeString:
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		if strings.Contains(strings.ToLower(name), "id") {
			return "id_example"
		}
		return "example"
	case "object":
		obj := map[string]any{}
		for propName, propSchema := range schema.Properties {
			if propSchema == nil {
				continue
			}
			if len(propSchema.Examples) > 0 {
				obj[propName] = propSchema.Examples[0]
				continue
			}
			obj[propName] = scalarOrStructuredExample(propName, propSchema)
		}
		return obj
	default:
		return map[string]any{}
	}
}

func schemaNameFromRef(ref string) string {
	return strings.TrimPrefix(ref, "#/components/schemas/")
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
