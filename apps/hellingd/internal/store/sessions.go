package store

// Sessions CRUD. Tokens are stored as the sha256 hex of the random token,
// so a database read alone cannot resurrect an active session.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Session mirrors the sessions table.
type Session struct {
	ID         string // sha256 hex of the raw session token
	UserID     int64
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

// CreateSession inserts a new session row keyed by tokenHashHex (sha256 of the
// raw cookie value). The caller hands the raw token to the user; the server
// only ever stores the hash.
func (s *Store) CreateSession(ctx context.Context, tokenHashHex string, userID int64, ttl time.Duration) (Session, error) {
	now := time.Now().UTC()
	expires := now.Add(ttl)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, created_at, expires_at, last_seen_at) VALUES (?, ?, ?, ?, ?)`,
		tokenHashHex, userID, now.Unix(), expires.Unix(), now.Unix(),
	)
	if err != nil {
		return Session{}, fmt.Errorf("inserting session: %w", err)
	}
	return Session{
		ID:         tokenHashHex,
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  expires,
		LastSeenAt: now,
	}, nil
}

// GetSessionByTokenHash returns a non-expired session for the given hash, or
// ErrNotFound if it doesn't exist or has expired.
func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHashHex string) (Session, error) {
	var sess Session
	var createdAt, expiresAt, lastSeenAt int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, created_at, expires_at, last_seen_at FROM sessions WHERE id = ?`,
		tokenHashHex,
	).Scan(&sess.ID, &sess.UserID, &createdAt, &expiresAt, &lastSeenAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("loading session: %w", err)
	}
	sess.CreatedAt = time.Unix(createdAt, 0).UTC()
	sess.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	sess.LastSeenAt = time.Unix(lastSeenAt, 0).UTC()

	if time.Now().UTC().After(sess.ExpiresAt) {
		// Best-effort cleanup; ignore error since the lookup itself succeeded.
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, tokenHashHex)
		return Session{}, ErrNotFound
	}
	return sess, nil
}

// TouchSession bumps last_seen_at to now. Errors are returned but rarely fatal.
func (s *Store) TouchSession(ctx context.Context, tokenHashHex string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ? WHERE id = ?`,
		time.Now().Unix(), tokenHashHex,
	)
	if err != nil {
		return fmt.Errorf("touching session: %w", err)
	}
	return nil
}

// DeleteSession removes a session row. Idempotent.
func (s *Store) DeleteSession(ctx context.Context, tokenHashHex string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, tokenHashHex)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}
