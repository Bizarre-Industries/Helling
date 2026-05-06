package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

const (
	ageIdentityFilename    = "age-identity.txt"
	ageEncryptionPrefix    = "age-encryption.org/v1"
	defaultAgeIdentityPath = "/etc/helling/age/identity.txt"
)

func loadOrCreateAgeIdentity(stateDir string) (*age.X25519Identity, error) {
	path := ageIdentityPath(stateDir)
	// #nosec G304 -- the filename is fixed and scoped under the configured Helling state directory.
	if body, err := os.ReadFile(path); err == nil {
		identity, parseErr := age.ParseX25519Identity(strings.TrimSpace(string(body)))
		if parseErr != nil {
			return nil, fmt.Errorf("parsing age identity %s: %w", path, parseErr)
		}
		return identity, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading age identity %s: %w", path, err)
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generating age identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating age identity directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(identity.String()+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("writing age identity %s: %w", path, err)
	}
	return identity, nil
}

func ageIdentityPath(stateDir string) string {
	if path := os.Getenv("HELLING_AGE_IDENTITY_PATH"); path != "" {
		return path
	}
	if stateDir == "/var/lib/helling" {
		return defaultAgeIdentityPath
	}
	return filepath.Join(stateDir, ageIdentityFilename)
}

// EncryptSecret encrypts a control-plane secret using the host age identity.
func (s *Store) EncryptSecret(plaintext string) (string, error) {
	if s == nil || s.ageRecipient == nil {
		return "", errors.New("store: age recipient unavailable")
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, s.ageRecipient)
	if err != nil {
		return "", fmt.Errorf("creating age writer: %w", err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("writing age payload: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("closing age writer: %w", err)
	}
	return buf.String(), nil
}

// DecryptSecret decrypts a control-plane secret encrypted by EncryptSecret.
func (s *Store) DecryptSecret(ciphertext string) (string, error) {
	if s == nil || s.ageIdentity == nil {
		return "", errors.New("store: age identity unavailable")
	}
	r, err := age.Decrypt(strings.NewReader(ciphertext), s.ageIdentity)
	if err != nil {
		return "", fmt.Errorf("creating age reader: %w", err)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("reading age payload: %w", err)
	}
	return string(body), nil
}

func (s *Store) encryptLegacyWebhookSecrets(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, secret_encrypted FROM webhooks`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") || strings.Contains(err.Error(), "no such column") {
			return nil
		}
		return fmt.Errorf("listing webhook secrets for encryption migration: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type legacySecret struct {
		id     string
		secret string
	}
	var legacy []legacySecret
	for rows.Next() {
		var row legacySecret
		if err := rows.Scan(&row.id, &row.secret); err != nil {
			return fmt.Errorf("scanning webhook secret for encryption migration: %w", err)
		}
		if strings.HasPrefix(row.secret, ageEncryptionPrefix) {
			if _, err := s.DecryptSecret(row.secret); err != nil {
				return fmt.Errorf("validating encrypted webhook secret %s: %w", row.id, err)
			}
			continue
		}
		legacy = append(legacy, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading webhook secrets for encryption migration: %w", err)
	}
	for _, row := range legacy {
		encrypted, err := s.EncryptSecret(row.secret)
		if err != nil {
			return fmt.Errorf("encrypting legacy webhook secret %s: %w", row.id, err)
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE webhooks SET secret_encrypted = ? WHERE id = ?`, encrypted, row.id); err != nil {
			return fmt.Errorf("storing encrypted legacy webhook secret %s: %w", row.id, err)
		}
	}
	return nil
}

func (s *Store) encryptLegacyIncusUserCerts(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, encrypted_cert_pem FROM incus_user_certs`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") || strings.Contains(err.Error(), "no such column") {
			return nil
		}
		return fmt.Errorf("listing Incus user certificates for encryption migration: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type legacyCert struct {
		userID  int64
		certPEM string
	}
	var legacy []legacyCert
	for rows.Next() {
		var row legacyCert
		if err := rows.Scan(&row.userID, &row.certPEM); err != nil {
			return fmt.Errorf("scanning Incus user certificate for encryption migration: %w", err)
		}
		if row.certPEM == "" {
			continue
		}
		if strings.HasPrefix(row.certPEM, ageEncryptionPrefix) {
			if _, err := s.DecryptSecret(row.certPEM); err != nil {
				return fmt.Errorf("validating encrypted Incus user certificate %d: %w", row.userID, err)
			}
			continue
		}
		legacy = append(legacy, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading Incus user certificates for encryption migration: %w", err)
	}
	for _, row := range legacy {
		encrypted, err := s.EncryptSecret(row.certPEM)
		if err != nil {
			return fmt.Errorf("encrypting legacy Incus user certificate %d: %w", row.userID, err)
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE incus_user_certs SET encrypted_cert_pem = ? WHERE user_id = ?`, encrypted, row.userID); err != nil {
			return fmt.Errorf("storing encrypted legacy Incus user certificate %d: %w", row.userID, err)
		}
	}
	return nil
}
