package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// mfaPendingTTL bounds how long the server holds an MFA challenge before the
// user must log in again.
const mfaPendingTTL = 5 * time.Minute

// apiTokenDefaultTTL matches docs/spec/auth.md §5 default 90-day expiry.
const apiTokenDefaultTTL = 90 * 24 * time.Hour

// apiTokenMaxTTL caps operator requests at 365 days per docs/spec/auth.md §5.
const apiTokenMaxTTL = 365 * 24 * time.Hour

// mfaPendingEntry holds state for an outstanding MFA challenge.
type mfaPendingEntry struct {
	userID    string
	expiresAt time.Time
}

// mfaPendingStore is a tiny in-process map with TTL expiry. Fine for v0.1:
// a single hellingd process holds session state already (refresh tokens live
// in SQLite; access tokens are JWTs). Horizontal scale-out moves MFA state
// to SQLite later.
type mfaPendingStore struct {
	mu  sync.Mutex
	set map[string]mfaPendingEntry
	now func() time.Time
}

func newMfaPendingStore() mfaPendingStore {
	return mfaPendingStore{
		set: map[string]mfaPendingEntry{},
		now: time.Now,
	}
}

func (m *mfaPendingStore) put(token, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.set[token] = mfaPendingEntry{userID: userID, expiresAt: m.now().Add(mfaPendingTTL)}
}

// takeAndConsume atomically removes and returns the pending entry. Returns
// ok=false when not found or already expired.
func (m *mfaPendingStore) takeAndConsume(token string) (userID string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, present := m.set[token]
	if !present {
		return "", false
	}
	delete(m.set, token)
	if m.now().After(entry.expiresAt) {
		return "", false
	}
	return entry.userID, true
}

func randomMfaToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mfa_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// ── Login with MFA branch ─────────────────────────────────────────────────

// PendingMFA describes a login that has passed password verification but
// must complete a TOTP / recovery code challenge.
type PendingMFA struct {
	MFAToken         string
	ExpiresInSeconds int
}

// ErrMFARequired is returned by Login when the user has an active TOTP
// factor. Callers must then call CompleteMFA.
var ErrMFARequired = errors.New("auth: mfa challenge required")

// LoginResult carries the outcome of a login attempt: either a fully-issued
// Identity or a Pending MFA challenge.
type LoginResult struct {
	Identity *Identity
	Pending  *PendingMFA
}

// LoginWithMFA is Login + TOTP gate. When the user has an active TOTP row,
// the caller receives a Pending challenge instead of session tokens.
func (s *Service) LoginWithMFA(ctx context.Context, username, password, ip, userAgent string) (LoginResult, error) {
	if username == "" || password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}

	u, err := s.repo.GetUserByUsername(ctx, username)
	if errors.Is(err, authrepo.ErrNotFound) {
		_ = s.repo.RecordEvent(ctx, "", "auth.login_fail", ip, userAgent, `{"reason":"unknown_user"}`)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	if u.Status != userStatusActive {
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_fail", ip, userAgent, `{"reason":"disabled"}`)
		return LoginResult{}, ErrUserDisabled
	}
	if !u.PasswordHash.Valid {
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_fail", ip, userAgent, `{"reason":"no_local_hash"}`)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := VerifyPassword(password, u.PasswordHash.String); err != nil {
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_fail", ip, userAgent, `{"reason":"bad_password"}`)
		return LoginResult{}, ErrInvalidCredentials
	}

	totp, err := s.repo.GetTOTPSecret(ctx, u.ID)
	switch {
	case errors.Is(err, authrepo.ErrNotFound):
		// No TOTP; issue session immediately.
	case err != nil:
		return LoginResult{}, err
	case totp.Enabled:
		token, err := randomMfaToken()
		if err != nil {
			return LoginResult{}, fmt.Errorf("auth: mint mfa token: %w", err)
		}
		s.mfaPending.put(token, u.ID)
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.mfa_challenge", ip, userAgent, "")
		return LoginResult{Pending: &PendingMFA{
			MFAToken:         token,
			ExpiresInSeconds: int(mfaPendingTTL.Seconds()),
		}}, nil
	}

	ident, err := s.issueSession(ctx, &u, ip, userAgent)
	if err != nil {
		return LoginResult{}, err
	}
	_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_ok", ip, userAgent, "")
	return LoginResult{Identity: &ident}, nil
}

// CompleteMFA finishes a TOTP-challenged login. Consumes the MFA token.
func (s *Service) CompleteMFA(ctx context.Context, mfaToken, code, ip, userAgent string) (Identity, error) {
	if mfaToken == "" || code == "" {
		return Identity{}, ErrInvalidCredentials
	}
	userID, ok := s.mfaPending.takeAndConsume(mfaToken)
	if !ok {
		return Identity{}, ErrInvalidCredentials
	}

	secretRow, err := s.repo.GetTOTPSecret(ctx, userID)
	if err != nil {
		return Identity{}, err
	}
	// v0.1: encrypted_secret holds the plaintext base32 secret until age
	// encryption lands (ADR-039).
	secret := string(secretRow.EncryptedSecret)

	if err := VerifyTOTP(secret, code); err != nil {
		// Recovery-code fallback.
		codes, listErr := s.repo.ListUnusedRecoveryCodes(ctx, userID)
		if listErr != nil {
			return Identity{}, listErr
		}
		hashes := make([]string, 0, len(codes))
		for _, c := range codes {
			hashes = append(hashes, c.CodeHash)
		}
		idx, matchErr := MatchRecoveryCode(code, hashes)
		if matchErr != nil {
			_ = s.repo.RecordEvent(ctx, userID, "auth.mfa_fail", ip, userAgent, "")
			return Identity{}, ErrInvalidCredentials
		}
		if err := s.repo.MarkRecoveryCodeUsed(ctx, codes[idx].ID); err != nil {
			return Identity{}, err
		}
		_ = s.repo.RecordEvent(ctx, userID, "auth.mfa_recovery_used", ip, userAgent, "")
	}

	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return Identity{}, err
	}
	ident, err := s.issueSession(ctx, &u, ip, userAgent)
	if err != nil {
		return Identity{}, err
	}
	_ = s.repo.RecordEvent(ctx, userID, "auth.login_ok", ip, userAgent, "")
	return ident, nil
}

// ── TOTP enrollment lifecycle ─────────────────────────────────────────────

// TOTPEnrollmentResult carries enrollment output for the API layer.
type TOTPEnrollmentResult struct {
	Secret          string
	ProvisioningURI string
	RecoveryCodes   []string
}

// EnrollTOTP starts a new TOTP enrollment for the user: generates secret +
// recovery codes, stores them disabled until VerifyTOTPEnroll.
func (s *Service) EnrollTOTP(ctx context.Context, userID string) (TOTPEnrollmentResult, error) {
	if userID == "" {
		return TOTPEnrollmentResult{}, errors.New("auth: userID required")
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return TOTPEnrollmentResult{}, err
	}
	enr, err := EnrollTOTP(u.Username)
	if err != nil {
		return TOTPEnrollmentResult{}, err
	}

	codes, err := GenerateRecoveryCodes()
	if err != nil {
		return TOTPEnrollmentResult{}, err
	}
	hashes, err := HashRecoveryCodes(codes)
	if err != nil {
		return TOTPEnrollmentResult{}, err
	}
	// Clear prior enrollment for idempotency, then seat new secret + codes.
	if err := s.repo.DeleteTOTPSecret(ctx, userID); err != nil {
		return TOTPEnrollmentResult{}, err
	}
	if err := s.repo.UpsertTOTPSecret(ctx, userID, []byte(enr.Secret), false); err != nil {
		return TOTPEnrollmentResult{}, err
	}
	if err := s.repo.InsertRecoveryCodes(ctx, userID, hashes); err != nil {
		return TOTPEnrollmentResult{}, err
	}

	_ = s.repo.RecordEvent(ctx, userID, "auth.totp_enroll", "", "", "")

	return TOTPEnrollmentResult{
		Secret:          enr.Secret,
		ProvisioningURI: enr.ProvisioningURI,
		RecoveryCodes:   codes,
	}, nil
}

// VerifyTOTPEnroll confirms enrollment; on success marks enabled=1.
func (s *Service) VerifyTOTPEnroll(ctx context.Context, userID, totpCode string) error {
	row, err := s.repo.GetTOTPSecret(ctx, userID)
	if err != nil {
		return err
	}
	if err := VerifyTOTP(string(row.EncryptedSecret), totpCode); err != nil {
		return err
	}
	if err := s.repo.SetTOTPEnabled(ctx, userID, true); err != nil {
		return err
	}
	_ = s.repo.RecordEvent(ctx, userID, "auth.totp_verify", "", "", "")
	return nil
}

// DisableTOTP removes the user's TOTP factor and recovery codes.
func (s *Service) DisableTOTP(ctx context.Context, userID string) error {
	if err := s.repo.DeleteTOTPSecret(ctx, userID); err != nil {
		return err
	}
	_ = s.repo.RecordEvent(ctx, userID, "auth.totp_disable", "", "", "")
	return nil
}

// ── API tokens ────────────────────────────────────────────────────────────

// APITokenIssued carries the plaintext token once.
type APITokenIssued struct {
	ID        string
	Name      string
	Scope     string
	Plaintext string
	ExpiresAt int64
	CreatedAt int64
}

// CreateAPIToken mints a new helling_ token for a user with the given scope.
// Returns the plaintext token exactly once; the store persists only the hash.
func (s *Service) CreateAPIToken(ctx context.Context, userID, name, scope string, ttl time.Duration) (APITokenIssued, error) {
	if userID == "" || name == "" {
		return APITokenIssued{}, errors.New("auth: userID/name required")
	}
	if !IsValidAPITokenScope(scope) {
		return APITokenIssued{}, ErrInvalidScope
	}
	if ttl <= 0 {
		ttl = apiTokenDefaultTTL
	}
	if ttl > apiTokenMaxTTL {
		ttl = apiTokenMaxTTL
	}

	plain, hash, err := GenerateAPIToken()
	if err != nil {
		return APITokenIssued{}, fmt.Errorf("auth: generate api token: %w", err)
	}
	expires := time.Now().Add(ttl)
	row, err := s.repo.CreateAPIToken(ctx, userID, name, hash, scope, expires)
	if err != nil {
		return APITokenIssued{}, err
	}
	_ = s.repo.RecordEvent(ctx, userID, "auth.token_create", "", "", `{"scope":"`+scope+`"}`)
	return APITokenIssued{
		ID:        row.ID,
		Name:      row.Name,
		Scope:     row.Scope,
		Plaintext: plain,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

// ListAPITokens returns a user's tokens.
func (s *Service) ListAPITokens(ctx context.Context, userID string) ([]authrepo.APIToken, error) {
	return s.repo.ListAPITokens(ctx, userID)
}

// RevokeAPIToken marks a token revoked, scoped to its owner.
func (s *Service) RevokeAPIToken(ctx context.Context, userID, tokenID string) error {
	if err := s.repo.RevokeAPIToken(ctx, userID, tokenID); err != nil {
		return err
	}
	_ = s.repo.RecordEvent(ctx, userID, "auth.token_revoke", "", "", `{"id":"`+tokenID+`"}`)
	return nil
}

// VerifyAPIToken resolves a plaintext Bearer token to its owning user.
// Updates last_used_at on success.
func (s *Service) VerifyAPIToken(ctx context.Context, plaintext string) (authrepo.User, authrepo.APIToken, error) {
	hash := HashAPIToken(plaintext)
	tok, err := s.repo.GetAPITokenByHash(ctx, hash)
	if err != nil {
		return authrepo.User{}, authrepo.APIToken{}, err
	}
	u, err := s.repo.GetUserByID(ctx, tok.UserID)
	if err != nil {
		return authrepo.User{}, authrepo.APIToken{}, err
	}
	if u.Status != userStatusActive {
		return authrepo.User{}, authrepo.APIToken{}, ErrUserDisabled
	}
	_ = s.repo.TouchAPITokenLastUsed(ctx, tok.ID)
	return u, tok, nil
}
