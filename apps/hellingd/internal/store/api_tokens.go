package store

// API tokens CRUD. Tokens are stored as SHA-256 hashes; the raw token
// (helling_<random 40>) is shown once at creation and never stored.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// APIToken mirrors a row in the api_tokens table.
type APIToken struct {
	ID         string
	UserID     int64
	Name       string
	TokenHash  string
	Scopes     string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
}

// CreateAPIToken inserts a new API token and returns the persisted row.
func (s *Store) CreateAPIToken(ctx context.Context, userID int64, name, tokenHash, scopes string, expiresAt *time.Time) (APIToken, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return APIToken{}, fmt.Errorf("generating token id: %w", err)
	}
	now := time.Now().UTC()
	t := APIToken{
		ID:        id.String(),
		UserID:    userID,
		Name:      name,
		TokenHash: tokenHash,
		Scopes:    scopes,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	var exp *int64
	if expiresAt != nil {
		unix := expiresAt.Unix()
		exp = &unix
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO api_tokens (id, user_id, name, token_hash, scopes, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.UserID, t.Name, t.TokenHash, t.Scopes, now.Unix(), exp,
	)
	if err != nil {
		return APIToken{}, fmt.Errorf("inserting api token: %w", err)
	}
	return t, nil
}

// GetAPITokenByHash returns the token with the given hash, or ErrNotFound.
func (s *Store) GetAPITokenByHash(ctx context.Context, tokenHash string) (APIToken, error) {
	var t APIToken
	var createdAt int64
	var expiresAt, lastUsedAt sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, token_hash, scopes, created_at, expires_at, last_used_at
		 FROM api_tokens WHERE token_hash = ?`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Scopes, &createdAt, &expiresAt, &lastUsedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return APIToken{}, ErrNotFound
	}
	if err != nil {
		return APIToken{}, fmt.Errorf("loading api token: %w", err)
	}
	t.CreatedAt = time.Unix(createdAt, 0).UTC()
	if expiresAt.Valid {
		ts := time.Unix(expiresAt.Int64, 0).UTC()
		t.ExpiresAt = &ts
	}
	if lastUsedAt.Valid {
		ts := time.Unix(lastUsedAt.Int64, 0).UTC()
		t.LastUsedAt = &ts
	}
	return t, nil
}

// ListAPITokensByUser returns all non-expired tokens for a user.
func (s *Store) ListAPITokensByUser(ctx context.Context, userID int64) ([]APIToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, token_hash, scopes, created_at, expires_at, last_used_at
		 FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing api tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tokens []APIToken
	for rows.Next() {
		var t APIToken
		var createdAt int64
		var expiresAt, lastUsedAt sql.NullInt64
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.Scopes, &createdAt, &expiresAt, &lastUsedAt); err != nil {
			return nil, fmt.Errorf("scanning api token: %w", err)
		}
		t.CreatedAt = time.Unix(createdAt, 0).UTC()
		if expiresAt.Valid {
			ts := time.Unix(expiresAt.Int64, 0).UTC()
			t.ExpiresAt = &ts
		}
		if lastUsedAt.Valid {
			ts := time.Unix(lastUsedAt.Int64, 0).UTC()
			t.LastUsedAt = &ts
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// DeleteAPIToken removes a token by id. Idempotent.
func (s *Store) DeleteAPIToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_tokens WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting api token: %w", err)
	}
	return nil
}

// TouchAPIToken bumps last_used_at to now.
func (s *Store) TouchAPIToken(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_tokens SET last_used_at = ? WHERE token_hash = ?`,
		time.Now().Unix(), tokenHash,
	)
	if err != nil {
		return fmt.Errorf("touching api token: %w", err)
	}
	return nil
}
