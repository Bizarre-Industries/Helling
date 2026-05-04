package auth

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// TOTPSecretBytes is the entropy size of a TOTP secret. 20 bytes = 160 bits,
// matching RFC 6238 recommendations.
const TOTPSecretBytes = 20

// RecoveryCodeCount is the number of recovery codes generated per TOTP setup.
const RecoveryCodeCount = 10

// RecoveryCodeBytes is the entropy per recovery code. 16 bytes = 128 bits.
const RecoveryCodeBytes = 16

// NewTOTPSecret generates a new base32-encoded TOTP secret suitable for
// use with Google Authenticator and similar apps.
func NewTOTPSecret() (string, error) {
	buf := make([]byte, TOTPSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth.NewTOTPSecret: rand: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// TOTPKeyURI returns the otpauth:// URI for QR code generation.
// issuer and account are URL-encoded in the label.
func TOTPKeyURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer) + ":" + url.PathEscape(account)
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", "6")
	params.Set("period", "30")
	return "otpauth://totp/" + label + "?" + params.Encode()
}

// NewRecoveryCodes generates count recovery codes, each 128 bits.
// Returns the raw codes (to show the user once) and their Argon2id hashes
// (to store in the database).
func NewRecoveryCodes(count int) (raw []string, hashes []string, err error) {
	return NewRecoveryCodesWithParams(count, DefaultArgon2Params())
}

// NewRecoveryCodesWithParams generates recovery codes using caller-supplied
// Argon2id parameters. Tests pass reduced parameters; production uses defaults.
func NewRecoveryCodesWithParams(count int, params Argon2Params) (raw []string, hashes []string, err error) {
	if count <= 0 {
		count = RecoveryCodeCount
	}
	raw = make([]string, count)
	hashes = make([]string, count)
	for i := 0; i < count; i++ {
		buf := make([]byte, RecoveryCodeBytes)
		if _, randErr := rand.Read(buf); randErr != nil {
			return nil, nil, fmt.Errorf("auth.NewRecoveryCodes: rand: %w", randErr)
		}
		// Format as 4 groups of 8 hex chars for the full 128-bit value.
		hexStr := hex.EncodeToString(buf)
		raw[i] = strings.ToUpper(hexStr[0:8] + "-" + hexStr[8:16] + "-" + hexStr[16:24] + "-" + hexStr[24:32])

		hash, hashErr := Hash(raw[i], params)
		if hashErr != nil {
			return nil, nil, fmt.Errorf("auth.NewRecoveryCodes: argon2id: %w", hashErr)
		}
		hashes[i] = hash
	}
	return raw, hashes, nil
}

// VerifyRecoveryCode checks a raw recovery code against an Argon2id hash.
func VerifyRecoveryCode(raw, hash string) bool {
	ok, err := Verify(hash, raw)
	return err == nil && ok
}
