package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDispatchEventWebhooksReschedulesRetriesAndRecordsDeliveries(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	u := seedAdminUser(t, st)
	hook, err := st.CreateWebhook(t.Context(), u.ID, "ops", "https://example.com/hook", "top-secret-value", []string{"schedule.created"})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	var attempts int
	srv.cfg.WebhookRetryDelays = []time.Duration{0, 0, 0}
	srv.cfg.WebhookDelivery = func(_ context.Context, destURL, secret string, body []byte) (*int, *string, error) {
		attempts++
		if destURL != "https://example.com/hook" {
			t.Fatalf("destURL: got %q", destURL)
		}
		if secret != "top-secret-value" {
			t.Fatalf("secret: got %q", secret)
		}
		if len(body) == 0 {
			t.Fatal("empty webhook body")
		}
		if attempts < 3 {
			status := 500
			resp := "retry"
			return &status, &resp, errors.New("temporary failure")
		}
		status := 204
		resp := ""
		return &status, &resp, nil
	}

	ev, err := st.CreateEvent(t.Context(), "schedule.created", "helling", "schedule-1", nil)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	for i := 0; i < 3; i++ {
		jobs, err := st.ClaimEventOutbox(t.Context(), ev.ID)
		if err != nil {
			t.Fatalf("ClaimEventOutbox %d: %v", i+1, err)
		}
		if len(jobs) != 1 {
			t.Fatalf("ClaimEventOutbox %d returned %#v", i+1, jobs)
		}
		srv.dispatchEventWebhooks(t.Context(), &jobs[0])
	}

	if attempts != 3 {
		t.Fatalf("attempts: got %d want 3", attempts)
	}
	deliveries, err := st.ListWebhookDeliveries(t.Context(), hook.ID, 10)
	if err != nil {
		t.Fatalf("ListWebhookDeliveries: %v", err)
	}
	if len(deliveries) != 3 {
		t.Fatalf("deliveries: got %d want 3", len(deliveries))
	}
	if deliveries[0].Status != outcomeSuccess || deliveries[0].Attempt != 3 {
		t.Fatalf("latest delivery: %#v", deliveries[0])
	}
	if deliveries[1].Status != outcomeFailed || deliveries[1].Attempt != 2 {
		t.Fatalf("second delivery: %#v", deliveries[1])
	}
	if deliveries[2].Status != outcomeFailed || deliveries[2].Attempt != 1 {
		t.Fatalf("first delivery: %#v", deliveries[2])
	}
}
