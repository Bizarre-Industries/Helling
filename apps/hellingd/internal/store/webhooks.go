package store

// Webhooks CRUD and delivery log.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Webhook mirrors a row in the webhooks table.
type Webhook struct {
	ID              string
	UserID          int64
	Name            string
	URL             string
	SecretEncrypted string
	Events          []string
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// WebhookDelivery mirrors a row in the webhook_deliveries table.
type WebhookDelivery struct {
	ID           string
	WebhookID    string
	EventID      string
	EventType    string
	Status       string
	StatusCode   *int
	ResponseBody *string
	Error        *string
	Attempt      int
	CreatedAt    time.Time
}

// CreateWebhook inserts a new webhook and returns it.
func (s *Store) CreateWebhook(ctx context.Context, userID int64, name, url, secret string, events []string) (Webhook, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Webhook{}, fmt.Errorf("generating webhook id: %w", err)
	}
	events, err = ValidateWebhookEvents(events)
	if err != nil {
		return Webhook{}, err
	}
	now := time.Now().UTC()
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return Webhook{}, fmt.Errorf("marshaling events: %w", err)
	}
	secretEncrypted, err := s.EncryptSecret(secret)
	if err != nil {
		return Webhook{}, fmt.Errorf("encrypting webhook secret: %w", err)
	}
	w := Webhook{
		ID:              id.String(),
		UserID:          userID,
		Name:            name,
		URL:             url,
		SecretEncrypted: secretEncrypted,
		Events:          events,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Webhook{}, fmt.Errorf("beginning webhook create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO webhooks (id, user_id, name, url, secret_encrypted, events, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		w.ID, w.UserID, w.Name, w.URL, w.SecretEncrypted, string(eventsJSON), now.Unix(), now.Unix(),
	)
	if err != nil {
		return Webhook{}, fmt.Errorf("inserting webhook: %w", err)
	}
	if err := replaceWebhookEventSubscriptions(ctx, tx, w.ID, events); err != nil {
		return Webhook{}, err
	}
	if err := tx.Commit(); err != nil {
		return Webhook{}, fmt.Errorf("committing webhook create: %w", err)
	}
	return w, nil
}

// GetWebhook returns a webhook by id, or ErrNotFound.
func (s *Store) GetWebhook(ctx context.Context, id string) (Webhook, error) {
	var w Webhook
	var createdAt, updatedAt int64
	var eventsJSON string
	var enabled int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, name, url, secret_encrypted, events, enabled, created_at, updated_at
		 FROM webhooks WHERE id = ?`, id,
	).Scan(&w.ID, &w.UserID, &w.Name, &w.URL, &w.SecretEncrypted, &eventsJSON, &enabled, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Webhook{}, ErrNotFound
	}
	if err != nil {
		return Webhook{}, fmt.Errorf("loading webhook: %w", err)
	}
	w.Enabled = enabled != 0
	w.CreatedAt = time.Unix(createdAt, 0).UTC()
	w.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
		w.Events = []string{}
	}
	return w, nil
}

// ListWebhooks returns all webhooks.
func (s *Store) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, url, secret_encrypted, events, enabled, created_at, updated_at
		 FROM webhooks ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing webhooks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var webhooks []Webhook
	for rows.Next() {
		var w Webhook
		var createdAt, updatedAt int64
		var eventsJSON string
		var enabled int
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.URL, &w.SecretEncrypted, &eventsJSON, &enabled, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning webhook: %w", err)
		}
		w.Enabled = enabled != 0
		w.CreatedAt = time.Unix(createdAt, 0).UTC()
		w.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
			w.Events = []string{}
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

// UpdateWebhook updates mutable fields of a webhook.
func (s *Store) UpdateWebhook(ctx context.Context, id, name, url, secret string, events []string, enabled bool) error {
	events, err := ValidateWebhookEvents(events)
	if err != nil {
		return err
	}
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshaling events: %w", err)
	}
	secretEncrypted, err := s.EncryptSecret(secret)
	if err != nil {
		return fmt.Errorf("encrypting webhook secret: %w", err)
	}
	en := 0
	if enabled {
		en = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning webhook update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`UPDATE webhooks SET name = ?, url = ?, secret_encrypted = ?, events = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		name, url, secretEncrypted, string(eventsJSON), en, time.Now().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("updating webhook: %w", err)
	}
	if err := replaceWebhookEventSubscriptions(ctx, tx, id, events); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing webhook update: %w", err)
	}
	return nil
}

func normalizeWebhookEvents(events []string) []string {
	out := make([]string, 0, len(events))
	seen := make(map[string]struct{}, len(events))
	for _, eventType := range events {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" {
			continue
		}
		if _, ok := seen[eventType]; ok {
			continue
		}
		seen[eventType] = struct{}{}
		out = append(out, eventType)
	}
	return out
}

// ValidateWebhookEvents normalizes and validates webhook event subscriptions.
func ValidateWebhookEvents(events []string) ([]string, error) {
	events = normalizeWebhookEvents(events)
	if len(events) == 0 {
		return nil, errors.New("webhook events cannot be empty")
	}
	for _, eventType := range events {
		if !webhookEventTypeAllowed(eventType) {
			return nil, fmt.Errorf("unsupported webhook event type %q", eventType)
		}
	}
	return events, nil
}

func webhookEventTypeAllowed(eventType string) bool {
	if eventType == "*" {
		return true
	}
	if _, ok := knownWebhookEventTypes[eventType]; ok {
		return true
	}
	prefix := ""
	if strings.HasSuffix(eventType, ".*") {
		prefix = strings.TrimSuffix(eventType, "*")
	} else if strings.HasSuffix(eventType, ".") {
		prefix = eventType
	}
	if prefix == "" {
		return false
	}
	for known := range knownWebhookEventTypes {
		if strings.HasPrefix(known, prefix) {
			return true
		}
	}
	return false
}

var knownWebhookEventTypes = map[string]struct{}{
	"auth.login.fail":          {},
	"auth.login.ok":            {},
	"auth.token.revoked":       {},
	"backup.completed":         {},
	"backup.failed":            {},
	"cluster.node.joined":      {},
	"cluster.node.left":        {},
	"container.created":        {},
	"container.deleted":        {},
	"container.started":        {},
	"container.stopped":        {},
	"firewall.rule.created":    {},
	"firewall.rule.deleted":    {},
	"instance.created":         {},
	"instance.deleted":         {},
	"instance.started":         {},
	"instance.stopped":         {},
	"instance.updated":         {},
	"schedule.completed":       {},
	"schedule.created":         {},
	"schedule.deleted":         {},
	"schedule.failed":          {},
	"schedule.started":         {},
	"schedule.updated":         {},
	"snapshot.created":         {},
	"snapshot.deleted":         {},
	"system.upgrade.completed": {},
	"system.upgrade.failed":    {},
	"system.upgrade.started":   {},
	"user.created":             {},
	"user.deleted":             {},
	"user.updated":             {},
	"warning.raised":           {},
	"warning.resolved":         {},
	"webhook.created":          {},
	"webhook.deleted":          {},
	"webhook.delivery.fail":    {},
	"webhook.delivery.ok":      {},
	"webhook.test":             {},
	"webhook.updated":          {},
}

func replaceWebhookEventSubscriptions(ctx context.Context, tx *sql.Tx, webhookID string, events []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM webhook_events WHERE webhook_id = ?`, webhookID); err != nil {
		return fmt.Errorf("clearing webhook event subscriptions: %w", err)
	}
	for _, eventType := range normalizeWebhookEvents(events) {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO webhook_events (webhook_id, event_type) VALUES (?, ?)`,
			webhookID, eventType,
		); err != nil {
			return fmt.Errorf("inserting webhook event subscription: %w", err)
		}
	}
	return nil
}

// DeleteWebhook removes a webhook by id. Idempotent.
func (s *Store) DeleteWebhook(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}
	return nil
}

// CreateWebhookDelivery logs a delivery attempt.
func (s *Store) CreateWebhookDelivery(ctx context.Context, webhookID, eventID, eventType, status string, statusCode *int, responseBody, errMsg *string, attempt int) (WebhookDelivery, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return WebhookDelivery{}, fmt.Errorf("generating delivery id: %w", err)
	}
	now := time.Now().UTC()
	d := WebhookDelivery{
		ID:           id.String(),
		WebhookID:    webhookID,
		EventID:      eventID,
		EventType:    eventType,
		Status:       status,
		StatusCode:   statusCode,
		ResponseBody: responseBody,
		Error:        errMsg,
		Attempt:      attempt,
		CreatedAt:    now,
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, event_type, status, status_code, response_body, error, attempt, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.WebhookID, d.EventType, d.EventID, d.EventType, d.Status, statusCode, responseBody, errMsg, d.Attempt, now.Unix(),
	)
	if err != nil {
		return WebhookDelivery{}, fmt.Errorf("inserting webhook delivery: %w", err)
	}
	_, _ = s.db.ExecContext(ctx,
		`DELETE FROM webhook_deliveries
		 WHERE webhook_id = ?
		   AND id NOT IN (
		     SELECT id FROM webhook_deliveries
		     WHERE webhook_id = ?
		     ORDER BY created_at DESC, attempt DESC, id DESC
		     LIMIT 100
		   )`,
		webhookID, webhookID,
	)
	return d, nil
}

// ListWebhookDeliveries returns recent deliveries for a webhook.
func (s *Store) ListWebhookDeliveries(ctx context.Context, webhookID string, limit int) ([]WebhookDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, webhook_id, COALESCE(event_id, ''), COALESCE(event_type, event), status, status_code, response_body, error, attempt, created_at
		 FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC, attempt DESC, id DESC LIMIT ?`,
		webhookID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing webhook deliveries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var deliveries []WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		var createdAt int64
		var statusCode sql.NullInt64
		var responseBody, errMsg sql.NullString
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventID, &d.EventType, &d.Status, &statusCode, &responseBody, &errMsg, &d.Attempt, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning delivery: %w", err)
		}
		if statusCode.Valid {
			sc := int(statusCode.Int64)
			d.StatusCode = &sc
		}
		if responseBody.Valid {
			d.ResponseBody = &responseBody.String
		}
		if errMsg.Valid {
			d.Error = &errMsg.String
		}
		d.CreatedAt = time.Unix(createdAt, 0).UTC()
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}
