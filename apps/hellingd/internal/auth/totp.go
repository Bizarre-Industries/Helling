package auth

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP parameters fixed by docs/spec/auth.md §2.3 (RFC 6238 SHA-1, 6 digits,
// 30-second period).
const (
	TOTPIssuer     = "Helling"
	TOTPDigits     = otp.DigitsSix
	TOTPPeriod     = 30
	TOTPSkew       = 1
	TOTPSecretSize = 20
)

// TOTPAlg is exposed as a var because otp.Algorithm is not a const.
var TOTPAlg = otp.AlgorithmSHA1

// TOTPEnrollment carries the artifacts a client needs to configure an
// authenticator app.
type TOTPEnrollment struct {
	Secret          string // base32-encoded
	ProvisioningURI string // otpauth://…
}

// ErrTOTPCodeInvalid indicates the supplied six-digit code does not validate
// against the user's stored secret within the configured skew window.
var ErrTOTPCodeInvalid = errors.New("auth: totp code invalid")

// totpNow is replaceable in tests.
var totpNow = time.Now

// EnrollTOTP generates a new TOTP secret + provisioning URI for a user.
// The caller persists the secret (encrypted) and surfaces the URI to the
// user exactly once.
func EnrollTOTP(username string) (TOTPEnrollment, error) {
	if username == "" {
		return TOTPEnrollment{}, errors.New("auth: username required")
	}
	raw := make([]byte, TOTPSecretSize)
	if _, err := rand.Read(raw); err != nil {
		return TOTPEnrollment{}, fmt.Errorf("auth: read totp secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)

	uri := (&url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   "/" + TOTPIssuer + ":" + username,
		RawQuery: url.Values{
			"secret":    []string{secret},
			"issuer":    []string{TOTPIssuer},
			"algorithm": []string{"SHA1"},
			"digits":    []string{"6"},
			"period":    []string{"30"},
		}.Encode(),
	}).String()

	return TOTPEnrollment{Secret: secret, ProvisioningURI: uri}, nil
}

// VerifyTOTP returns nil when the supplied code is valid for the given base32
// secret within a ±TOTPSkew window.
func VerifyTOTP(secret, code string) error {
	if secret == "" || code == "" {
		return ErrTOTPCodeInvalid
	}
	ok, err := totp.ValidateCustom(code, secret, totpNow(), totp.ValidateOpts{
		Period:    TOTPPeriod,
		Skew:      TOTPSkew,
		Digits:    TOTPDigits,
		Algorithm: TOTPAlg,
	})
	if err != nil || !ok {
		return ErrTOTPCodeInvalid
	}
	return nil
}

// GenerateTOTPCode returns the 6-digit code for the given secret at the
// current time. Used by tests and operational tooling only; real clients
// generate their own codes.
func GenerateTOTPCode(secret string) (string, error) {
	return totp.GenerateCodeCustom(secret, totpNow(), totp.ValidateOpts{
		Period:    TOTPPeriod,
		Skew:      TOTPSkew,
		Digits:    TOTPDigits,
		Algorithm: TOTPAlg,
	})
}
