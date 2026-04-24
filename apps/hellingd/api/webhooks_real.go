package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// Webhook real handlers back the /api/v1/webhooks* stubs. Secrets are stored
// as-is (raw bytes) for v0.1-alpha; age-encryption at rest per ADR-039 lands
// in v0.1-beta alongside the internal CA work.

type webhookListBearerInput struct {
	Authorization string `header:"Authorization"`
	Limit         int    `query:"limit" default:"50" minimum:"1" maximum:"500"`
	Cursor        string `query:"cursor" maxLength:"512"`
}

type webhookCreateBearerInput struct {
	Authorization string `header:"Authorization"`
	Body          WebhookCreateRequest
}

type webhookGetBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
}

type webhookUpdateBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
	Body          WebhookUpdateRequest
}

type webhookDeleteBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
}

type webhookTestBearerInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" minLength:"1" maxLength:"64"`
}

func registerWebhookListReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookList",
		Method:      http.MethodGet,
		Path:        "/api/v1/webhooks",
		Summary:     "List webhooks",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *webhookListBearerInput) (*WebhookListOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		offset := 0
		if input.Cursor != "" {
			if n, perr := parseIntSafe(input.Cursor); perr == nil {
				offset = n
			}
		}
		rows, total, err := svc.Repo().ListWebhooks(ctx, userID, offset, limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("WEBHOOK_LIST_FAILED")
		}
		records := make([]WebhookRecord, 0, len(rows))
		for i := range rows {
			records = append(records, webhookRowToRecord(&rows[i]))
		}
		next := offset + limit
		hasNext := next < total
		nextCursor := ""
		if hasNext {
			nextCursor = intToCursor(next)
		}
		return &WebhookListOutput{
			Body: WebhookListEnvelope{
				Data: records,
				Meta: WebhookListMeta{
					RequestID: "req_webhook_list",
					Page: WebhookPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerWebhookCreateReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/webhooks",
		Summary:     "Create a webhook",
		Tags:        []string{"Webhooks"},
		RequestBody: &huma.RequestBody{Description: "Webhook creation payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized, http.StatusBadRequest},
	}, func(ctx context.Context, input *webhookCreateBearerInput) (*WebhookCreateOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if len(input.Body.Events) == 0 {
			return nil, huma.Error400BadRequest("WEBHOOK_EVENTS_REQUIRED")
		}
		w, err := svc.Repo().CreateWebhook(ctx, input.Body.Name, input.Body.URL, input.Body.Events, []byte(input.Body.Secret), userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("WEBHOOK_CREATE_FAILED")
		}
		return &WebhookCreateOutput{
			Body: WebhookCreateEnvelope{
				Data: webhookRowToRecord(&w),
				Meta: WebhookCreateMeta{RequestID: "req_webhook_create"},
			},
		}, nil
	})
}

func registerWebhookGetReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/webhooks/{id}",
		Summary:     "Get a webhook",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *webhookGetBearerInput) (*WebhookGetOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		w, err := svc.Repo().GetWebhook(ctx, input.ID)
		if errors.Is(err, authrepo.ErrNotFound) {
			return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("WEBHOOK_GET_FAILED")
		}
		return &WebhookGetOutput{
			Body: WebhookGetEnvelope{
				Data: webhookRowToRecord(&w),
				Meta: WebhookGetMeta{RequestID: "req_webhook_get"},
			},
		}, nil
	})
}

func registerWebhookUpdateReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookUpdate",
		Method:      http.MethodPatch,
		Path:        "/api/v1/webhooks/{id}",
		Summary:     "Partially update a webhook",
		Tags:        []string{"Webhooks"},
		RequestBody: &huma.RequestBody{Description: "Partial update payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *webhookUpdateBearerInput) (*WebhookUpdateOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		var name, url *string
		if input.Body.Name != "" {
			name = &input.Body.Name
		}
		if input.Body.URL != "" {
			url = &input.Body.URL
		}
		if err := svc.Repo().UpdateWebhook(ctx, input.ID, name, url, input.Body.Events, input.Body.Enabled); err != nil {
			if errors.Is(err, authrepo.ErrNotFound) {
				return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
			}
			return nil, huma.Error500InternalServerError("WEBHOOK_UPDATE_FAILED")
		}
		w, err := svc.Repo().GetWebhook(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("WEBHOOK_GET_AFTER_UPDATE_FAILED")
		}
		return &WebhookUpdateOutput{
			Body: WebhookUpdateEnvelope{
				Data: webhookRowToRecord(&w),
				Meta: WebhookUpdateMeta{RequestID: "req_webhook_update"},
			},
		}, nil
	})
}

func registerWebhookDeleteReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/webhooks/{id}",
		Summary:     "Delete a webhook",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *webhookDeleteBearerInput) (*WebhookDeleteOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if err := svc.Repo().DeleteWebhook(ctx, input.ID); err != nil {
			if errors.Is(err, authrepo.ErrNotFound) {
				return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
			}
			return nil, huma.Error500InternalServerError("WEBHOOK_DELETE_FAILED")
		}
		return &WebhookDeleteOutput{
			Body: WebhookDeleteEnvelope{
				Data: WebhookDeleteData{},
				Meta: WebhookDeleteMeta{RequestID: "req_webhook_delete"},
			},
		}, nil
	})
}

func registerWebhookTestReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookTest",
		Method:      http.MethodPost,
		Path:        "/api/v1/webhooks/{id}/test",
		Summary:     "Send a synthetic test delivery",
		Description: "POSTs a ping payload to the webhook URL and records delivery. Secrets are stored raw in v0.1-alpha; signed delivery + retry worker land in v0.1-beta.",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *webhookTestBearerInput) (*WebhookTestOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		w, err := svc.Repo().GetWebhook(ctx, input.ID)
		if errors.Is(err, authrepo.ErrNotFound) {
			return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("WEBHOOK_GET_FAILED")
		}

		client := &http.Client{Timeout: 10 * time.Second}
		start := time.Now()
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, http.NoBody)
		req.Header.Set("User-Agent", "hellingd/"+hellingdSemver+" webhook-test")
		req.Header.Set("X-Helling-Event", "ping")
		resp, doErr := client.Do(req)
		latency := int(time.Since(start).Milliseconds())
		status := "delivered"
		statusCode := 0
		if doErr != nil {
			status = "failed"
		} else {
			defer func() { _ = resp.Body.Close() }()
			statusCode = resp.StatusCode
			if statusCode >= 400 {
				status = "failed"
			}
			_ = svc.Repo().MarkWebhookDelivered(ctx, w.ID, start)
		}

		return &WebhookTestOutput{
			Body: WebhookTestEnvelope{
				Data: WebhookTestData{
					ID:         w.ID,
					Status:     status,
					StatusCode: statusCode,
					Latency:    latency,
				},
				Meta: WebhookTestMeta{RequestID: "req_webhook_test"},
			},
		}, nil
	})
}

func webhookRowToRecord(w *authrepo.Webhook) WebhookRecord {
	rec := WebhookRecord{
		ID:      w.ID,
		Name:    w.Name,
		URL:     w.URL,
		Events:  w.Events,
		Enabled: w.Enabled,
	}
	if w.LastDeliveryAt.Valid {
		rec.LastSent = time.Unix(w.LastDeliveryAt.Int64, 0).UTC().Format(time.RFC3339)
	}
	return rec
}
