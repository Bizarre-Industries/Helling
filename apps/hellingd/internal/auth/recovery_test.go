package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateRecoveryCodes_Count(t *testing.T) {
	codes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("got %d codes, want %d", len(codes), RecoveryCodeCount)
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if seen[c] {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = true
		if c != strings.ToUpper(c) {
			t.Errorf("code not uppercased: %q", c)
		}
	}
}

func TestHashRecoveryCodes_Roundtrip(t *testing.T) {
	codes := []string{"ABCDEFGH12345678", "ZZZZ1111YYYY2222"}
	hashes, err := HashRecoveryCodes(codes)
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != len(codes) {
		t.Fatalf("hashes len = %d", len(hashes))
	}
	idx, err := MatchRecoveryCode("ZZZZ1111YYYY2222", hashes)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if idx != 1 {
		t.Errorf("expected match idx 1, got %d", idx)
	}
}

func TestMatchRecoveryCode_NoMatchReturnsError(t *testing.T) {
	hashes, _ := HashRecoveryCodes([]string{"CODE1"})
	if _, err := MatchRecoveryCode("wrong", hashes); !errors.Is(err, ErrRecoveryCodeInvalid) {
		t.Fatalf("want ErrRecoveryCodeInvalid, got %v", err)
	}
}
