package incus

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

func TestIssueUserCertificateCreatesClientAuthCertificate(t *testing.T) {
	t.Parallel()
	issuedAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	ca, err := NewInternalCertificateAuthority(issuedAt)
	if err != nil {
		t.Fatalf("NewInternalCertificateAuthority: %v", err)
	}
	certPEM, keyPEM, fingerprint, expiresAt, err := IssueUserCertificateWithCA(42, "alice", issuedAt, ca)
	if err != nil {
		t.Fatalf("IssueUserCertificateWithCA: %v", err)
	}
	if len(fingerprint) != 64 {
		t.Fatalf("fingerprint length: got %d want 64", len(fingerprint))
	}
	if !expiresAt.Equal(issuedAt.Add(90 * 24 * time.Hour)) {
		t.Fatalf("expiresAt: got %s", expiresAt)
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		t.Fatal("certificate PEM did not decode")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if cert.Subject.CommonName != "helling-user-42-alice" {
		t.Fatalf("CommonName: got %q", cert.Subject.CommonName)
	}
	if cert.Issuer.String() != ca.Certificate.Subject.String() {
		t.Fatalf("issuer: got %q want %q", cert.Issuer.String(), ca.Certificate.Subject.String())
	}
	if err := cert.CheckSignatureFrom(ca.Certificate); err != nil {
		t.Fatalf("user certificate is not signed by internal CA: %v", err)
	}
	if len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth {
		t.Fatalf("ExtKeyUsage: got %#v want client auth only", cert.ExtKeyUsage)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "PRIVATE KEY" {
		t.Fatalf("private key PEM did not decode as PRIVATE KEY")
	}
}

func TestNewInternalCertificateAuthorityCreatesCA(t *testing.T) {
	t.Parallel()
	generatedAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	ca, err := NewInternalCertificateAuthority(generatedAt)
	if err != nil {
		t.Fatalf("NewInternalCertificateAuthority: %v", err)
	}
	if ca.Certificate.Subject.CommonName != "Helling CA" {
		t.Fatalf("CA CommonName: got %q", ca.Certificate.Subject.CommonName)
	}
	if !ca.Certificate.IsCA {
		t.Fatal("internal CA certificate is not a CA")
	}
	if ca.Certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatalf("CA KeyUsage missing CertSign: %#v", ca.Certificate.KeyUsage)
	}
	if err := ca.Certificate.CheckSignatureFrom(ca.Certificate); err != nil {
		t.Fatalf("CA certificate is not self-signed: %v", err)
	}
}

func TestLoadOrCreateCertificateAuthorityPersistsEncryptedKey(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "ca.key.age")
	certPath := filepath.Join(dir, "ca.crt")
	generatedAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	first, err := LoadOrCreateCertificateAuthority(keyPath, certPath, st.EncryptSecret, st.DecryptSecret, generatedAt)
	if err != nil {
		t.Fatalf("LoadOrCreateCertificateAuthority first: %v", err)
	}
	encryptedKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile key: %v", err)
	}
	if bytes.Contains(encryptedKey, []byte("PRIVATE KEY")) {
		t.Fatal("stored CA key contains plaintext private key PEM")
	}
	second, err := LoadOrCreateCertificateAuthority(keyPath, certPath, st.EncryptSecret, st.DecryptSecret, generatedAt.Add(time.Hour))
	if err != nil {
		t.Fatalf("LoadOrCreateCertificateAuthority second: %v", err)
	}
	if !bytes.Equal(first.Certificate.Raw, second.Certificate.Raw) {
		t.Fatal("LoadOrCreateCertificateAuthority created a replacement CA instead of loading the stored CA")
	}
}
