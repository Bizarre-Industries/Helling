package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

func TestService_EnrollVerifyTOTP_Flow(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, err := s.Setup(ctx, "admin", "testpassword12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	enr, err := s.EnrollTOTP(ctx, ident.UserID)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if enr.Secret == "" || len(enr.RecoveryCodes) != auth.RecoveryCodeCount {
		t.Fatalf("bad enrollment: %+v", enr)
	}
	code, err := auth.GenerateTOTPCode(enr.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTOTPEnroll(ctx, ident.UserID, code); err != nil {
		t.Fatalf("verify enroll: %v", err)
	}
	if err := s.VerifyTOTPEnroll(ctx, ident.UserID, "000000"); err == nil {
		t.Fatal("expected error on wrong totp")
	}
}

func TestService_LoginWithMFA_Branches(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, _ := s.Setup(ctx, "admin", "testpassword12", "", "")

	res, err := s.LoginWithMFA(ctx, "admin", "testpassword12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Identity == nil || res.Pending != nil {
		t.Fatal("expected identity, no pending")
	}

	enr, _ := s.EnrollTOTP(ctx, ident.UserID)
	code, _ := auth.GenerateTOTPCode(enr.Secret)
	_ = s.VerifyTOTPEnroll(ctx, ident.UserID, code)

	res2, err := s.LoginWithMFA(ctx, "admin", "testpassword12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if res2.Pending == nil {
		t.Fatal("expected pending challenge")
	}

	code2, _ := auth.GenerateTOTPCode(enr.Secret)
	got, err := s.CompleteMFA(ctx, res2.Pending.MFAToken, code2, "", "")
	if err != nil {
		t.Fatalf("complete mfa: %v", err)
	}
	if got.AccessToken == "" {
		t.Fatal("missing access token")
	}

	if _, err := s.LoginWithMFA(ctx, "admin", "wrong", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
	if _, err := s.LoginWithMFA(ctx, "ghost", "x", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatal("want ErrInvalidCredentials")
	}
}

func TestService_CompleteMFA_RecoveryCodeFlow(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, _ := s.Setup(ctx, "admin", "testpassword12", "", "")
	enr, _ := s.EnrollTOTP(ctx, ident.UserID)
	code, _ := auth.GenerateTOTPCode(enr.Secret)
	_ = s.VerifyTOTPEnroll(ctx, ident.UserID, code)

	res, _ := s.LoginWithMFA(ctx, "admin", "testpassword12", "", "")
	got, err := s.CompleteMFA(ctx, res.Pending.MFAToken, enr.RecoveryCodes[0], "", "")
	if err != nil {
		t.Fatalf("recovery: %v", err)
	}
	if got.AccessToken == "" {
		t.Fatal("missing access token")
	}

	res2, _ := s.LoginWithMFA(ctx, "admin", "testpassword12", "", "")
	if _, err := s.CompleteMFA(ctx, res2.Pending.MFAToken, enr.RecoveryCodes[0], "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("used code should fail, got %v", err)
	}
}

func TestService_CompleteMFA_InvalidToken(t *testing.T) {
	s := newTestService(t)
	if _, err := s.CompleteMFA(context.Background(), "bogus", "000000", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestService_DisableTOTP(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, _ := s.Setup(ctx, "admin", "testpassword12", "", "")
	_, _ = s.EnrollTOTP(ctx, ident.UserID)
	if err := s.DisableTOTP(ctx, ident.UserID); err != nil {
		t.Fatal(err)
	}
}

func TestService_APITokenLifecycle(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, _ := s.Setup(ctx, "admin", "testpassword12", "", "")

	issued, err := s.CreateAPIToken(ctx, ident.UserID, "ci", "read", 0)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Plaintext == "" {
		t.Fatal("missing plaintext")
	}

	u, tok, err := s.VerifyAPIToken(ctx, issued.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != ident.UserID || tok.Scope != "read" {
		t.Fatalf("unexpected user/scope: %+v / %+v", u, tok)
	}

	toks, err := s.ListAPITokens(ctx, ident.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 1 {
		t.Fatalf("want 1 token, got %d", len(toks))
	}

	if err := s.RevokeAPIToken(ctx, ident.UserID, issued.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.VerifyAPIToken(ctx, issued.Plaintext); err == nil {
		t.Fatal("expected verify to fail after revoke")
	}
}

func TestService_CreateAPIToken_BadScope(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, _ := s.Setup(ctx, "admin", "testpassword12", "", "")
	if _, err := s.CreateAPIToken(ctx, ident.UserID, "bad", "owner", 0); !errors.Is(err, auth.ErrInvalidScope) {
		t.Fatalf("want ErrInvalidScope, got %v", err)
	}
}

func TestService_CreateAPIToken_ClampsTTL(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	ident, _ := s.Setup(ctx, "admin", "testpassword12", "", "")
	issued, err := s.CreateAPIToken(ctx, ident.UserID, "huge-ttl", "read", 10*365*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if issued.ExpiresAt-issued.CreatedAt > int64((366 * 24 * time.Hour).Seconds()) {
		t.Errorf("TTL not clamped: %d seconds", issued.ExpiresAt-issued.CreatedAt)
	}
}
