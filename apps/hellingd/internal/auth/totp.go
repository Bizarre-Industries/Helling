package auth

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"
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

// NewRecoveryCodes generates count recovery codes, each 16 hex chars.
// Returns the raw codes (to show the user once) and their bcrypt hashes
// (to store in the database).
func NewRecoveryCodes(count int) (raw []string, hashes []string, err error) {
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
		// Format as 4 groups of 4 hex chars: XXXX-XXXX-XXXX-XXXX
		hexStr := hex.EncodeToString(buf)
		raw[i] = strings.ToUpper(hexStr[0:4] + "-" + hexStr[4:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16])

		hash, bcryptErr := bcrypt.GenerateFromPassword([]byte(raw[i]), bcrypt.DefaultCost)
		if bcryptErr != nil {
			return nil, nil, fmt.Errorf("auth.NewRecoveryCodes: bcrypt: %w", bcryptErr)
		}
		hashes[i] = string(hash)
	}
	return raw, hashes, nil
}

// VerifyRecoveryCode checks a raw recovery code against a bcrypt hash.
func VerifyRecoveryCode(raw, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}
