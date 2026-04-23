package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestEnrollTOTP_ShapeAndSelfVerify(t *testing.T) {
	enr, err := EnrollTOTP("alice")
	if err != nil {
		t.Fatal(err)
	}
	if enr.Secret == "" || len(enr.Secret) < 16 {
		t.Errorf("secret too short: %q", enr.Secret)
	}
	if !strings.HasPrefix(enr.ProvisioningURI, "otpauth://totp/Helling:alice?") {
		t.Errorf("bad provisioning URI: %q", enr.ProvisioningURI)
	}
	if !strings.Contains(enr.ProvisioningURI, "issuer=Helling") {
		t.Errorf("missing issuer: %q", enr.ProvisioningURI)
	}

	code, err := GenerateTOTPCode(enr.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTOTP(enr.Secret, code); err != nil {
		t.Fatalf("self-verify: %v", err)
	}
}

func TestEnrollTOTP_RejectsEmptyUsername(t *testing.T) {
	if _, err := EnrollTOTP(""); err == nil {
		t.Fatal("expected error on empty username")
	}
}

func TestVerifyTOTP_RejectsWrongCode(t *testing.T) {
	enr, _ := EnrollTOTP("bob")
	if err := VerifyTOTP(enr.Secret, "000000"); !errors.Is(err, ErrTOTPCodeInvalid) {
		t.Fatalf("want ErrTOTPCodeInvalid, got %v", err)
	}
}

func TestVerifyTOTP_RejectsEmpty(t *testing.T) {
	if err := VerifyTOTP("", ""); !errors.Is(err, ErrTOTPCodeInvalid) {
		t.Fatalf("want ErrTOTPCodeInvalid on empty inputs, got %v", err)
	}
}
