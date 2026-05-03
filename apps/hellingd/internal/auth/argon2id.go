// Package auth owns password hashing, session token minting, and login
// rate limiting for hellingd. All cryptographic primitives are concentrated
// here so they can be audited in one place.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params controls the cost and shape of the password hash. Defaults
// match docs/standards/security.md (time=3, memory=64 MiB, parallelism=4,
// saltLen=16, keyLen=32).
type Argon2Params struct {
	Time        uint32
	MemoryKiB   uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

// DefaultArgon2Params returns the production-default Argon2id parameters.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Time:        3,
		MemoryKiB:   64 * 1024,
		Parallelism: 4,
		SaltLen:     16,
		KeyLen:      32,
	}
}

// ErrInvalidHash is returned when a stored hash cannot be parsed.
var ErrInvalidHash = errors.New("auth: invalid argon2id hash")

// Hash derives an Argon2id hash from the password and returns the PHC-encoded
// string suitable for direct storage in the users.password_hash column.
func Hash(password string, p Argon2Params) (string, error) {
	if password == "" {
		return "", errors.New("auth.Hash: empty password")
	}
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth.Hash: salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKiB, p.Parallelism, p.KeyLen)
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.MemoryKiB, p.Time, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, nil
}

// Verify constant-time-compares the password against a stored PHC hash.
// Returns (true, nil) on match; (false, nil) on mismatch with valid hash;
// (false, err) on parse failure.
func Verify(encoded, password string) (bool, error) {
	p, salt, key, err := decodeArgon2idHash(encoded)
	if err != nil {
		return false, err
	}
	candidate := argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKiB, p.Parallelism, uint32(len(key))) //nolint:gosec // len(key) is bounded by argon2 KeyLen (max 32 in our hashes)
	return subtle.ConstantTimeCompare(key, candidate) == 1, nil
}

// decodeArgon2idHash parses an argon2id PHC string into its parts.
func decodeArgon2idHash(encoded string) (params Argon2Params, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	// Expected: ["", "argon2id", "v=19", "m=65536,t=3,p=4", "<salt>", "<key>"]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	var p Argon2Params
	if _, scanErr := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Time, &p.Parallelism); scanErr != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLen = uint32(len(salt)) //nolint:gosec // salt length is bounded by argon2 SaltLen
	p.KeyLen = uint32(len(key))   //nolint:gosec // key length is bounded by argon2 KeyLen
	params = p
	return params, salt, key, nil
}
