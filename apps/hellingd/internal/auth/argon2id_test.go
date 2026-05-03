package auth

import (
	"strings"
	"testing"
)

// fastParams keeps tests under a few ms; production uses DefaultArgon2Params.
func fastParams() Argon2Params {
	return Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32}
}

func TestHashVerifyRoundtrip(t *testing.T) {
	t.Parallel()
	p := fastParams()

	hash, err := Hash("correct horse battery staple", p)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash missing PHC prefix: %s", hash)
	}

	ok, err := Verify(hash, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Verify ok-case: %v", err)
	}
	if !ok {
		t.Fatal("Verify returned false for correct password")
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	t.Parallel()
	hash, err := Hash("hunter2", fastParams())
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	ok, err := Verify(hash, "hunter3")
	if err != nil {
		t.Fatalf("Verify wrong-case: %v", err)
	}
	if ok {
		t.Fatal("Verify returned true for wrong password")
	}
}

func TestHashRefusesEmptyPassword(t *testing.T) {
	t.Parallel()
	if _, err := Hash("", fastParams()); err == nil {
		t.Fatal("Hash(\"\") returned nil error")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"plaintext",
		"$argon2id$v=19$m=8192,t=1,p=1$bad-base64!$bad-base64!",
		"$argon2i$v=19$m=8192,t=1,p=1$AAAA$BBBB",
	}
	for _, c := range cases {
		if _, err := Verify(c, "x"); err == nil {
			t.Errorf("Verify(%q) returned nil error", c)
		}
	}
}
