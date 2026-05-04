package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	Role     string `json:"role"`
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

// LoadOrCreateJWTSigner loads a persisted Ed25519 seed or creates one at path.
// The file stores only the base64url-encoded seed and is written 0600.
func LoadOrCreateJWTSigner(path string) (*JWTSigner, error) {
	if path == "" {
		return nil, errors.New("auth.LoadOrCreateJWTSigner: empty path")
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- operator-configured local key path.
	if err == nil {
		return NewJWTSignerFromSeed(string(raw))
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("auth.LoadOrCreateJWTSigner: read %s: %w", path, err)
	}
	signer, err := NewJWTSigner()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("auth.LoadOrCreateJWTSigner: mkdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(signer.Seed()), 0o600); err != nil {
		return nil, fmt.Errorf("auth.LoadOrCreateJWTSigner: write %s: %w", path, err)
	}
	return signer, nil
}

// NewJWTSignerFromSeed restores a signer from a base64-encoded Ed25519 seed.
func NewJWTSignerFromSeed(seedB64 string) (*JWTSigner, error) {
	seed, err := base64.RawStdEncoding.DecodeString(string(trimSpaceBytes([]byte(seedB64))))
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

// Verify validates an EdDSA JWT and returns its Helling claims.
func (s *JWTSigner) Verify(raw string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("auth.Verify: unexpected JWT method %s", token.Method.Alg())
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth.Verify: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("auth.Verify: invalid token")
	}
	return claims, nil
}

// UserID parses the subject as a Helling user ID.
func (c *JWTClaims) UserID() int64 {
	id, _ := strconv.ParseInt(c.Subject, 10, 64)
	return id
}

// NewAccessToken creates a short-lived access JWT for a user.
func (s *JWTSigner) NewAccessToken(userID int64, username string, isAdmin bool, scopes string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	role := "user"
	if isAdmin {
		role = "admin"
	}
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
		Role:     role,
		Scopes:   scopes,
	}
	return s.Sign(claims)
}

func trimSpaceBytes(in []byte) []byte {
	start := 0
	for start < len(in) && (in[start] == ' ' || in[start] == '\n' || in[start] == '\r' || in[start] == '\t') {
		start++
	}
	end := len(in)
	for end > start && (in[end-1] == ' ' || in[end-1] == '\n' || in[end-1] == '\r' || in[end-1] == '\t') {
		end--
	}
	return in[start:end]
}

func mustRandomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("auth.mustRandomHex: %v", err))
	}
	return hex.EncodeToString(buf)
}
