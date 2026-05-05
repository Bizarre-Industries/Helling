package server

import (
	"net"
	"strings"
	"testing"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

func TestValidateWebhookSecretEnforcesOpenAPILength(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{name: "too short", secret: strings.Repeat("a", 15), wantErr: true},
		{name: "minimum", secret: strings.Repeat("a", 16), wantErr: false},
		{name: "maximum", secret: strings.Repeat("a", 512), wantErr: false},
		{name: "too long", secret: strings.Repeat("a", 513), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateWebhookSecret(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateWebhookSecret() error=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockedWebhookIPRejectsSpecialUseRanges(t *testing.T) {
	t.Parallel()
	tests := []string{
		"100.64.0.1",
		"198.18.0.1",
		"192.88.99.2",
		"192.0.0.8",
		"240.0.0.1",
		"255.255.255.255",
		"64:ff9b:1::1",
		"100:0:0:1::1",
		"2002::1",
		"2001:db8::1",
		"3fff::1",
		"5f00::1",
		"fc00::1",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			if !blockedWebhookIP(net.ParseIP(raw)) {
				t.Fatalf("blockedWebhookIP(%s) = false, want true", raw)
			}
		})
	}
}

func TestApplyWebhookUpdateValidatesReplacementSecret(t *testing.T) {
	t.Parallel()
	row := store.Webhook{
		ID:      "hook-1",
		Name:    "ops",
		URL:     "https://hooks.example.com/helling",
		Events:  []string{"instance.started"},
		Enabled: true,
	}
	short := "too-short"

	_, err := applyWebhookUpdate(t.Context(), &row, updateWebhookRequest{Secret: &short}, "valid-secret-1234")
	if err == nil {
		t.Fatal("applyWebhookUpdate accepted a short replacement secret")
	}
}

func TestWebhookDeliveryToResponseIncludesFailureDetails(t *testing.T) {
	t.Parallel()
	statusCode := 503
	errMsg := "webhook returned status 503"
	row := store.WebhookDelivery{
		ID:         "delivery-1",
		WebhookID:  "webhook-1",
		EventID:    "event-1",
		EventType:  "webhook.test",
		Status:     outcomeFailed,
		StatusCode: &statusCode,
		Error:      &errMsg,
		Attempt:    1,
	}

	resp := webhookDeliveryToResponse(&row)
	if resp.HTTPStatus == nil || *resp.HTTPStatus != statusCode {
		t.Fatalf("HTTPStatus = %v, want %d", resp.HTTPStatus, statusCode)
	}
	if resp.Error == nil || *resp.Error != errMsg {
		t.Fatalf("Error = %v, want %q", resp.Error, errMsg)
	}
}
