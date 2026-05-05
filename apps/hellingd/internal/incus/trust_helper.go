package incus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
)

const (
	// DefaultTrustStagingDir is where hellingd stages certificate PEM files for the helper.
	DefaultTrustStagingDir = "/var/lib/helling/incus-trust"
	// DefaultTrustRequestDir is where hellingd stages one-time trust request manifests.
	DefaultTrustRequestDir = "/var/lib/helling/incus-trust/requests"
	// DefaultTrustPolicyDir is the root-owned trust policy boundary consumed by the helper.
	DefaultTrustPolicyDir = "/etc/helling/incus-trust.d"
	incusCLIPath          = "/usr/bin/incus"
	trustActionRegister   = "register"
	trustActionRevoke     = "revoke"
)

var (
	trustFingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	trustProjectPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)
	trustCertNamePattern    = regexp.MustCompile(`^helling-user-(\d+)-([0-9a-f]{64})\.crt$`)
	trustRequestNamePattern = regexp.MustCompile(`^helling-trust-(register|revoke)-(\d+)-([0-9a-f]{64})-[0-9a-f]{16}\.json$`)
)

// TrustHelper implements the narrow Incus trust mutation helper surface.
type TrustHelper struct {
	StagingDir     string
	RequestDir     string
	PolicyDir      string
	PolicyOwnerUID *uint32
	RunCommand     func(ctx context.Context, name string, args ...string) error
}

// Handle validates helper args and executes register/revoke.
func (h TrustHelper) Handle(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: helling-incus-trust register|revoke ...")
	}
	switch args[0] {
	case trustActionRegister:
		if len(args) != 5 {
			return errors.New("usage: helling-incus-trust register <fingerprint> <project> <cert-basename> <request-basename>")
		}
		return h.register(ctx, args[1], args[2], args[3], args[4])
	case trustActionRevoke:
		if len(args) != 3 {
			return errors.New("usage: helling-incus-trust revoke <fingerprint> <request-basename>")
		}
		return h.revoke(ctx, args[1], args[2])
	default:
		return fmt.Errorf("unsupported Incus trust action %q", args[0])
	}
}

func (h TrustHelper) register(ctx context.Context, fingerprint, project, certName, requestName string) error {
	if err := validateTrustFingerprint(fingerprint); err != nil {
		return err
	}
	if !trustProjectPattern.MatchString(project) {
		return fmt.Errorf("invalid Incus project %q", project)
	}
	matches := trustCertNamePattern.FindStringSubmatch(certName)
	if matches == nil {
		return fmt.Errorf("invalid trust certificate basename %q", certName)
	}
	if filepath.Base(certName) != certName {
		return fmt.Errorf("trust certificate %q must be a basename", certName)
	}
	if matches[2] != fingerprint {
		return errors.New("trust certificate basename fingerprint does not match fingerprint argument")
	}
	userID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil || userID <= 0 {
		return fmt.Errorf("invalid trust certificate user id %q", matches[1])
	}
	requestPath, err := h.validateRequest(trustActionRegister, userID, fingerprint, project, certName, requestName)
	if err != nil {
		return err
	}
	if err := validateStagedTrustCertificate(filepath.Join(h.stagingDir(), certName), fingerprint); err != nil {
		return err
	}
	if err := os.Remove(requestPath); err != nil {
		return fmt.Errorf("consuming Incus trust request: %w", err)
	}
	name := fmt.Sprintf("helling-user-%s-%s", matches[1], fingerprint[:12])
	return h.run(ctx,
		incusCLIPath,
		"--force-local",
		"config", "trust", "add-certificate",
		filepath.Join(h.stagingDir(), certName),
		"--name", name,
		"--restricted",
		"--projects", project,
	)
}

func (h TrustHelper) revoke(ctx context.Context, fingerprint, requestName string) error {
	if err := validateTrustFingerprint(fingerprint); err != nil {
		return err
	}
	requestPath, err := h.validateRequest(trustActionRevoke, 0, fingerprint, "", "", requestName)
	if err != nil {
		return err
	}
	if err := os.Remove(requestPath); err != nil {
		return fmt.Errorf("consuming Incus trust request: %w", err)
	}
	return h.run(ctx, incusCLIPath, "--force-local", "config", "trust", "remove", fingerprint)
}

func validateTrustFingerprint(fingerprint string) error {
	if !trustFingerprintPattern.MatchString(fingerprint) {
		return fmt.Errorf("invalid Incus trust fingerprint %q", fingerprint)
	}
	return nil
}

// ValidTrustProjectName reports whether project is safe for the trust helper argv.
func ValidTrustProjectName(project string) bool {
	return trustProjectPattern.MatchString(project)
}

func (h TrustHelper) stagingDir() string {
	if h.StagingDir != "" {
		return h.StagingDir
	}
	return DefaultTrustStagingDir
}

func (h TrustHelper) requestDir() string {
	if h.RequestDir != "" {
		return h.RequestDir
	}
	return DefaultTrustRequestDir
}

func (h TrustHelper) policyDir() string {
	if h.PolicyDir != "" {
		return h.PolicyDir
	}
	return DefaultTrustPolicyDir
}

func (h TrustHelper) policyOwnerUID() uint32 {
	if h.PolicyOwnerUID != nil {
		return *h.PolicyOwnerUID
	}
	return 0
}

func (h TrustHelper) run(ctx context.Context, name string, args ...string) error {
	if h.RunCommand != nil {
		return h.RunCommand(ctx, name, args...)
	}
	// #nosec G204 -- name is fixed by this helper and args are validated command literals.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C"}
	return cmd.Run()
}

type trustRequestManifest struct {
	Action       string `json:"action"`
	UserID       int64  `json:"user_id"`
	Fingerprint  string `json:"fingerprint"`
	Project      string `json:"project,omitempty"`
	CertBasename string `json:"cert_basename,omitempty"`
}

type trustPolicy struct {
	UserID   int64    `json:"user_id"`
	Project  string   `json:"project,omitempty"`
	Projects []string `json:"projects,omitempty"`
	Actions  []string `json:"actions"`
}

func (h TrustHelper) validateRequest(action string, userID int64, fingerprint, project, certName, requestName string) (string, error) {
	requestUserID, err := parseTrustRequestName(action, userID, fingerprint, requestName)
	if err != nil {
		return "", err
	}
	requestPath := filepath.Join(h.requestDir(), requestName)
	var request trustRequestManifest
	if err := readLockedDownJSONFile(requestPath, "trust request", 64*1024, 0, false, &request); err != nil {
		return "", err
	}
	if err := validateTrustRequestManifest(request, action, requestUserID, fingerprint, project, certName); err != nil {
		return "", err
	}
	policy, err := h.loadPolicy(requestUserID)
	if err != nil {
		return "", err
	}
	if policy.UserID != 0 && policy.UserID != requestUserID {
		return "", errors.New("trust policy user id does not match request")
	}
	if !policyAllowsProject(policy, request.Project) {
		return "", errors.New("trust policy project does not match request")
	}
	if !policyAllowsAction(policy, action) {
		return "", fmt.Errorf("trust policy does not allow %s", action)
	}
	return requestPath, nil
}

func parseTrustRequestName(action string, userID int64, fingerprint, requestName string) (int64, error) {
	if filepath.Base(requestName) != requestName {
		return 0, fmt.Errorf("trust request %q must be a basename", requestName)
	}
	matches := trustRequestNamePattern.FindStringSubmatch(requestName)
	if matches == nil {
		return 0, fmt.Errorf("invalid trust request basename %q", requestName)
	}
	if matches[1] != action {
		return 0, errors.New("trust request action does not match helper command")
	}
	requestUserID, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil || requestUserID <= 0 {
		return 0, fmt.Errorf("invalid trust request user id %q", matches[2])
	}
	if action == trustActionRegister && requestUserID != userID {
		return 0, errors.New("trust request user id does not match certificate basename")
	}
	if matches[3] != fingerprint {
		return 0, errors.New("trust request fingerprint does not match helper command")
	}
	return requestUserID, nil
}

func validateTrustRequestManifest(request trustRequestManifest, action string, userID int64, fingerprint, project, certName string) error {
	if request.Action != action || request.UserID != userID || request.Fingerprint != fingerprint || request.CertBasename != certName {
		return errors.New("trust request manifest does not match helper command")
	}
	if action == trustActionRegister && request.Project != project {
		return errors.New("trust request project does not match helper command")
	}
	return nil
}

func (h TrustHelper) loadPolicy(userID int64) (trustPolicy, error) {
	var policy trustPolicy
	policyName := fmt.Sprintf("helling-user-%d.json", userID)
	if filepath.Base(policyName) != policyName {
		return policy, fmt.Errorf("invalid trust policy basename %q", policyName)
	}
	if err := verifyLockedDownPath(h.policyDir(), "trust policy directory", h.policyOwnerUID(), true); err != nil {
		return policy, err
	}
	policyPath := filepath.Join(h.policyDir(), policyName)
	if err := readLockedDownJSONFile(policyPath, "trust policy", 64*1024, h.policyOwnerUID(), true, &policy); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return policy, err
		}
		policyPath = filepath.Join(h.policyDir(), "default.json")
		if err := readLockedDownJSONFile(policyPath, "default trust policy", 64*1024, h.policyOwnerUID(), true, &policy); err != nil {
			return policy, err
		}
	}
	if policy.UserID < 0 {
		return policy, errors.New("trust policy user id must be non-negative")
	}
	if policy.UserID != 0 && policy.UserID != userID {
		return policy, errors.New("trust policy user id does not match request")
	}
	if err := validateTrustPolicyProjects(policy); err != nil {
		return policy, err
	}
	return policy, nil
}

func validateTrustPolicyProjects(policy trustPolicy) error {
	if policy.Project == "" && len(policy.Projects) == 0 {
		return errors.New("trust policy project is required")
	}
	if policy.Project != "" && !trustProjectPattern.MatchString(policy.Project) {
		return fmt.Errorf("invalid trust policy project %q", policy.Project)
	}
	for _, project := range policy.Projects {
		if !trustProjectPattern.MatchString(project) {
			return fmt.Errorf("invalid trust policy project %q", project)
		}
	}
	return nil
}

func policyAllowsProject(policy trustPolicy, project string) bool {
	if policy.Project == project {
		return true
	}
	for _, allowed := range policy.Projects {
		if allowed == project {
			return true
		}
	}
	return false
}

func policyAllowsAction(policy trustPolicy, action string) bool {
	for _, allowed := range policy.Actions {
		if allowed == action {
			return true
		}
	}
	return false
}

func readLockedDownJSONFile(path, label string, maxBytes int64, ownerUID uint32, requireOwner bool, out any) error {
	if err := verifyLockedDownPath(path, label, ownerUID, requireOwner); err != nil {
		return err
	}
	file, err := os.Open(path) // #nosec G304 -- path is built from validated basenames under fixed helper dirs.
	if err != nil {
		return fmt.Errorf("opening %s: %w", label, err)
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("reading %s: %w", label, err)
	}
	if int64(len(body)) > maxBytes {
		return fmt.Errorf("%s is too large", label)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decoding %s: %w", label, err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("%s contains trailing JSON", label)
	}
	return nil
}

func verifyLockedDownPath(path, label string, ownerUID uint32, requireOwner bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s must not be a symlink", label)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%s must not be group/world writable", label)
	}
	if requireOwner {
		uid, err := fileUID(info)
		if err != nil {
			return err
		}
		if uid != ownerUID {
			return fmt.Errorf("%s owner uid %d does not match required uid %d", label, uid, ownerUID)
		}
	}
	return nil
}

func fileUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("file ownership metadata is unavailable")
	}
	return stat.Uid, nil
}

func validateStagedTrustCertificate(path, fingerprint string) error {
	body, err := os.ReadFile(path) // #nosec G304 -- path is built from a validated basename under the staging dir.
	if err != nil {
		return fmt.Errorf("reading staged trust certificate: %w", err)
	}
	if len(body) > 16*1024 {
		return errors.New("staged trust certificate is too large")
	}
	block, rest := pem.Decode(body)
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("staged trust certificate is not a PEM certificate")
	}
	if len(bytesTrimSpace(rest)) != 0 {
		return errors.New("staged trust certificate contains trailing data")
	}
	sum := sha256.Sum256(block.Bytes)
	if hex.EncodeToString(sum[:]) != fingerprint {
		return errors.New("staged trust certificate fingerprint does not match fingerprint argument")
	}
	return nil
}

func bytesTrimSpace(in []byte) []byte {
	for len(in) > 0 && (in[0] == ' ' || in[0] == '\n' || in[0] == '\r' || in[0] == '\t') {
		in = in[1:]
	}
	for len(in) > 0 {
		last := in[len(in)-1]
		if last != ' ' && last != '\n' && last != '\r' && last != '\t' {
			break
		}
		in = in[:len(in)-1]
	}
	return in
}
