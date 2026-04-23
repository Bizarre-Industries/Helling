package auth

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
)

// RecoveryCodeCount is fixed at 10 per docs/spec/auth.md §2.3.
const RecoveryCodeCount = 10

// recoveryCodeRawBytes is the random byte count before base32 encoding.
// 10 bytes → 16 base32 chars without padding.
const recoveryCodeRawBytes = 10

// GenerateRecoveryCodes returns RecoveryCodeCount new recovery codes.
// Codes are base32-encoded uppercase. The caller stores argon2id hashes of
// each code and returns the plaintext list to the user exactly once.
func GenerateRecoveryCodes() ([]string, error) {
	out := make([]string, 0, RecoveryCodeCount)
	for range RecoveryCodeCount {
		code, err := generateOneRecoveryCode()
		if err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, nil
}

func generateOneRecoveryCode() (string, error) {
	buf := make([]byte, recoveryCodeRawBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return strings.ToUpper(encoded), nil
}

// HashRecoveryCodes returns argon2id PHC hashes of the given plaintext codes.
// Callers persist the hashes in recovery_codes.code_hash per
// docs/spec/sqlite-schema.md §2.2.
func HashRecoveryCodes(codes []string) ([]string, error) {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		h, err := HashPassword(c, DefaultArgon2idParams)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

// ErrRecoveryCodeInvalid is returned when a candidate does not match any
// un-used stored hash.
var ErrRecoveryCodeInvalid = errors.New("auth: recovery code invalid")

// MatchRecoveryCode checks the candidate against each supplied hash and
// returns the index of the first match, or ErrRecoveryCodeInvalid.
func MatchRecoveryCode(candidate string, hashes []string) (int, error) {
	for i, h := range hashes {
		if err := VerifyPassword(candidate, h); err == nil {
			return i, nil
		}
	}
	return -1, ErrRecoveryCodeInvalid
}
