package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// SessionTokenBytes is the entropy size of a session token before encoding.
// 32 bytes = 256 bits, matching docs/standards/security.md.
const SessionTokenBytes = 32

// NewToken returns (rawCookieValue, sha256HexHash). The raw value is what
// goes in the user's cookie; the hash is what gets persisted in the
// sessions table.
func NewToken() (raw, hash string, err error) {
	buf := make([]byte, SessionTokenBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("auth.NewToken: rand: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken returns the sha256 hex of a raw token. Same value the DB stores.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
