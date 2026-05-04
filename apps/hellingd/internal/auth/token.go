package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// APITokenPrefix is the human-readable prefix for all Helling API tokens.
const APITokenPrefix = "helling_"

// APITokenRandomBytes is the entropy size of the random portion of an API
// token. 40 bytes = 320 bits, matching docs/standards/security.md.
const APITokenRandomBytes = 40

const (
	// ScopeRead allows read-only API token access.
	ScopeRead = "read"
	// ScopeWrite allows read and write API token access.
	ScopeWrite = "write"
	// ScopeAdmin allows admin API token access when the user is also admin.
	ScopeAdmin = "admin"
)

const apiTokenCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// NewAPIToken returns (rawToken, sha256HexHash). The raw token has the form
// "helling_<random 40 alphanumeric chars>". The hash is what gets persisted
// in the api_tokens table. The raw token is shown once at creation.
func NewAPIToken() (raw, hash string, err error) {
	buf := make([]byte, APITokenRandomBytes)
	for i := range buf {
		n, randErr := rand.Int(rand.Reader, big.NewInt(int64(len(apiTokenCharset))))
		if randErr != nil {
			return "", "", fmt.Errorf("auth.NewAPIToken: rand: %w", randErr)
		}
		buf[i] = apiTokenCharset[n.Int64()]
	}
	raw = APITokenPrefix + string(buf)
	hash = HashAPIToken(raw)
	return raw, hash, nil
}

// HashAPIToken returns the sha256 hex of a raw API token.
func HashAPIToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ValidScopes returns true if the scopes string is one of the allowed values.
func ValidScopes(scopes string) bool {
	switch scopes {
	case ScopeRead, ScopeWrite, ScopeAdmin:
		return true
	default:
		return false
	}
}
