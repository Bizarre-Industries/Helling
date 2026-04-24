package authrepo

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

// Webhook mirrors the webhooks table row.
type Webhook struct {
	ID              string
	Name            string
	URL             string
	Events          []string // decoded from events_json
	SecretEncrypted []byte
	Enabled         bool
	LastDeliveryAt  sql.NullInt64
	CreatedBy       string
	CreatedAt       int64
	UpdatedAt       int64
}

// CreateWebhook inserts a new webhook row. `events` is JSON-serialized.
// `secretEncrypted` carries the encrypted HMAC secret; callers own encryption.
func (r *Repo) CreateWebhook(ctx context.Context, name, url string, events []string, secretEncrypted []byte, createdBy string) (Webhook, error) {
	if createdBy == "" {
		return Webhook{}, errors.New("authrepo: createdBy required")
	}
	id := uuid.NewString()
	now := r.now().Unix()
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return Webhook{}, fmt.Errorf("authrepo: marshal events: %w", err)
	}
	const q = `INSERT INTO webhooks (id, name, url, events_json, secret_encrypted, enabled, created_by, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, q, id, name, url, string(eventsJSON), secretEncrypted, createdBy, now, now); err != nil {
		return Webhook{}, fmt.Errorf("authrepo: insert webhook: %w", err)
	}
	return Webhook{
		ID: id, Name: name, URL: url, Events: events, SecretEncrypted: secretEncrypted,
		Enabled: true, CreatedBy: createdBy, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GetWebhook returns a single webhook by id, or ErrNotFound.
func (r *Repo) GetWebhook(ctx context.Context, id string) (Webhook, error) {
	const q = `SELECT id, name, url, events_json, secret_encrypted, enabled, last_delivery_at, created_by, created_at, updated_at
	           FROM webhooks WHERE id = ?`
	row := r.db.QueryRowContext(ctx, q, id)
	return scanWebhook(row.Scan)
}

// ListWebhooks returns the user's webhooks ordered by created_at ASC.
func (r *Repo) ListWebhooks(ctx context.Context, createdBy string, offset, limit int) ([]Webhook, int, error) {
	const countQ = `SELECT count(*) FROM webhooks WHERE created_by = ?`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, createdBy).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("authrepo: count webhooks: %w", err)
	}
	const q = `SELECT id, name, url, events_json, secret_encrypted, enabled, last_delivery_at, created_by, created_at, updated_at
	           FROM webhooks WHERE created_by = ?
	           ORDER BY created_at ASC, id ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, createdBy, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("authrepo: list webhooks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make([]Webhook, 0, limit)
	for rows.Next() {
		w, err := scanWebhook(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("authrepo: rows webhooks: %w", err)
	}
	return out, total, nil
}

// UpdateWebhook applies partial updates. Pass nil for unchanged fields.
func (r *Repo) UpdateWebhook(ctx context.Context, id string, name, url *string, events []string, enabled *bool) error {
	sets := []string{}
	args := []any{}
	if name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *name)
	}
	if url != nil {
		sets = append(sets, "url = ?")
		args = append(args, *url)
	}
	if events != nil {
		j, err := json.Marshal(events)
		if err != nil {
			return fmt.Errorf("authrepo: marshal events: %w", err)
		}
		sets = append(sets, "events_json = ?")
		args = append(args, string(j))
	}
	if enabled != nil {
		v := 0
		if *enabled {
			v = 1
		}
		sets = append(sets, "enabled = ?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, r.now().Unix())
	args = append(args, id)
	// G202: sets is a controlled allow-list of column assignments built above.
	q := "UPDATE webhooks SET " + strings.Join(sets, ", ") + " WHERE id = ?" //nolint:gosec // allow-listed set clauses
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("authrepo: update webhook: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteWebhook removes a webhook row. Cascades webhook_deliveries via schema.
func (r *Repo) DeleteWebhook(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("authrepo: delete webhook: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkWebhookDelivered stamps last_delivery_at.
func (r *Repo) MarkWebhookDelivered(ctx context.Context, id string, at time.Time) error {
	const q = `UPDATE webhooks SET last_delivery_at = ?, updated_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, at.Unix(), r.now().Unix(), id); err != nil {
		return fmt.Errorf("authrepo: mark webhook delivered: %w", err)
	}
	return nil
}

func scanWebhook(scan func(dest ...any) error) (Webhook, error) {
	var w Webhook
	var eventsJSON string
	var enabledInt int
	err := scan(&w.ID, &w.Name, &w.URL, &eventsJSON, &w.SecretEncrypted, &enabledInt, &w.LastDeliveryAt, &w.CreatedBy, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Webhook{}, ErrNotFound
	}
	if err != nil {
		return Webhook{}, fmt.Errorf("authrepo: scan webhook: %w", err)
	}
	w.Enabled = enabledInt == 1
	if eventsJSON != "" {
		if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
			return Webhook{}, fmt.Errorf("authrepo: parse events_json: %w", err)
		}
	}
	return w, nil
}
