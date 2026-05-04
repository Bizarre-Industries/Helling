package store

// Schedules CRUD. systemd timer-backed schedules per ADR-017.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Schedule mirrors a row in the schedules table.
type Schedule struct {
	ID        string
	UserID    int64
	Name      string
	Kind      string
	Target    string
	CronExpr  string
	Enabled   bool
	LastRunAt *time.Time
	NextRunAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateSchedule inserts a new schedule and returns it.
func (s *Store) CreateSchedule(ctx context.Context, userID int64, name, kind, target, cronExpr string) (Schedule, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Schedule{}, fmt.Errorf("generating schedule id: %w", err)
	}
	now := time.Now().UTC()
	sch := Schedule{
		ID:        id.String(),
		UserID:    userID,
		Name:      name,
		Kind:      kind,
		Target:    target,
		CronExpr:  cronExpr,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO schedules (id, user_id, name, kind, target, cron_expr, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		sch.ID, sch.UserID, sch.Name, sch.Kind, sch.Target, sch.CronExpr, now.Unix(), now.Unix(),
	)
	if err != nil {
		return Schedule{}, fmt.Errorf("inserting schedule: %w", err)
	}
	return sch, nil
}

// GetSchedule returns a schedule by id, or ErrNotFound.
func (s *Store) GetSchedule(ctx context.Context, id string) (Schedule, error) {
	var sch Schedule
	var createdAt, updatedAt int64
	var lastRunAt, nextRunAt sql.NullInt64
	var enabled int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, name, kind, target, cron_expr, enabled, last_run_at, next_run_at, created_at, updated_at
		 FROM schedules WHERE id = ?`, id,
	).Scan(&sch.ID, &sch.UserID, &sch.Name, &sch.Kind, &sch.Target, &sch.CronExpr, &enabled,
		&lastRunAt, &nextRunAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Schedule{}, ErrNotFound
	}
	if err != nil {
		return Schedule{}, fmt.Errorf("loading schedule: %w", err)
	}
	sch.Enabled = enabled != 0
	if lastRunAt.Valid {
		ts := time.Unix(lastRunAt.Int64, 0).UTC()
		sch.LastRunAt = &ts
	}
	if nextRunAt.Valid {
		ts := time.Unix(nextRunAt.Int64, 0).UTC()
		sch.NextRunAt = &ts
	}
	sch.CreatedAt = time.Unix(createdAt, 0).UTC()
	sch.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return sch, nil
}

// ListSchedules returns all schedules, optionally filtered by kind.
func (s *Store) ListSchedules(ctx context.Context, kind string) ([]Schedule, error) {
	var rows *sql.Rows
	var err error
	if kind == "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, user_id, name, kind, target, cron_expr, enabled, last_run_at, next_run_at, created_at, updated_at
			 FROM schedules ORDER BY created_at DESC`)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, user_id, name, kind, target, cron_expr, enabled, last_run_at, next_run_at, created_at, updated_at
			 FROM schedules WHERE kind = ? ORDER BY created_at DESC`, kind)
	}
	if err != nil {
		return nil, fmt.Errorf("listing schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanSchedules(rows)
}

// UpdateSchedule updates mutable fields of a schedule.
func (s *Store) UpdateSchedule(ctx context.Context, id, name, cronExpr string, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE schedules SET name = ?, cron_expr = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		name, cronExpr, en, time.Now().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("updating schedule: %w", err)
	}
	return nil
}

// TouchScheduleRun bumps last_run_at and next_run_at.
func (s *Store) TouchScheduleRun(ctx context.Context, id string, nextRunAt time.Time) error {
	now := time.Now().Unix()
	next := nextRunAt.Unix()
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE schedules SET last_run_at = ?, next_run_at = ?, updated_at = ? WHERE id = ?`,
		now, next, now, id,
	)
	if err != nil {
		return fmt.Errorf("touching schedule run: %w", err)
	}
	return nil
}

// DeleteSchedule removes a schedule by id. Idempotent.
func (s *Store) DeleteSchedule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM schedules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting schedule: %w", err)
	}
	return nil
}

func scanSchedules(rows *sql.Rows) ([]Schedule, error) {
	var schedules []Schedule
	for rows.Next() {
		var sch Schedule
		var createdAt, updatedAt int64
		var lastRunAt, nextRunAt sql.NullInt64
		var enabled int
		if err := rows.Scan(&sch.ID, &sch.UserID, &sch.Name, &sch.Kind, &sch.Target, &sch.CronExpr, &enabled,
			&lastRunAt, &nextRunAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning schedule: %w", err)
		}
		sch.Enabled = enabled != 0
		if lastRunAt.Valid {
			ts := time.Unix(lastRunAt.Int64, 0).UTC()
			sch.LastRunAt = &ts
		}
		if nextRunAt.Valid {
			ts := time.Unix(nextRunAt.Int64, 0).UTC()
			sch.NextRunAt = &ts
		}
		sch.CreatedAt = time.Unix(createdAt, 0).UTC()
		sch.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		schedules = append(schedules, sch)
	}
	return schedules, rows.Err()
}
