package incus

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

func TestUserTrustServiceProvisionWritesCertRunsHelperAndStoresRow(t *testing.T) {
	t.Parallel()
	st := newIncusTrustTestStore(t)
	u, err := st.CreateUser(t.Context(), "alice", "hash", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	u.IncusProject = "alice"
	ca, err := NewInternalCertificateAuthority(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewInternalCertificateAuthority: %v", err)
	}
	var calls [][]string
	requestDir := t.TempDir()
	service := UserTrustService{
		Store:      st,
		CA:         ca,
		StagingDir: t.TempDir(),
		RequestDir: requestDir,
		HelperPath: "/usr/lib/helling/helling-incus-trust",
		RunHelper: func(_ context.Context, path string, args ...string) error {
			if path != "/usr/lib/helling/helling-incus-trust" {
				t.Fatalf("helper path: got %q", path)
			}
			calls = append(calls, append([]string(nil), args...))
			return nil
		},
	}

	if err := service.Provision(t.Context(), &u); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	row, err := st.GetIncusUserCert(t.Context(), u.ID)
	if err != nil {
		t.Fatalf("GetIncusUserCert: %v", err)
	}
	if row.Fingerprint == "" || row.ProjectScope != "alice" || row.RevokedAt != nil {
		t.Fatalf("cert row: %#v", row)
	}
	block, _ := pem.Decode([]byte(row.CertPEM))
	if block == nil {
		t.Fatal("stored certificate PEM did not decode")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if err := cert.CheckSignatureFrom(ca.Certificate); err != nil {
		t.Fatalf("stored user cert is not signed by service CA: %v", err)
	}
	if len(calls) != 1 || len(calls[0]) != 5 || calls[0][0] != "register" ||
		calls[0][1] != row.Fingerprint || calls[0][2] != "alice" ||
		!trustRequestNamePattern.MatchString(calls[0][4]) {
		t.Fatalf("helper calls: %#v", calls)
	}
}

func TestUserTrustServiceRevokeRunsHelperAndMarksRevoked(t *testing.T) {
	t.Parallel()
	st := newIncusTrustTestStore(t)
	u, err := st.CreateUser(t.Context(), "alice", "hash", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := st.UpsertIncusUserCert(t.Context(), u.ID, "CERT", "KEY", testFingerprint, "alice", time.Now().UTC().Add(24*time.Hour)); err != nil {
		t.Fatalf("UpsertIncusUserCert: %v", err)
	}
	var calls [][]string
	requestDir := t.TempDir()
	service := UserTrustService{
		Store:      st,
		RequestDir: requestDir,
		RunHelper: func(_ context.Context, _ string, args ...string) error {
			calls = append(calls, append([]string(nil), args...))
			return nil
		},
	}

	if err := service.Revoke(t.Context(), &u); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if len(calls) != 1 || len(calls[0]) != 3 || calls[0][0] != "revoke" ||
		calls[0][1] != testFingerprint || !trustRequestNamePattern.MatchString(calls[0][2]) {
		t.Fatalf("helper calls: %#v", calls)
	}
	row, err := st.GetIncusUserCert(t.Context(), u.ID)
	if err != nil {
		t.Fatalf("GetIncusUserCert: %v", err)
	}
	if row.RevokedAt == nil {
		t.Fatal("cert row not marked revoked")
	}
}

func TestUserTrustServiceProvisionRegistersReplacementBeforeRevokingOldTrust(t *testing.T) {
	t.Parallel()
	st := newIncusTrustTestStore(t)
	u, err := st.CreateUser(t.Context(), "alice", "hash", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	u.IncusProject = "bob"
	if err := st.UpsertIncusUserCert(t.Context(), u.ID, "OLD CERT", "OLD KEY", testFingerprint, "alice", time.Now().UTC().Add(24*time.Hour)); err != nil {
		t.Fatalf("UpsertIncusUserCert: %v", err)
	}
	ca, err := NewInternalCertificateAuthority(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewInternalCertificateAuthority: %v", err)
	}
	var calls [][]string
	service := UserTrustService{
		Store:      st,
		CA:         ca,
		StagingDir: t.TempDir(),
		RequestDir: t.TempDir(),
		RunHelper: func(_ context.Context, _ string, args ...string) error {
			calls = append(calls, append([]string(nil), args...))
			return nil
		},
	}

	if err := service.Provision(t.Context(), &u); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("helper calls: got %#v want register then revoke", calls)
	}
	if calls[0][0] != "register" || calls[0][2] != "bob" {
		t.Fatalf("first helper call = %#v, want register bob", calls[0])
	}
	if calls[1][0] != "revoke" || calls[1][1] != testFingerprint {
		t.Fatalf("second helper call = %#v, want revoke old fingerprint", calls[1])
	}
	row, err := st.GetIncusUserCert(t.Context(), u.ID)
	if err != nil {
		t.Fatalf("GetIncusUserCert: %v", err)
	}
	if row.ProjectScope != "bob" || row.Fingerprint == testFingerprint || row.RevokedAt != nil {
		t.Fatalf("replacement row = %#v", row)
	}
}

func TestUserTrustServiceProvisionRestoresPreviousRowWhenOldRevokeFails(t *testing.T) {
	t.Parallel()
	st := newIncusTrustTestStore(t)
	u, err := st.CreateUser(t.Context(), "alice", "hash", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	u.IncusProject = "bob"
	if err := st.UpsertIncusUserCert(t.Context(), u.ID, "OLD CERT", "OLD KEY", testFingerprint, "alice", time.Now().UTC().Add(24*time.Hour)); err != nil {
		t.Fatalf("UpsertIncusUserCert: %v", err)
	}
	ca, err := NewInternalCertificateAuthority(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewInternalCertificateAuthority: %v", err)
	}
	var newFingerprint string
	service := UserTrustService{
		Store:      st,
		CA:         ca,
		StagingDir: t.TempDir(),
		RequestDir: t.TempDir(),
		RunHelper: func(_ context.Context, _ string, args ...string) error {
			switch args[0] {
			case "register":
				newFingerprint = args[1]
				return nil
			case "revoke":
				if args[1] == testFingerprint {
					return errors.New("revocation failed")
				}
				return nil
			default:
				return nil
			}
		},
	}

	if err := service.Provision(t.Context(), &u); err == nil {
		t.Fatal("Provision unexpectedly succeeded")
	}
	row, err := st.GetIncusUserCert(t.Context(), u.ID)
	if err != nil {
		t.Fatalf("GetIncusUserCert: %v", err)
	}
	if row.Fingerprint != testFingerprint || row.ProjectScope != "alice" {
		t.Fatalf("row was not restored to previous trust: %#v new=%s", row, newFingerprint)
	}
}

func newIncusTrustTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
