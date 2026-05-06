package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// IncusUserCert mirrors the active certificate row for a Helling user.
type IncusUserCert struct {
	UserID           int64
	CertPEM          string
	EncryptedCertPEM string
	EncryptedKeyPEM  string
	Fingerprint      string
	Restricted       bool
	ProjectScope     string
	ExpiresAt        *time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// UpsertIncusUserCert stores or replaces the active Incus certificate for a user.
func (s *Store) UpsertIncusUserCert(ctx context.Context, userID int64, certPEM, keyPEM, fingerprint, project string, expiresAt time.Time) error {
	encryptedCert, encryptedKey, err := s.encryptIncusUserCert(certPEM, keyPEM)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.db.ExecContext(ctx, incusUserCertUpsertSQL,
		userID, encryptedCert, encryptedKey, fingerprint, project, expiresAt.Unix(), now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting Incus user certificate: %w", err)
	}
	return nil
}

// UpsertIncusUserCertAndProject stores the active Incus certificate and assigned project atomically.
func (s *Store) UpsertIncusUserCertAndProject(ctx context.Context, userID int64, certPEM, keyPEM, fingerprint, project string, expiresAt time.Time) error {
	encryptedCert, encryptedKey, err := s.encryptIncusUserCert(certPEM, keyPEM)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning Incus user certificate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET incus_project = ? WHERE id = ?`, project, userID); err != nil {
		return fmt.Errorf("updating user Incus project: %w", err)
	}
	result, err := tx.ExecContext(ctx, incusUserCertUpsertSQL,
		userID, encryptedCert, encryptedKey, fingerprint, project, expiresAt.Unix(), now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting Incus user certificate: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("checking Incus user certificate upsert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing Incus user certificate transaction: %w", err)
	}
	return nil
}

// RestoreIncusUserCertAndProject restores a previously loaded certificate row
// without decrypting and re-encrypting secret material.
func (s *Store) RestoreIncusUserCertAndProject(ctx context.Context, row *IncusUserCert) error {
	if row == nil || row.UserID == 0 || row.EncryptedCertPEM == "" || row.EncryptedKeyPEM == "" || row.Fingerprint == "" || row.ProjectScope == "" {
		return errors.New("invalid Incus user certificate restore row")
	}
	now := time.Now().UTC().Unix()
	var expiresAt any
	if row.ExpiresAt != nil {
		expiresAt = row.ExpiresAt.Unix()
	}
	var revokedAt any
	if row.RevokedAt != nil {
		revokedAt = row.RevokedAt.Unix()
	}
	createdAt := row.CreatedAt.Unix()
	if createdAt <= 0 {
		createdAt = now
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning Incus user certificate restore transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET incus_project = ? WHERE id = ?`, row.ProjectScope, row.UserID); err != nil {
		return fmt.Errorf("restoring user Incus project: %w", err)
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO incus_user_certs
		 (user_id, encrypted_cert_pem, encrypted_key_pem, fingerprint, restricted, project_scope, expires_at, revoked_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   encrypted_cert_pem = excluded.encrypted_cert_pem,
		   encrypted_key_pem = excluded.encrypted_key_pem,
		   fingerprint = excluded.fingerprint,
		   restricted = excluded.restricted,
		   project_scope = excluded.project_scope,
		   expires_at = excluded.expires_at,
		   revoked_at = excluded.revoked_at,
		   updated_at = excluded.updated_at`,
		row.UserID, row.EncryptedCertPEM, row.EncryptedKeyPEM, row.Fingerprint, boolToInt(row.Restricted), row.ProjectScope, expiresAt, revokedAt, createdAt, now,
	)
	if err != nil {
		return fmt.Errorf("restoring Incus user certificate: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing Incus user certificate restore: %w", err)
	}
	return nil
}

func (s *Store) encryptIncusUserCert(certPEM, keyPEM string) (encryptedCert, encryptedKey string, err error) {
	encryptedCert, err = s.EncryptSecret(certPEM)
	if err != nil {
		return "", "", fmt.Errorf("encrypting Incus user certificate: %w", err)
	}
	encryptedKey, err = s.EncryptSecret(keyPEM)
	if err != nil {
		return "", "", fmt.Errorf("encrypting Incus user certificate key: %w", err)
	}
	return encryptedCert, encryptedKey, nil
}

const incusUserCertUpsertSQL = `INSERT INTO incus_user_certs
		 (user_id, encrypted_cert_pem, encrypted_key_pem, fingerprint, restricted, project_scope, expires_at, revoked_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?, NULL, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   encrypted_cert_pem = excluded.encrypted_cert_pem,
		   encrypted_key_pem = excluded.encrypted_key_pem,
		   fingerprint = excluded.fingerprint,
		   restricted = 1,
		   project_scope = excluded.project_scope,
		   expires_at = excluded.expires_at,
		   revoked_at = NULL,
		   updated_at = excluded.updated_at`

// GetIncusUserCert returns the certificate row for a user.
func (s *Store) GetIncusUserCert(ctx context.Context, userID int64) (IncusUserCert, error) {
	var row IncusUserCert
	var restricted int
	var expiresAt, revokedAt sql.NullInt64
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, encrypted_cert_pem, encrypted_key_pem, fingerprint, restricted, project_scope, expires_at, revoked_at, created_at, updated_at
		 FROM incus_user_certs WHERE user_id = ?`,
		userID,
	).Scan(&row.UserID, &row.EncryptedCertPEM, &row.EncryptedKeyPEM, &row.Fingerprint, &restricted, &row.ProjectScope, &expiresAt, &revokedAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return IncusUserCert{}, ErrNotFound
		}
		return IncusUserCert{}, fmt.Errorf("loading Incus user certificate: %w", err)
	}
	certPEM, err := s.DecryptSecret(row.EncryptedCertPEM)
	if err != nil {
		return IncusUserCert{}, fmt.Errorf("decrypting Incus user certificate: %w", err)
	}
	row.CertPEM = certPEM
	row.Restricted = restricted != 0
	if expiresAt.Valid {
		ts := time.Unix(expiresAt.Int64, 0).UTC()
		row.ExpiresAt = &ts
	}
	if revokedAt.Valid {
		ts := time.Unix(revokedAt.Int64, 0).UTC()
		row.RevokedAt = &ts
	}
	row.CreatedAt = time.Unix(createdAt, 0).UTC()
	row.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return row, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// RevokeIncusUserCert marks a user's current Incus certificate as revoked.
func (s *Store) RevokeIncusUserCert(ctx context.Context, userID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.db.ExecContext(ctx,
		`UPDATE incus_user_certs SET revoked_at = ?, updated_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		now, now, userID,
	)
	if err != nil {
		return fmt.Errorf("revoking Incus user certificate: %w", err)
	}
	return nil
}
