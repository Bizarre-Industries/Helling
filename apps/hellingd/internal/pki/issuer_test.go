package pki_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/pki"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

func TestIssuer_IssueForUser_PersistsEncryptedCert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "issuer.db") + "?cache=shared"
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	repo := authrepo.New(pool)
	u, err := repo.CreateUser(ctx, "alice", "admin", "")
	if err != nil {
		t.Fatal(err)
	}
	ca, err := pki.Bootstrap(nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := pki.NewIdentity()
	iss := &pki.Issuer{CA: ca, Identity: id, Repo: repo}
	if err := iss.IssueForUser(ctx, u.ID, u.Username); err != nil {
		t.Fatalf("issue: %v", err)
	}
	got, err := repo.GetActiveUserCertificate(ctx, u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.UserID != u.ID || got.SerialNumber == "" {
		t.Fatalf("missing fields: %+v", got)
	}
	other, _ := pki.NewIdentity()
	if _, err := pki.DecryptWithIdentity(other, got.CertPEM); err == nil {
		t.Fatal("decrypt with wrong identity should fail")
	}
	pem, err := pki.DecryptWithIdentity(id, got.CertPEM)
	if err != nil {
		t.Fatalf("decrypt cert: %v", err)
	}
	if len(pem) == 0 {
		t.Fatal("empty cert pem after decrypt")
	}
}

func TestIssuer_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	var iss *pki.Issuer
	if err := iss.IssueForUser(context.Background(), "u", "n"); err == nil {
		t.Fatal("expected error for nil issuer")
	}
	if err := (&pki.Issuer{}).IssueForUser(context.Background(), "u", "n"); err == nil {
		t.Fatal("expected error for unconfigured issuer")
	}
}
