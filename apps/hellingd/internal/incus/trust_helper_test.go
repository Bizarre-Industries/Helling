package incus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const testFingerprint = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestTrustHelperRegisterUsesRestrictedProjectCommand(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	requestDir := t.TempDir()
	policyDir := t.TempDir()
	policyOwnerUID := testPolicyOwnerUID(t)
	certName, fingerprint := writeTrustHelperCert(t, stageDir)
	requestName := writeTrustHelperRequest(t, requestDir, trustRequestManifest{
		Action:       "register",
		UserID:       42,
		Fingerprint:  fingerprint,
		Project:      "alice",
		CertBasename: certName,
	})
	writeTrustHelperPolicy(t, policyDir, 42, "alice", "register")
	var calls []trustCommandCall
	helper := TrustHelper{
		StagingDir:     stageDir,
		RequestDir:     requestDir,
		PolicyDir:      policyDir,
		PolicyOwnerUID: &policyOwnerUID,
		RunCommand: func(_ context.Context, name string, args ...string) error {
			if _, err := os.Stat(filepath.Join(requestDir, requestName)); !os.IsNotExist(err) {
				t.Fatalf("request was not consumed before command, stat err=%v", err)
			}
			calls = append(calls, trustCommandCall{name: name, args: append([]string(nil), args...)})
			return nil
		},
	}

	if err := helper.Handle(t.Context(), []string{"register", fingerprint, "alice", certName, requestName}); err != nil {
		t.Fatalf("Handle register: %v", err)
	}
	want := []string{
		"--force-local",
		"config", "trust", "add-certificate",
		filepath.Join(stageDir, certName),
		"--name", "helling-user-42-" + fingerprint[:12],
		"--restricted",
		"--projects", "alice",
	}
	assertTrustCommandCalls(t, calls, []trustCommandCall{{name: incusCLIPath, args: want}})
}

func TestTrustHelperRevokeUsesFingerprintOnly(t *testing.T) {
	t.Parallel()
	requestDir := t.TempDir()
	policyDir := t.TempDir()
	policyOwnerUID := testPolicyOwnerUID(t)
	requestName := writeTrustHelperRequest(t, requestDir, trustRequestManifest{
		Action:      "revoke",
		UserID:      42,
		Fingerprint: testFingerprint,
		Project:     "alice",
	})
	writeTrustHelperPolicy(t, policyDir, 42, "alice", "revoke")
	var calls []trustCommandCall
	helper := TrustHelper{
		RequestDir:     requestDir,
		PolicyDir:      policyDir,
		PolicyOwnerUID: &policyOwnerUID,
		RunCommand: func(_ context.Context, name string, args ...string) error {
			if _, err := os.Stat(filepath.Join(requestDir, requestName)); !os.IsNotExist(err) {
				t.Fatalf("request was not consumed before command, stat err=%v", err)
			}
			calls = append(calls, trustCommandCall{name: name, args: append([]string(nil), args...)})
			return nil
		},
	}

	if err := helper.Handle(t.Context(), []string{"revoke", testFingerprint, requestName}); err != nil {
		t.Fatalf("Handle revoke: %v", err)
	}
	want := []string{"--force-local", "config", "trust", "remove", testFingerprint}
	assertTrustCommandCalls(t, calls, []trustCommandCall{{name: incusCLIPath, args: want}})
}

func TestTrustHelperRejectsUnsafeArgs(t *testing.T) {
	t.Parallel()
	helper := TrustHelper{}
	for _, args := range [][]string{
		{"register", "bad", "default", "helling-user-42-" + testFingerprint + ".crt", trustRequestName("register", 42, testFingerprint)},
		{"register", testFingerprint, "../default", "helling-user-42-" + testFingerprint + ".crt", trustRequestName("register", 42, testFingerprint)},
		{"register", testFingerprint, "default", "../helling-user-42-" + testFingerprint + ".crt", trustRequestName("register", 42, testFingerprint)},
		{"register", testFingerprint, "default", "helling-user-42-ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff.crt", trustRequestName("register", 42, testFingerprint)},
		{"register", testFingerprint, "default", "helling-user-42-" + testFingerprint + ".crt", "../" + trustRequestName("register", 42, testFingerprint)},
		{"revoke", "../" + testFingerprint, trustRequestName("revoke", 42, testFingerprint)},
		{"delete", testFingerprint},
	} {
		if err := helper.Handle(t.Context(), args); err == nil {
			t.Fatalf("Handle(%v) unexpectedly succeeded", args)
		}
	}
}

func TestTrustHelperRejectsForgedOrInsecurePolicy(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	requestDir := t.TempDir()
	policyDir := t.TempDir()
	certName, fingerprint := writeTrustHelperCert(t, stageDir)
	requestName := writeTrustHelperRequest(t, requestDir, trustRequestManifest{
		Action:       "register",
		UserID:       42,
		Fingerprint:  fingerprint,
		Project:      "alice",
		CertBasename: certName,
	})
	writeTrustHelperPolicy(t, policyDir, 42, "alice", "register")
	wrongOwnerUID := testPolicyOwnerUID(t) + 1
	helper := TrustHelper{
		StagingDir:     stageDir,
		RequestDir:     requestDir,
		PolicyDir:      policyDir,
		PolicyOwnerUID: &wrongOwnerUID,
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("RunCommand should not be called")
			return nil
		},
	}
	if err := helper.Handle(t.Context(), []string{"register", fingerprint, "alice", certName, requestName}); err == nil {
		t.Fatal("Handle register unexpectedly accepted policy with forged owner")
	}

	if err := os.Chmod(filepath.Join(policyDir, "helling-user-42.json"), 0o660); err != nil {
		t.Fatalf("chmod policy: %v", err)
	}
	ownerUID := testPolicyOwnerUID(t)
	helper.PolicyOwnerUID = &ownerUID
	if err := helper.Handle(t.Context(), []string{"register", fingerprint, "alice", certName, requestName}); err == nil {
		t.Fatal("Handle register unexpectedly accepted group-writable policy")
	}
}

func TestTrustHelperRejectsMismatchedRequestAndPolicy(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	requestDir := t.TempDir()
	policyDir := t.TempDir()
	policyOwnerUID := testPolicyOwnerUID(t)
	certName, fingerprint := writeTrustHelperCert(t, stageDir)
	writeTrustHelperPolicy(t, policyDir, 42, "alice", "register")

	projectMismatch := writeTrustHelperRequest(t, requestDir, trustRequestManifest{
		Action:       "register",
		UserID:       42,
		Fingerprint:  fingerprint,
		Project:      "bob",
		CertBasename: certName,
	})
	helper := TrustHelper{
		StagingDir:     stageDir,
		RequestDir:     requestDir,
		PolicyDir:      policyDir,
		PolicyOwnerUID: &policyOwnerUID,
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("RunCommand should not be called")
			return nil
		},
	}
	if err := helper.Handle(t.Context(), []string{"register", fingerprint, "alice", certName, projectMismatch}); err == nil {
		t.Fatal("Handle register unexpectedly accepted mismatched project")
	}

	fingerprintMismatch := writeTrustHelperRequest(t, requestDir, trustRequestManifest{
		Action:       "register",
		UserID:       42,
		Fingerprint:  testFingerprint,
		Project:      "alice",
		CertBasename: certName,
	})
	fingerprintMismatchName := trustRequestName("register", 42, fingerprint)
	if err := os.Rename(filepath.Join(requestDir, fingerprintMismatch), filepath.Join(requestDir, fingerprintMismatchName)); err != nil {
		t.Fatalf("rename request: %v", err)
	}
	if err := helper.Handle(t.Context(), []string{"register", fingerprint, "alice", certName, fingerprintMismatchName}); err == nil {
		t.Fatal("Handle register unexpectedly accepted mismatched fingerprint")
	}
}

func TestTrustHelperUsesDefaultPolicyForAllowedProject(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	requestDir := t.TempDir()
	policyDir := t.TempDir()
	policyOwnerUID := testPolicyOwnerUID(t)
	certName, fingerprint := writeTrustHelperCert(t, stageDir)
	requestName := writeTrustHelperRequest(t, requestDir, trustRequestManifest{
		Action:       "register",
		UserID:       42,
		Fingerprint:  fingerprint,
		Project:      "default",
		CertBasename: certName,
	})
	writeTrustHelperDefaultPolicy(t, policyDir, []string{"default"}, "register")
	var called bool
	helper := TrustHelper{
		StagingDir:     stageDir,
		RequestDir:     requestDir,
		PolicyDir:      policyDir,
		PolicyOwnerUID: &policyOwnerUID,
		RunCommand: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}
	if err := helper.Handle(t.Context(), []string{"register", fingerprint, "default", certName, requestName}); err != nil {
		t.Fatalf("Handle register with default policy: %v", err)
	}
	if !called {
		t.Fatal("RunCommand was not called")
	}
}

func writeTrustHelperCert(t *testing.T, dir string) (name string, fingerprint string) {
	t.Helper()
	certPEM, _, fingerprint, _, err := IssueUserCertificate(42, "alice", time.Now().UTC())
	if err != nil {
		t.Fatalf("IssueUserCertificate: %v", err)
	}
	name = "helling-user-42-" + fingerprint + ".crt"
	if err := os.WriteFile(filepath.Join(dir, name), certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	return name, fingerprint
}

func writeTrustHelperRequest(t *testing.T, dir string, request trustRequestManifest) string {
	t.Helper()
	name := trustRequestName(request.Action, request.UserID, request.Fingerprint)
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
		t.Fatalf("write request: %v", err)
	}
	return name
}

func writeTrustHelperPolicy(t *testing.T, dir string, userID int64, project string, actions ...string) { //nolint:unparam // Keep userID explicit so policy fixtures mirror real filenames.
	t.Helper()
	body, err := json.Marshal(trustPolicy{UserID: userID, Project: project, Actions: actions})
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "helling-user-"+strconv.FormatInt(userID, 10)+".json"), body, 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
}

func writeTrustHelperDefaultPolicy(t *testing.T, dir string, projects []string, actions ...string) {
	t.Helper()
	body, err := json.Marshal(trustPolicy{UserID: 0, Projects: projects, Actions: actions})
	if err != nil {
		t.Fatalf("marshal default policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "default.json"), body, 0o600); err != nil {
		t.Fatalf("write default policy: %v", err)
	}
}

func trustRequestName(action string, userID int64, fingerprint string) string {
	return "helling-trust-" + action + "-" + strconv.FormatInt(userID, 10) + "-" + fingerprint + "-0011223344556677.json"
}

func testPolicyOwnerUID(t *testing.T) uint32 {
	t.Helper()
	uid, err := strconv.ParseUint(strconv.Itoa(os.Getuid()), 10, 32)
	if err != nil {
		t.Fatalf("parse uid: %v", err)
	}
	return uint32(uid)
}

type trustCommandCall struct {
	name string
	args []string
}

func assertTrustCommandCalls(t *testing.T, got, want []trustCommandCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("commands: got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i].name != want[i].name {
			t.Fatalf("command %d name: got %q want %q", i, got[i].name, want[i].name)
		}
		if len(got[i].args) != len(want[i].args) {
			t.Fatalf("command %d args: got %#v want %#v", i, got[i].args, want[i].args)
		}
		for j := range want[i].args {
			if got[i].args[j] != want[i].args[j] {
				t.Fatalf("command %d arg %d: got %q want %q", i, j, got[i].args[j], want[i].args[j])
			}
		}
	}
}
