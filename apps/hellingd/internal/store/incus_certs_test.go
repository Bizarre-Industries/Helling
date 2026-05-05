package store

import (
	"testing"
	"time"
)

func TestUpsertAndRevokeIncusUserCert(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	u, err := st.CreateUser(t.Context(), "alice", "hash", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	if err := st.UpsertIncusUserCert(t.Context(), u.ID, "CERT", "KEY", "fp1", "alice", expiresAt); err != nil {
		t.Fatalf("UpsertIncusUserCert: %v", err)
	}
	row, err := st.GetIncusUserCert(t.Context(), u.ID)
	if err != nil {
		t.Fatalf("GetIncusUserCert: %v", err)
	}
	if row.Fingerprint != "fp1" || row.ProjectScope != "alice" || row.RevokedAt != nil {
		t.Fatalf("cert row: %#v", row)
	}
	if row.CertPEM != "CERT" {
		t.Fatalf("cert: got %q want CERT", row.CertPEM)
	}
	var encryptedCert string
	if err := st.db.QueryRowContext(t.Context(), `SELECT encrypted_cert_pem FROM incus_user_certs WHERE user_id = ?`, u.ID).Scan(&encryptedCert); err != nil {
		t.Fatalf("select encrypted cert: %v", err)
	}
	if encryptedCert == "CERT" {
		t.Fatal("certificate was stored in plaintext")
	}
	key, err := st.DecryptSecret(row.EncryptedKeyPEM)
	if err != nil {
		t.Fatalf("DecryptSecret: %v", err)
	}
	if key != "KEY" {
		t.Fatalf("key: got %q want KEY", key)
	}
	if err := st.RevokeIncusUserCert(t.Context(), u.ID); err != nil {
		t.Fatalf("RevokeIncusUserCert: %v", err)
	}
	row, err = st.GetIncusUserCert(t.Context(), u.ID)
	if err != nil {
		t.Fatalf("GetIncusUserCert after revoke: %v", err)
	}
	if row.RevokedAt == nil {
		t.Fatal("RevokedAt is nil after revoke")
	}
}
