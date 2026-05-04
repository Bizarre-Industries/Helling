package store

// Operations CRUD. Mirrors the operations table from 001_initial.sql.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OperationStatus enumerates the lifecycle states of an async operation.
// Values match the enum in api/openapi.yaml.
type OperationStatus string

// Operation lifecycle states. Wire values match the OpenAPI enum.
const (
	OpStatusPending   OperationStatus = "pending"
	OpStatusRunning   OperationStatus = "running"
	OpStatusSuccess   OperationStatus = "success"
	OpStatusFailure   OperationStatus = "failure"
	OpStatusCancelled OperationStatus = "cancelled" //nolint:misspell // OpenAPI spec uses British spelling
)

// Operation mirrors a row in the operations table.
type Operation struct {
	ID        string
	UserID    int64
	Kind      string
	Target    string
	IncusOpID string
	Status    OperationStatus
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateOperation inserts a new operation in pending state and returns it.
// The id is a freshly generated UUIDv7 for time-ordered scans.
func (s *Store) CreateOperation(ctx context.Context, userID int64, kind, target, incusOpID string) (Operation, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Operation{}, fmt.Errorf("generating operation id: %w", err)
	}
	now := time.Now().UTC()
	op := Operation{
		ID:        id.String(),
		UserID:    userID,
		Kind:      kind,
		Target:    target,
		IncusOpID: incusOpID,
		Status:    OpStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO operations (id, user_id, kind, target, incus_op_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		op.ID, op.UserID, op.Kind, op.Target, op.IncusOpID, string(op.Status), now.Unix(), now.Unix(),
	)
	if err != nil {
		return Operation{}, fmt.Errorf("inserting operation: %w", err)
	}
	return op, nil
}

// GetOperation returns the operation owned by userID with the given id.
// Returns ErrNotFound if missing or owned by someone else.
func (s *Store) GetOperation(ctx context.Context, userID int64, id string) (Operation, error) {
	op, err := s.scanOperation(
		ctx,
		`SELECT id, user_id, kind, target, COALESCE(incus_op_id, ''), status, COALESCE(error, ''), created_at, updated_at
		 FROM operations WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Operation{}, ErrNotFound
	}
	return op, err
}

// ListOperations returns recent operations for a user, optionally filtered by
// status. limit caps the result count (0 falls back to 50, max 200).
func (s *Store) ListOperations(ctx context.Context, userID int64, status OperationStatus, limit int) ([]Operation, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var (
		rows *sql.Rows
		err  error
	)
	if status == "" {
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT id, user_id, kind, target, COALESCE(incus_op_id, ''), status, COALESCE(error, ''), created_at, updated_at
			 FROM operations WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`,
			userID, limit,
		)
	} else {
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT id, user_id, kind, target, COALESCE(incus_op_id, ''), status, COALESCE(error, ''), created_at, updated_at
			 FROM operations WHERE user_id = ? AND status = ? ORDER BY created_at DESC LIMIT ?`,
			userID, string(status), limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]Operation, 0, limit)
	for rows.Next() {
		op, err := scanOperationRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating operations: %w", err)
	}
	return out, nil
}

// ListActiveOperations returns operations still in pending or running state.
// Used by the background poller to advance state from Incus.
func (s *Store) ListActiveOperations(ctx context.Context) ([]Operation, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, user_id, kind, target, COALESCE(incus_op_id, ''), status, COALESCE(error, ''), created_at, updated_at
		 FROM operations WHERE status IN (?, ?) ORDER BY created_at`,
		string(OpStatusPending), string(OpStatusRunning),
	)
	if err != nil {
		return nil, fmt.Errorf("listing active operations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Operation
	for rows.Next() {
		op, err := scanOperationRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating active operations: %w", err)
	}
	return out, nil
}

// UpdateOperationStatus advances the operation's status and optionally records
// an error message. Bumps updated_at to now.
func (s *Store) UpdateOperationStatus(ctx context.Context, id string, status OperationStatus, errMsg string) error {
	now := time.Now().Unix()
	var errField any
	if errMsg != "" {
		errField = errMsg
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE operations SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(status), errField, now, id,
	)
	if err != nil {
		return fmt.Errorf("updating operation %s: %w", id, err)
	}
	return nil
}

func (s *Store) scanOperation(ctx context.Context, query string, args ...any) (Operation, error) {
	var op Operation
	var status string
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&op.ID, &op.UserID, &op.Kind, &op.Target, &op.IncusOpID, &status, &op.Error, &createdAt, &updatedAt,
	)
	if err != nil {
		return Operation{}, err
	}
	op.Status = OperationStatus(status)
	op.CreatedAt = time.Unix(createdAt, 0).UTC()
	op.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return op, nil
}

func scanOperationRow(rows *sql.Rows) (Operation, error) {
	var op Operation
	var status string
	var createdAt, updatedAt int64
	if err := rows.Scan(&op.ID, &op.UserID, &op.Kind, &op.Target, &op.IncusOpID, &status, &op.Error, &createdAt, &updatedAt); err != nil {
		return Operation{}, fmt.Errorf("scanning operation: %w", err)
	}
	op.Status = OperationStatus(status)
	op.CreatedAt = time.Unix(createdAt, 0).UTC()
	op.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return op, nil
}
