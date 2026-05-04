package store

// TOTP secrets and recovery codes CRUD.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
)

// TOTPSecret mirrors a row in the totp_secrets table.
type TOTPSecret struct {
	UserID    int64
	Secret    string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetTOTPSecret returns the TOTP secret for a user, or ErrNotFound.
func (s *Store) GetTOTPSecret(ctx context.Context, userID int64) (TOTPSecret, error) {
	var ts TOTPSecret
	var createdAt, updatedAt int64
	var enabled int
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, secret, enabled, created_at, updated_at FROM totp_secrets WHERE user_id = ?`,
		userID,
	).Scan(&ts.UserID, &ts.Secret, &enabled, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return TOTPSecret{}, ErrNotFound
	}
	if err != nil {
		return TOTPSecret{}, fmt.Errorf("loading totp secret: %w", err)
	}
	ts.Enabled = enabled != 0
	ts.CreatedAt = time.Unix(createdAt, 0).UTC()
	ts.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return ts, nil
}

// SetTOTPSecret upserts a TOTP secret for a user.
func (s *Store) SetTOTPSecret(ctx context.Context, userID int64, secret string, enabled bool) error {
	now := time.Now().UTC().Unix()
	en := 0
	if enabled {
		en = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO totp_secrets (user_id, secret, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET secret = excluded.secret, enabled = excluded.enabled, updated_at = excluded.updated_at`,
		userID, secret, en, now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting totp secret: %w", err)
	}
	return nil
}

// DeleteTOTPSecret removes the TOTP secret for a user. Idempotent.
func (s *Store) DeleteTOTPSecret(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM totp_secrets WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("deleting totp secret: %w", err)
	}
	return nil
}

// SaveRecoveryCodes inserts Argon2id-hashed recovery codes for a user.
// Caller should delete existing codes first.
func (s *Store) SaveRecoveryCodes(ctx context.Context, userID int64, codeHashes []string) error {
	for _, h := range codeHashes {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO totp_recovery_codes (user_id, code_hash) VALUES (?, ?)`,
			userID, h,
		)
		if err != nil {
			return fmt.Errorf("inserting recovery code: %w", err)
		}
	}
	return nil
}

// ConsumeRecoveryCode verifies a raw recovery code against unused code hashes,
// marks the matching row as used, and returns true when a code was consumed.
func (s *Store) ConsumeRecoveryCode(ctx context.Context, userID int64, rawCode string) (bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code_hash FROM totp_recovery_codes WHERE user_id = ? AND used = 0`,
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("listing recovery codes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type recoveryCodeRow struct {
		id       int64
		codeHash string
	}
	var candidates []recoveryCodeRow
	for rows.Next() {
		var candidate recoveryCodeRow
		if err := rows.Scan(&candidate.id, &candidate.codeHash); err != nil {
			return false, fmt.Errorf("scanning recovery code: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterating recovery codes: %w", err)
	}
	_ = rows.Close()

	for _, candidate := range candidates {
		if !auth.VerifyRecoveryCode(rawCode, candidate.codeHash) {
			continue
		}
		res, err := s.db.ExecContext(ctx,
			`UPDATE totp_recovery_codes SET used = 1 WHERE id = ? AND user_id = ? AND used = 0`,
			candidate.id, userID,
		)
		if err != nil {
			return false, fmt.Errorf("consuming recovery code: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("checking recovery code rows: %w", err)
		}
		return n > 0, nil
	}
	return false, nil
}

// DeleteRecoveryCodes removes all recovery codes for a user.
func (s *Store) DeleteRecoveryCodes(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM totp_recovery_codes WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("deleting recovery codes: %w", err)
	}
	return nil
}

// CountUnusedRecoveryCodes returns the number of unused recovery codes.
func (s *Store) CountUnusedRecoveryCodes(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM totp_recovery_codes WHERE user_id = ? AND used = 0`,
		userID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting recovery codes: %w", err)
	}
	return n, nil
}
