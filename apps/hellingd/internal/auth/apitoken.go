package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"
)

// APITokenPrefix is the required user-visible prefix for API tokens per
// docs/spec/auth.md §5.
const APITokenPrefix = "helling_"

// APITokenRawBytes is the raw entropy before base32 encoding. 24 bytes →
// 39 base32 chars without padding, producing ~192 bits of entropy.
const APITokenRawBytes = 24

// Valid scopes per docs/spec/auth.md §5.
const (
	APITokenScopeRead  = "read"
	APITokenScopeWrite = "write"
	APITokenScopeAdmin = "admin"
)

// ErrInvalidScope is returned when an API-token scope is not one of the
// values listed in docs/spec/auth.md §5.
var ErrInvalidScope = errors.New("auth: invalid token scope")

// ValidAPITokenScopes enumerates the accepted scope values.
var ValidAPITokenScopes = []string{APITokenScopeRead, APITokenScopeWrite, APITokenScopeAdmin}

// IsValidAPITokenScope reports whether the given scope is one of the fixed
// Helling token scopes.
func IsValidAPITokenScope(scope string) bool {
	switch scope {
	case APITokenScopeRead, APITokenScopeWrite, APITokenScopeAdmin:
		return true
	default:
		return false
	}
}

// GenerateAPIToken returns a freshly-minted API token and its SHA-256
// lowercase-hex hash. Helling stores only the hash; the plaintext token is
// surfaced to the user exactly once per docs/spec/auth.md §5.
func GenerateAPIToken() (plaintext, tokenHash string, err error) {
	buf := make([]byte, APITokenRawBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	plain := APITokenPrefix + strings.ToLower(enc)
	return plain, HashAPIToken(plain), nil
}

// HashAPIToken returns the SHA-256 lowercase-hex digest of an existing
// plaintext token (e.g. during Bearer-auth verification).
func HashAPIToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}
