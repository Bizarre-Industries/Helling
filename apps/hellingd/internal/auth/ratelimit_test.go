package auth

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToLimit(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(5, time.Minute)
	for i := 0; i < 5; i++ {
		if !rl.Allow("user") {
			t.Fatalf("attempt %d: expected allow", i+1)
		}
	}
	if rl.Allow("user") {
		t.Fatal("attempt 6: expected deny")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(2, time.Minute)
	for i := 0; i < 2; i++ {
		_ = rl.Allow("alice")
	}
	if rl.Allow("alice") {
		t.Fatal("alice should be over limit")
	}
	if !rl.Allow("bob") {
		t.Fatal("bob should still be allowed")
	}
}

func TestRateLimiterReset(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, time.Minute)
	if !rl.Allow("k") {
		t.Fatal("first should allow")
	}
	if rl.Allow("k") {
		t.Fatal("second should deny")
	}
	rl.Reset("k")
	if !rl.Allow("k") {
		t.Fatal("after Reset, should allow again")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 50*time.Millisecond)
	if !rl.Allow("k") {
		t.Fatal("first should allow")
	}
	if rl.Allow("k") {
		t.Fatal("second should deny within window")
	}
	time.Sleep(80 * time.Millisecond)
	if !rl.Allow("k") {
		t.Fatal("after window expiry, should allow")
	}
}
