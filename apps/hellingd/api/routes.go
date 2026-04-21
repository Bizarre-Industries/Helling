// Package api defines Huma operations for Helling-owned routes.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HealthData is the minimal payload used to prove the Huma pipeline wiring.
type HealthData struct {
	Status string `json:"status" doc:"Service health state." enum:"ok"`
}

// HealthMeta keeps the envelope shape aligned with docs/spec/api.md.
type HealthMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// HealthEnvelope follows the Helling success envelope contract.
type HealthEnvelope struct {
	Data HealthData `json:"data"`
	Meta HealthMeta `json:"meta"`
}

// HealthOutput is the response shape for GET /api/v1/health.
type HealthOutput struct {
	Body HealthEnvelope
}

// RegisterOperations wires the current Huma spike operations.
func RegisterOperations(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Health check",
		Description: "Returns service health for unauthenticated readiness checks.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		_ = ctx
		_ = input

		return &HealthOutput{
			Body: HealthEnvelope{
				Data: HealthData{Status: "ok"},
				Meta: HealthMeta{RequestID: "req_huma_spike"},
			},
		}, nil
	})
}
