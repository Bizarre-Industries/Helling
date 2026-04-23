package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIToken_ShapeAndDigest(t *testing.T) {
	plain, hash, err := GenerateAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plain, APITokenPrefix) {
		t.Fatalf("token missing prefix: %q", plain)
	}
	if plain != strings.ToLower(plain) {
		t.Errorf("token must be lowercase: %q", plain)
	}
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
	if hash != HashAPIToken(plain) {
		t.Fatal("HashAPIToken must be deterministic for same plaintext")
	}
	plain2, _, _ := GenerateAPIToken()
	if plain == plain2 {
		t.Fatal("two generated tokens must differ")
	}
}

func TestIsValidAPITokenScope(t *testing.T) {
	for _, s := range []string{"read", "write", "admin"} {
		if !IsValidAPITokenScope(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range []string{"", "owner", "ADMIN", "user"} {
		if IsValidAPITokenScope(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestValidAPITokenScopesList(t *testing.T) {
	if len(ValidAPITokenScopes) != 3 {
		t.Errorf("ValidAPITokenScopes len = %d, want 3", len(ValidAPITokenScopes))
	}
}
