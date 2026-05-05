package store

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

// Event mirrors a persisted Helling event.
type Event struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Time    time.Time      `json:"time"`
	Source  string         `json:"source"`
	Subject string         `json:"subject"`
	Data    map[string]any `json:"data"`
}

// OutboxEvent is a claimed event plus its durable delivery attempt number.
type OutboxEvent struct {
	Event
	OutboxID  string
	WebhookID string
	Attempt   int
}

const (
	outboxProcessingLease = 2 * time.Minute
	eventPruneBatchLimit  = 5000
	eventPruneMaxBatches  = 20
)

// CreateEvent persists an event and queues it for webhook fan-out.
func (s *Store) CreateEvent(ctx context.Context, eventType, source, subject string, data map[string]any) (Event, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Event{}, fmt.Errorf("generating event id: %w", err)
	}
	if data == nil {
		data = map[string]any{}
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("marshaling event data: %w", err)
	}
	now := time.Now().UTC()
	ev := Event{ID: id.String(), Type: eventType, Time: now, Source: source, Subject: subject, Data: data}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("beginning event transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO events (id, type, source, subject, data_json, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.Type, ev.Source, ev.Subject, string(dataJSON), now.Unix(),
	)
	if err != nil {
		return Event{}, fmt.Errorf("inserting event: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO event_outbox (id, event_id, webhook_id, status, attempts, next_run_at, created_at, updated_at)
		 SELECT DISTINCT ? || ':' || w.id, ?, w.id, 'pending', 0, ?, ?, ?
		 FROM webhook_events AS we
		 JOIN webhooks AS w ON w.id = we.webhook_id
		 WHERE w.enabled = 1
		   AND (
		     we.event_type = '*'
		     OR we.event_type = ?
		     OR (substr(we.event_type, -2) = '.*' AND ? LIKE substr(we.event_type, 1, length(we.event_type)-1) || '%')
		     OR (substr(we.event_type, -1) = '.' AND ? LIKE we.event_type || '%')
		   )`,
		ev.ID, ev.ID, now.Unix(), now.Unix(), now.Unix(), ev.Type, ev.Type, ev.Type,
	); err != nil {
		return Event{}, fmt.Errorf("inserting event outbox row: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("committing event: %w", err)
	}
	return ev, nil
}

// ListEvents returns recent events, optionally filtered by exact type or type prefix.
func (s *Store) ListEvents(ctx context.Context, limit int, typeFilter, since string) ([]Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	exactType := typeFilter
	prefixType := ""
	if typeFilter != "" {
		if strings.HasSuffix(typeFilter, ".") {
			prefixType = typeFilter + "%"
			exactType = ""
		}
	}
	order := "ORDER BY created_at DESC, id DESC"
	if since != "" {
		order = "ORDER BY id ASC"
	}
	query := `SELECT id, type, source, subject, data_json, created_at
		 FROM events
		 WHERE ((? = '' AND ? = '') OR (? <> '' AND type = ?) OR (? <> '' AND type LIKE ?))
		   AND (? = '' OR id > ?)
		 ` + order + ` LIMIT ?` // #nosec G202 -- order is selected from fixed constants above.
	rows, err := s.db.QueryContext(ctx,
		query,
		exactType, prefixType, exactType, exactType, prefixType, prefixType, since, since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Event
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// ClaimPendingOutboxEvents atomically marks due or stale-processing events as
// processing and returns the claimed events for webhook fan-out.
func (s *Store) ClaimPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	now := time.Now().UTC().Unix()
	staleBefore := time.Now().UTC().Add(-outboxProcessingLease).Unix()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning outbox claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	claimed, err := claimOutboxEventIDs(ctx, tx, now, staleBefore, limit)
	if err != nil {
		return nil, err
	}
	if len(claimed) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("committing empty outbox claim: %w", err)
		}
		return nil, nil
	}
	out, err := listOutboxEvents(ctx, tx, claimed)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing outbox claim: %w", err)
	}
	return out, nil
}

type claimedOutbox struct {
	EventID   string
	WebhookID string
	Attempt   int
}

func claimOutboxEventIDs(ctx context.Context, tx *sql.Tx, now, staleBefore int64, limit int) (map[string]claimedOutbox, error) {
	rows, err := tx.QueryContext(ctx,
		`UPDATE event_outbox
		 SET status = 'processing',
		     attempts = attempts + 1,
		     next_run_at = NULL,
		     updated_at = ?
		 WHERE id IN (
		   SELECT id
		   FROM event_outbox
		   WHERE webhook_id IS NOT NULL
		     AND (
		       (status = 'pending' AND next_run_at <= ?)
		       OR (status = 'processing' AND updated_at <= ?)
		     )
		   ORDER BY next_run_at, created_at, id
		   LIMIT ?
		 )
		 RETURNING id, event_id, webhook_id, attempts`,
		now, now, staleBefore, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("claiming pending outbox events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	claimed := make(map[string]claimedOutbox)
	for rows.Next() {
		var outboxID, eventID, webhookID string
		var attempt int
		if err := rows.Scan(&outboxID, &eventID, &webhookID, &attempt); err != nil {
			return nil, fmt.Errorf("scanning claimed outbox event: %w", err)
		}
		claimed[outboxID] = claimedOutbox{EventID: eventID, WebhookID: webhookID, Attempt: attempt}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func listOutboxEvents(ctx context.Context, tx *sql.Tx, claimed map[string]claimedOutbox) ([]OutboxEvent, error) {
	ids := make([]string, 0, len(claimed))
	args := make([]any, 0, len(claimed))
	for outboxID := range claimed {
		ids = append(ids, outboxID)
		args = append(args, outboxID)
	}
	query := `SELECT ob.id, ob.webhook_id, e.id, e.type, e.source, e.subject, e.data_json, e.created_at
		 FROM event_outbox AS ob
		 JOIN events AS e ON e.id = ob.event_id
		 WHERE ob.id IN (` + placeholders(len(ids)) + `)
		 ORDER BY e.created_at ASC, e.id ASC, ob.id ASC` // #nosec G202 -- placeholders are generated as "?" for claimed UUIDs only.
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing claimed outbox events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]OutboxEvent, 0, len(ids))
	for rows.Next() {
		var outboxID, webhookID, dataJSON string
		var ev Event
		var createdAt int64
		if err := rows.Scan(&outboxID, &webhookID, &ev.ID, &ev.Type, &ev.Source, &ev.Subject, &dataJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning outbox event: %w", err)
		}
		if err := json.Unmarshal([]byte(dataJSON), &ev.Data); err != nil {
			ev.Data = map[string]any{}
		}
		ev.Time = time.Unix(createdAt, 0).UTC()
		claim := claimed[outboxID]
		out = append(out, OutboxEvent{Event: ev, OutboxID: outboxID, WebhookID: webhookID, Attempt: claim.Attempt})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// MarkEventOutboxDelivered marks webhook fan-out complete for an event.
func (s *Store) MarkEventOutboxDelivered(ctx context.Context, eventID string) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx,
		`UPDATE event_outbox SET status = 'delivered', updated_at = ? WHERE event_id = ?`,
		now, eventID,
	)
	if err != nil {
		return fmt.Errorf("marking event outbox delivered: %w", err)
	}
	return nil
}

// MarkOutboxDelivered marks one webhook delivery job complete.
func (s *Store) MarkOutboxDelivered(ctx context.Context, outboxID string) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx,
		`UPDATE event_outbox SET status = 'delivered', updated_at = ? WHERE id = ?`,
		now, outboxID,
	)
	if err != nil {
		return fmt.Errorf("marking outbox delivered: %w", err)
	}
	return nil
}

// MarkEventOutboxPending reschedules every webhook job for one event.
func (s *Store) MarkEventOutboxPending(ctx context.Context, eventID string, nextRun time.Time) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx,
		`UPDATE event_outbox
		 SET status = 'pending', next_run_at = ?, updated_at = ?
		 WHERE event_id = ?`,
		nextRun.UTC().Unix(), now, eventID,
	)
	if err != nil {
		return fmt.Errorf("marking event outbox pending: %w", err)
	}
	return nil
}

// MarkOutboxPending reschedules one webhook job for later fan-out.
func (s *Store) MarkOutboxPending(ctx context.Context, outboxID string, nextRun time.Time) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx,
		`UPDATE event_outbox
		 SET status = 'pending', next_run_at = ?, updated_at = ?
		 WHERE id = ?`,
		nextRun.UTC().Unix(), now, outboxID,
	)
	if err != nil {
		return fmt.Errorf("marking outbox pending: %w", err)
	}
	return nil
}

// ClaimEventOutbox marks eagerly queued webhook jobs for one event in-flight.
func (s *Store) ClaimEventOutbox(ctx context.Context, eventID string) ([]OutboxEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning event outbox claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	claimed, err := claimEventOutbox(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	if len(claimed) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("committing empty event outbox claim: %w", err)
		}
		return nil, nil
	}
	events, err := listOutboxEvents(ctx, tx, claimed)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing event outbox claim: %w", err)
	}
	return events, nil
}

// MarkEventOutboxProcessing marks one eagerly queued event as in-flight.
func (s *Store) MarkEventOutboxProcessing(ctx context.Context, eventID string) (bool, error) {
	claimed, err := s.ClaimEventOutbox(ctx, eventID)
	if err != nil {
		return false, err
	}
	return len(claimed) > 0, nil
}

func claimEventOutbox(ctx context.Context, tx *sql.Tx, eventID string) (map[string]claimedOutbox, error) {
	now := time.Now().UTC().Unix()
	staleBefore := time.Now().UTC().Add(-outboxProcessingLease).Unix()
	rows, err := tx.QueryContext(ctx,
		`UPDATE event_outbox
		 SET status = 'processing', attempts = attempts + 1, next_run_at = NULL, updated_at = ?
		 WHERE event_id = ?
		   AND webhook_id IS NOT NULL
		   AND (
		     (status = 'pending' AND (next_run_at IS NULL OR next_run_at <= ?))
		     OR (status = 'processing' AND updated_at <= ?)
		   )
		 RETURNING id, event_id, webhook_id, attempts`,
		now, eventID, now, staleBefore,
	)
	if err != nil {
		return nil, fmt.Errorf("marking event outbox processing: %w", err)
	}
	defer func() { _ = rows.Close() }()

	claimed := make(map[string]claimedOutbox)
	for rows.Next() {
		var outboxID, returnedEventID, webhookID string
		var attempt int
		if err := rows.Scan(&outboxID, &returnedEventID, &webhookID, &attempt); err != nil {
			return nil, fmt.Errorf("scanning event outbox claim: %w", err)
		}
		claimed[outboxID] = claimedOutbox{EventID: returnedEventID, WebhookID: webhookID, Attempt: attempt}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return claimed, nil
}

// PruneEvents keeps recent event replay bounded.
func (s *Store) PruneEvents(ctx context.Context, keepRows int, maxAge time.Duration) error {
	if keepRows <= 0 {
		keepRows = 10000
	}
	cutoff := time.Now().UTC().Add(-maxAge).Unix()
	if err := s.pruneEventsBefore(ctx, cutoff); err != nil {
		return err
	}
	return s.pruneEventsAboveLimit(ctx, keepRows)
}

func (s *Store) pruneEventsBefore(ctx context.Context, cutoff int64) error {
	for range eventPruneMaxBatches {
		affected, err := s.deleteEventBatch(ctx,
			`SELECT id FROM events WHERE created_at < ? ORDER BY created_at ASC, id ASC LIMIT ?`,
			cutoff, eventPruneBatchLimit,
		)
		if err != nil || affected < eventPruneBatchLimit {
			return err
		}
	}
	return nil
}

func (s *Store) pruneEventsAboveLimit(ctx context.Context, keepRows int) error {
	var boundaryID string
	var boundaryCreatedAt int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id, created_at
		 FROM events
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1 OFFSET ?`,
		keepRows-1,
	).Scan(&boundaryID, &boundaryCreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("finding event prune boundary: %w", err)
	}
	for range eventPruneMaxBatches {
		affected, err := s.deleteEventBatch(ctx,
			`SELECT id FROM events
			 WHERE created_at < ? OR (created_at = ? AND id < ?)
			 ORDER BY created_at ASC, id ASC
			 LIMIT ?`,
			boundaryCreatedAt, boundaryCreatedAt, boundaryID, eventPruneBatchLimit,
		)
		if err != nil || affected < eventPruneBatchLimit {
			return err
		}
	}
	return nil
}

func (s *Store) deleteEventBatch(ctx context.Context, selector string, args ...any) (int64, error) {
	query := `DELETE FROM events WHERE id IN (` + selector + `)` // #nosec G202 -- selector is a fixed package-local SQL fragment.
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("pruning events: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("checking pruned events: %w", err)
	}
	return affected, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(rows eventScanner) (Event, error) {
	var ev Event
	var dataJSON string
	var createdAt int64
	if err := rows.Scan(&ev.ID, &ev.Type, &ev.Source, &ev.Subject, &dataJSON, &createdAt); err != nil {
		return Event{}, fmt.Errorf("scanning event: %w", err)
	}
	if err := json.Unmarshal([]byte(dataJSON), &ev.Data); err != nil {
		ev.Data = map[string]any{}
	}
	ev.Time = time.Unix(createdAt, 0).UTC()
	return ev, nil
}
