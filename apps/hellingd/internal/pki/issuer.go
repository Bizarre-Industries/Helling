package pki

import (
	"context"
	"errors"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// Issuer ties together the in-memory CA, the on-host age identity, and the
// authrepo persistence layer so userCreate (and renewal cron) can mint a new
// per-user certificate via a single Issuer.IssueForUser(ctx, userID, username)
// call. age-encryption happens here so the repo never sees plaintext PEM.
type Issuer struct {
	CA       *CA
	Identity string // age X25519 secret loaded from p.Identity at boot.
	Repo     *authrepo.Repo
}

// IssueForUser mints a fresh user cert against the CA, age-encrypts the
// PEM artifacts, and writes the row via authrepo.InsertUserCertificate.
func (i *Issuer) IssueForUser(ctx context.Context, userID, username string) error {
	if i == nil || i.CA == nil || i.Repo == nil {
		return errors.New("pki.Issuer: not configured")
	}
	if i.Identity == "" {
		return errors.New("pki.Issuer: missing age identity")
	}
	uc, err := i.CA.IssueUserCert(UserCertRequest{Username: username, UserID: userID})
	if err != nil {
		return err
	}
	encCert, err := EncryptWithIdentity(i.Identity, uc.CertPEM)
	if err != nil {
		return err
	}
	encKey, err := EncryptWithIdentity(i.Identity, uc.PrivateKeyPEM)
	if err != nil {
		return err
	}
	_, err = i.Repo.InsertUserCertificate(ctx, &authrepo.CreateUserCertificateInput{
		UserID:                 userID,
		SerialNumber:           uc.SerialHex,
		CertPEMEncrypted:       encCert,
		PrivateKeyPEMEncrypted: encKey,
		PublicKeySHA256:        toHex(uc.PublicKeySHA),
		IssuedAt:               uc.IssuedAt,
		ExpiresAt:              uc.ExpiresAt,
	})
	return err
}

func toHex(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}
