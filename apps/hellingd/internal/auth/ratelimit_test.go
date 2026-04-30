package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

// TestService_LoginRateLimit verifies the alpha-gate Auth checklist item:
// "Rate limiting: 6 failed logins → 429".
//
// Six bad-password attempts against the same user from the same IP must
// trigger ErrRateLimited on the seventh attempt (the API layer maps that to
// HTTP 429). A successful login resets the counter.
func TestService_LoginRateLimit(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if _, err := s.Setup(ctx, "admin", "correct-horse-battery-staple", "127.0.0.1", "go-test"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	const wrongPassword = "definitely-not-the-password"
	for i := range 6 {
		_, err := s.Login(ctx, "admin", wrongPassword, "10.0.0.1", "go-test")
		if !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// 7th attempt — limiter should now block before any password check.
	_, err := s.Login(ctx, "admin", wrongPassword, "10.0.0.1", "go-test")
	if !errors.Is(err, auth.ErrRateLimited) {
		t.Fatalf("attempt 7: expected ErrRateLimited, got %v", err)
	}

	// Even a correct password on the limited (username, ip) is denied.
	_, err = s.Login(ctx, "admin", "correct-horse-battery-staple", "10.0.0.1", "go-test")
	if !errors.Is(err, auth.ErrRateLimited) {
		t.Fatalf("rate-limited correct password: expected ErrRateLimited, got %v", err)
	}

	// Different IP, same user — limiter is per (username, ip) so this is
	// independent and should succeed.
	if _, err := s.Login(ctx, "admin", "correct-horse-battery-staple", "10.0.0.2", "go-test"); err != nil {
		t.Fatalf("different-ip login should succeed, got %v", err)
	}
}

// TestService_LoginRateLimit_ResetsAfterSuccess verifies that a successful
// login on an under-threshold key clears the failure counter so the user is
// not locked out by stale prior fumbles.
func TestService_LoginRateLimit_ResetsAfterSuccess(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if _, err := s.Setup(ctx, "admin", "correct-horse-battery-staple", "127.0.0.1", "go-test"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	for i := range 5 {
		_, err := s.Login(ctx, "admin", "wrong", "10.0.0.5", "go-test")
		if !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// Success at attempt 6 (still under threshold) clears the counter.
	if _, err := s.Login(ctx, "admin", "correct-horse-battery-staple", "10.0.0.5", "go-test"); err != nil {
		t.Fatalf("attempt 6 success: %v", err)
	}

	// Six fresh failures should be needed before next lockout, not zero.
	for i := range 6 {
		_, err := s.Login(ctx, "admin", "wrong-again", "10.0.0.5", "go-test")
		if !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("post-reset attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	_, err := s.Login(ctx, "admin", "wrong-again", "10.0.0.5", "go-test")
	if !errors.Is(err, auth.ErrRateLimited) {
		t.Fatalf("post-reset attempt 7: expected ErrRateLimited, got %v", err)
	}
}
