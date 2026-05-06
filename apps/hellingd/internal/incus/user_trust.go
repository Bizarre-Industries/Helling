package incus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// DefaultTrustHelperPath is the setuid helper used for Incus trust mutation.
const DefaultTrustHelperPath = "/usr/lib/helling/helling-incus-trust"

// UserTrustService issues, stores, registers, and revokes per-user Incus certs.
type UserTrustService struct {
	Store      *store.Store
	CA         *CertificateAuthority
	StagingDir string
	RequestDir string
	HelperPath string
	RunHelper  func(ctx context.Context, path string, args ...string) error
}

type userTrustMaterial struct {
	certPEM     []byte
	keyPEM      []byte
	fingerprint string
	expiresAt   time.Time
	certName    string
	requestName string
}

// Provision replaces a user's Incus trust entry with a project-restricted certificate.
func (s UserTrustService) Provision(ctx context.Context, user *store.User) error {
	if err := s.validateProvisionRequest(user); err != nil {
		return err
	}
	material, err := s.prepareUserTrustMaterial(user)
	if err != nil {
		return err
	}
	previous, hasPrevious, err := s.currentTrust(ctx, user.ID)
	if err != nil {
		return err
	}
	if err := s.runHelper(ctx, "register", material.fingerprint, user.IncusProject, material.certName, material.requestName); err != nil {
		return fmt.Errorf("registering Incus trust certificate: %w", err)
	}
	if err := s.Store.UpsertIncusUserCertAndProject(ctx, user.ID, string(material.certPEM), string(material.keyPEM), material.fingerprint, user.IncusProject, material.expiresAt); err != nil {
		if revokeErr := s.revokeFingerprint(ctx, user.ID, material.fingerprint, user.IncusProject); revokeErr != nil {
			return fmt.Errorf("storing Incus trust certificate and revoking new trust %s: %w", material.fingerprint, errors.Join(err, revokeErr))
		}
		return fmt.Errorf("storing Incus trust certificate: %w", err)
	}
	if hasPrevious && previous.Fingerprint != material.fingerprint {
		if err := s.revokeFingerprint(ctx, user.ID, previous.Fingerprint, previous.ProjectScope); err != nil {
			restoreErr := s.Store.RestoreIncusUserCertAndProject(ctx, &previous)
			revokeNewErr := s.revokeFingerprint(ctx, user.ID, material.fingerprint, user.IncusProject)
			return fmt.Errorf("revoking previous Incus trust %s: %w", previous.Fingerprint, errors.Join(err, restoreErr, revokeNewErr))
		}
	}
	return nil
}

func (s UserTrustService) currentTrust(ctx context.Context, userID int64) (store.IncusUserCert, bool, error) {
	old, err := s.Store.GetIncusUserCert(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return store.IncusUserCert{}, false, nil
	}
	if err != nil {
		return store.IncusUserCert{}, false, err
	}
	if old.RevokedAt != nil || old.Fingerprint == "" {
		return store.IncusUserCert{}, false, nil
	}
	return old, true, nil
}

func (s UserTrustService) validateProvisionRequest(user *store.User) error {
	if s.Store == nil {
		return errors.New("store is required")
	}
	if user == nil {
		return errors.New("user is required")
	}
	if user.IncusProject == "" {
		user.IncusProject = "default"
	}
	if !trustProjectPattern.MatchString(user.IncusProject) {
		return fmt.Errorf("invalid Incus project %q", user.IncusProject)
	}
	return nil
}

func (s UserTrustService) prepareUserTrustMaterial(user *store.User) (userTrustMaterial, error) {
	issuedAt := time.Now().UTC()
	ca, err := s.certificateAuthority(issuedAt)
	if err != nil {
		return userTrustMaterial{}, err
	}
	certPEM, keyPEM, fingerprint, expiresAt, err := IssueUserCertificateWithCA(user.ID, user.Username, issuedAt, ca)
	if err != nil {
		return userTrustMaterial{}, err
	}
	certName := fmt.Sprintf("helling-user-%d-%s.crt", user.ID, fingerprint)
	if err := os.MkdirAll(s.stagingDir(), 0o750); err != nil {
		return userTrustMaterial{}, fmt.Errorf("creating Incus trust staging dir: %w", err)
	}
	certPath := filepath.Join(s.stagingDir(), certName)
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return userTrustMaterial{}, fmt.Errorf("writing Incus trust certificate: %w", err)
	}
	requestName, err := s.writeTrustRequest("register", user.ID, fingerprint, user.IncusProject, certName)
	if err != nil {
		return userTrustMaterial{}, err
	}
	return userTrustMaterial{
		certPEM:     certPEM,
		keyPEM:      keyPEM,
		fingerprint: fingerprint,
		expiresAt:   expiresAt,
		certName:    certName,
		requestName: requestName,
	}, nil
}

// Revoke removes the user's current Incus trust entry when one exists.
func (s UserTrustService) Revoke(ctx context.Context, user *store.User) error {
	if s.Store == nil {
		return errors.New("store is required")
	}
	if user == nil {
		return errors.New("user is required")
	}
	row, err := s.Store.GetIncusUserCert(ctx, user.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	if row.RevokedAt != nil || row.Fingerprint == "" {
		return nil
	}
	if err := s.revokeFingerprint(ctx, user.ID, row.Fingerprint, row.ProjectScope); err != nil {
		return err
	}
	return s.Store.RevokeIncusUserCert(ctx, user.ID)
}

func (s UserTrustService) revokeFingerprint(ctx context.Context, userID int64, fingerprint, project string) error {
	requestName, err := s.writeTrustRequest("revoke", userID, fingerprint, project, "")
	if err != nil {
		return err
	}
	if err := s.runHelper(ctx, "revoke", fingerprint, requestName); err != nil {
		return fmt.Errorf("revoking Incus trust certificate: %w", err)
	}
	return nil
}

func (s UserTrustService) stagingDir() string {
	if s.StagingDir != "" {
		return s.StagingDir
	}
	return DefaultTrustStagingDir
}

func (s UserTrustService) helperPath() string {
	if s.HelperPath != "" {
		return s.HelperPath
	}
	return DefaultTrustHelperPath
}

func (s UserTrustService) requestDir() string {
	if s.RequestDir != "" {
		return s.RequestDir
	}
	return DefaultTrustRequestDir
}

func (s UserTrustService) certificateAuthority(generatedAt time.Time) (*CertificateAuthority, error) {
	if s.CA != nil {
		return s.CA, nil
	}
	_ = generatedAt
	return nil, errors.New("persistent Helling CA is required")
}

func (s UserTrustService) runHelper(ctx context.Context, args ...string) error {
	if s.RunHelper != nil {
		return s.RunHelper(ctx, s.helperPath(), args...)
	}
	// #nosec G204 -- helper path is fixed by default or operator-configured; args are generated by this service.
	return exec.CommandContext(ctx, s.helperPath(), args...).Run()
}

func (s UserTrustService) writeTrustRequest(action string, userID int64, fingerprint, project, certName string) (string, error) {
	if userID <= 0 {
		return "", errors.New("user id is required")
	}
	if err := validateTrustFingerprint(fingerprint); err != nil {
		return "", err
	}
	if !trustProjectPattern.MatchString(project) {
		return "", fmt.Errorf("invalid Incus project %q", project)
	}
	token, err := trustRequestToken()
	if err != nil {
		return "", err
	}
	requestName := fmt.Sprintf("helling-trust-%s-%d-%s-%s.json", action, userID, fingerprint, token)
	request := trustRequestManifest{
		Action:       action,
		UserID:       userID,
		Fingerprint:  fingerprint,
		Project:      project,
		CertBasename: certName,
	}
	if err := os.MkdirAll(s.requestDir(), 0o700); err != nil {
		return "", fmt.Errorf("creating Incus trust request dir: %w", err)
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encoding Incus trust request: %w", err)
	}
	requestPath := filepath.Join(s.requestDir(), requestName)
	file, err := os.OpenFile(requestPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- requestName is generated from validated fields and a random token under requestDir.
	if err != nil {
		return "", fmt.Errorf("creating Incus trust request: %w", err)
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("writing Incus trust request: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("closing Incus trust request: %w", err)
	}
	return requestName, nil
}

func trustRequestToken() (string, error) {
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", fmt.Errorf("generating Incus trust request token: %w", err)
	}
	return hex.EncodeToString(token[:]), nil
}
