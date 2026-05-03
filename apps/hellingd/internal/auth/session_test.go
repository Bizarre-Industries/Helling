package auth

import (
	"testing"
)

func TestNewTokenUniqueAndHashStable(t *testing.T) {
	t.Parallel()
	raw1, hash1, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken 1: %v", err)
	}
	raw2, hash2, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken 2: %v", err)
	}
	if raw1 == raw2 {
		t.Fatal("two NewToken calls returned the same raw token")
	}
	if hash1 == hash2 {
		t.Fatal("two NewToken calls returned the same hash")
	}
	// 32 random bytes base64url-encoded → 43 chars (no padding)
	if len(raw1) != 43 {
		t.Fatalf("raw token length: got %d want 43", len(raw1))
	}
	// sha256 hex → 64 chars
	if len(hash1) != 64 {
		t.Fatalf("hash length: got %d want 64", len(hash1))
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	t.Parallel()
	a := HashToken("abc")
	b := HashToken("abc")
	if a != b {
		t.Fatalf("HashToken not deterministic: %s vs %s", a, b)
	}
	if HashToken("abc") == HashToken("abd") {
		t.Fatal("HashToken collided on different inputs")
	}
}
