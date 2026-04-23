package authrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ── TOTP ───────────────────────────────────────────────────────────────────

// TOTPSecret mirrors the totp_secrets row.
type TOTPSecret struct {
	UserID          string
	EncryptedSecret []byte
	Enabled         bool
	CreatedAt       int64
	UpdatedAt       int64
}

// UpsertTOTPSecret inserts or replaces the per-user TOTP row. `enabled` is
// false during enrollment and flipped true by SetTOTPEnabled after verify.
func (r *Repo) UpsertTOTPSecret(ctx context.Context, userID string, encryptedSecret []byte, enabled bool) error {
	now := r.now().Unix()
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	const q = `INSERT INTO totp_secrets (user_id, encrypted_secret, enabled, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?)
	           ON CONFLICT(user_id) DO UPDATE SET
	             encrypted_secret = excluded.encrypted_secret,
	             enabled          = excluded.enabled,
	             updated_at       = excluded.updated_at`
	if _, err := r.db.ExecContext(ctx, q, userID, encryptedSecret, enabledInt, now, now); err != nil {
		return fmt.Errorf("authrepo: upsert totp secret: %w", err)
	}
	return nil
}

// GetTOTPSecret returns the TOTP row or ErrNotFound.
func (r *Repo) GetTOTPSecret(ctx context.Context, userID string) (TOTPSecret, error) {
	const q = `SELECT user_id, encrypted_secret, enabled, created_at, updated_at
	           FROM totp_secrets WHERE user_id = ?`
	row := r.db.QueryRowContext(ctx, q, userID)
	var s TOTPSecret
	var enabled int
	err := row.Scan(&s.UserID, &s.EncryptedSecret, &enabled, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return TOTPSecret{}, ErrNotFound
	}
	if err != nil {
		return TOTPSecret{}, fmt.Errorf("authrepo: get totp: %w", err)
	}
	s.Enabled = enabled == 1
	return s, nil
}

// SetTOTPEnabled flips the enabled flag on an existing TOTP row.
func (r *Repo) SetTOTPEnabled(ctx context.Context, userID string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	const q = `UPDATE totp_secrets SET enabled = ?, updated_at = ? WHERE user_id = ?`
	if _, err := r.db.ExecContext(ctx, q, enabledInt, r.now().Unix(), userID); err != nil {
		return fmt.Errorf("authrepo: set totp enabled: %w", err)
	}
	return nil
}

// DeleteTOTPSecret removes the TOTP row AND all recovery codes for the user.
func (r *Repo) DeleteTOTPSecret(ctx context.Context, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("authrepo: begin delete totp: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM totp_secrets WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("authrepo: delete totp: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("authrepo: delete recovery codes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("authrepo: commit delete totp: %w", err)
	}
	return nil
}

// ── Recovery codes ─────────────────────────────────────────────────────────

// InsertRecoveryCodes writes one row per hash for the given user.
func (r *Repo) InsertRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("authrepo: begin recovery insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	const q = `INSERT INTO recovery_codes (id, user_id, code_hash, created_at) VALUES (?, ?, ?, ?)`
	now := r.now().Unix()
	for _, h := range hashes {
		if _, err := tx.ExecContext(ctx, q, uuid.NewString(), userID, h, now); err != nil {
			return fmt.Errorf("authrepo: insert recovery code: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("authrepo: commit recovery codes: %w", err)
	}
	return nil
}

// RecoveryCode mirrors a recovery_codes row.
type RecoveryCode struct {
	ID        string
	UserID    string
	CodeHash  string
	UsedAt    sql.NullInt64
	CreatedAt int64
}

// ListUnusedRecoveryCodes returns un-consumed codes for a user.
func (r *Repo) ListUnusedRecoveryCodes(ctx context.Context, userID string) ([]RecoveryCode, error) {
	const q = `SELECT id, user_id, code_hash, used_at, created_at
	           FROM recovery_codes WHERE user_id = ? AND used_at IS NULL`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("authrepo: list recovery codes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []RecoveryCode
	for rows.Next() {
		var rc RecoveryCode
		if err := rows.Scan(&rc.ID, &rc.UserID, &rc.CodeHash, &rc.UsedAt, &rc.CreatedAt); err != nil {
			return nil, fmt.Errorf("authrepo: scan recovery code: %w", err)
		}
		out = append(out, rc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("authrepo: rows recovery codes: %w", err)
	}
	return out, nil
}

// MarkRecoveryCodeUsed stamps used_at on a recovery code id.
func (r *Repo) MarkRecoveryCodeUsed(ctx context.Context, id string) error {
	const q = `UPDATE recovery_codes SET used_at = ? WHERE id = ? AND used_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, r.now().Unix(), id); err != nil {
		return fmt.Errorf("authrepo: mark recovery code used: %w", err)
	}
	return nil
}

// ── API tokens ─────────────────────────────────────────────────────────────

// APIToken mirrors the api_tokens row.
type APIToken struct {
	ID         string
	UserID     string
	Name       string
	TokenHash  string
	Scope      string
	LastUsedAt sql.NullInt64
	ExpiresAt  int64
	RevokedAt  sql.NullInt64
	CreatedAt  int64
}

// CreateAPIToken inserts an api_tokens row.
func (r *Repo) CreateAPIToken(ctx context.Context, userID, name, tokenHash, scope string, expiresAt time.Time) (APIToken, error) {
	id := uuid.NewString()
	now := r.now().Unix()
	const q = `INSERT INTO api_tokens (id, user_id, name, token_hash, scope, expires_at, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, q, id, userID, name, tokenHash, scope, expiresAt.Unix(), now); err != nil {
		return APIToken{}, fmt.Errorf("authrepo: insert api token: %w", err)
	}
	return APIToken{
		ID: id, UserID: userID, Name: name, TokenHash: tokenHash, Scope: scope,
		ExpiresAt: expiresAt.Unix(), CreatedAt: now,
	}, nil
}

// ListAPITokens returns a user's tokens ordered by created_at ASC.
func (r *Repo) ListAPITokens(ctx context.Context, userID string) ([]APIToken, error) {
	const q = `SELECT id, user_id, name, token_hash, scope, last_used_at, expires_at, revoked_at, created_at
	           FROM api_tokens WHERE user_id = ? ORDER BY created_at ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("authrepo: list api tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []APIToken
	for rows.Next() {
		var tok APIToken
		if err := rows.Scan(&tok.ID, &tok.UserID, &tok.Name, &tok.TokenHash, &tok.Scope,
			&tok.LastUsedAt, &tok.ExpiresAt, &tok.RevokedAt, &tok.CreatedAt); err != nil {
			return nil, fmt.Errorf("authrepo: scan api token: %w", err)
		}
		out = append(out, tok)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("authrepo: rows api tokens: %w", err)
	}
	return out, nil
}

// RevokeAPIToken marks a token revoked by id (owner-scoped). Returns
// ErrNotFound when no active matching row exists.
func (r *Repo) RevokeAPIToken(ctx context.Context, userID, tokenID string) error {
	const q = `UPDATE api_tokens SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`
	res, err := r.db.ExecContext(ctx, q, r.now().Unix(), tokenID, userID)
	if err != nil {
		return fmt.Errorf("authrepo: revoke api token: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetAPITokenByHash returns the active token matching the SHA-256 digest or
// ErrNotFound. Used by Bearer-auth middleware.
func (r *Repo) GetAPITokenByHash(ctx context.Context, tokenHash string) (APIToken, error) {
	now := r.now().Unix()
	const q = `SELECT id, user_id, name, token_hash, scope, last_used_at, expires_at, revoked_at, created_at
	           FROM api_tokens WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ?`
	row := r.db.QueryRowContext(ctx, q, tokenHash, now)
	var tok APIToken
	err := row.Scan(&tok.ID, &tok.UserID, &tok.Name, &tok.TokenHash, &tok.Scope,
		&tok.LastUsedAt, &tok.ExpiresAt, &tok.RevokedAt, &tok.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return APIToken{}, ErrNotFound
	}
	if err != nil {
		return APIToken{}, fmt.Errorf("authrepo: get api token: %w", err)
	}
	return tok, nil
}

// TouchAPITokenLastUsed updates the last_used_at stamp for audit.
func (r *Repo) TouchAPITokenLastUsed(ctx context.Context, id string) error {
	const q = `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, r.now().Unix(), id); err != nil {
		return fmt.Errorf("authrepo: touch api token: %w", err)
	}
	return nil
}
