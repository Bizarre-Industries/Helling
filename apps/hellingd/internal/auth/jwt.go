package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims carries the standard Helling JWT payload. The Subject is the
// user ID (stringified int64). JWT is not used for session auth in v0.1
// (session tokens serve that role); it exists for future API token auth
// and inter-service communication.
type JWTClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	Scopes   string `json:"scopes,omitempty"`
}

// JWTSigner holds an Ed25519 private key used to sign JWTs. The public key
// is derivable from the private key.
type JWTSigner struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewJWTSigner generates a fresh Ed25519 keypair. In production the key
// should be loaded from a persistent file; this constructor is for testing
// and first-boot key generation.
func NewJWTSigner() (*JWTSigner, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("auth.NewJWTSigner: %w", err)
	}
	return &JWTSigner{privateKey: priv, publicKey: pub}, nil
}

// NewJWTSignerFromSeed restores a signer from a base64-encoded Ed25519 seed.
func NewJWTSignerFromSeed(seedB64 string) (*JWTSigner, error) {
	seed, err := base64.RawStdEncoding.DecodeString(seedB64)
	if err != nil {
		return nil, fmt.Errorf("auth.NewJWTSignerFromSeed: decode: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("auth.NewJWTSignerFromSeed: seed must be %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return &JWTSigner{privateKey: priv, publicKey: priv.Public().(ed25519.PublicKey)}, nil
}

// Seed returns the base64-encoded Ed25519 seed for persistence.
func (s *JWTSigner) Seed() string {
	return base64.RawStdEncoding.EncodeToString(s.privateKey.Seed())
}

// PublicKey returns the base64-encoded Ed25519 public key.
func (s *JWTSigner) PublicKey() string {
	return base64.RawStdEncoding.EncodeToString(s.publicKey)
}

// Sign creates a signed JWT string for the given claims.
func (s *JWTSigner) Sign(claims *JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("auth.Sign: %w", err)
	}
	return tokenString, nil
}

// NewAccessToken creates a short-lived access JWT for a user.
func (s *JWTSigner) NewAccessToken(userID int64, username string, isAdmin bool, scopes string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        mustRandomHex(16),
		},
		Username: username,
		IsAdmin:  isAdmin,
		Scopes:   scopes,
	}
	return s.Sign(claims)
}

func mustRandomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("auth.mustRandomHex: %v", err))
	}
	return hex.EncodeToString(buf)
}
