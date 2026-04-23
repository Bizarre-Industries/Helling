package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// Service ties password hashing, JWT signing, and persistence into the
// high-level auth operations used by the API layer.
type Service struct {
	repo   *authrepo.Repo
	signer *Signer
	hashP  Argon2idParams
}

// NewService wires a Service. params default to DefaultArgon2idParams when zero.
func NewService(repo *authrepo.Repo, signer *Signer, params Argon2idParams) *Service {
	if params == (Argon2idParams{}) {
		params = DefaultArgon2idParams
	}
	return &Service{repo: repo, signer: signer, hashP: params}
}

// Identity identifies the caller and carries the issued tokens.
type Identity struct {
	UserID         string
	Username       string
	Role           string
	AccessToken    string
	AccessExpires  int
	RefreshToken   string
	RefreshExpires int64 // Unix seconds
}

// SetupRequired is true when no active admin exists yet.
func (s *Service) SetupRequired(ctx context.Context) (bool, error) {
	n, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// ErrSetupNotRequired is returned when authSetup is called but an admin
// already exists.
var ErrSetupNotRequired = errors.New("auth: setup already completed")

// ErrInvalidCredentials is the single 401 error emitted by login.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrUserDisabled is returned when a disabled user attempts to log in.
var ErrUserDisabled = errors.New("auth: user disabled")

// Setup creates the initial admin account and issues a fresh token pair.
// Fails with ErrSetupNotRequired when an admin already exists.
func (s *Service) Setup(ctx context.Context, username, password, ip, userAgent string) (Identity, error) {
	if username == "" || password == "" {
		return Identity{}, errors.New("auth: username and password required")
	}
	required, err := s.SetupRequired(ctx)
	if err != nil {
		return Identity{}, err
	}
	if !required {
		return Identity{}, ErrSetupNotRequired
	}

	hash, err := HashPassword(password, s.hashP)
	if err != nil {
		return Identity{}, fmt.Errorf("auth: hash password: %w", err)
	}
	u, err := s.repo.CreateUser(ctx, username, "admin", hash)
	if err != nil {
		return Identity{}, fmt.Errorf("auth: create user: %w", err)
	}

	ident, err := s.issueSession(ctx, &u, ip, userAgent)
	if err != nil {
		return Identity{}, err
	}
	_ = s.repo.RecordEvent(ctx, u.ID, "auth.setup", ip, userAgent, "")
	return ident, nil
}

// Login verifies the password against a Helling-managed argon2id hash and
// issues a fresh token pair. Returns ErrInvalidCredentials for unknown users
// and password mismatches alike to avoid user enumeration.
func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (Identity, error) {
	if username == "" || password == "" {
		return Identity{}, ErrInvalidCredentials
	}

	u, err := s.repo.GetUserByUsername(ctx, username)
	if errors.Is(err, authrepo.ErrNotFound) {
		_ = s.repo.RecordEvent(ctx, "", "auth.login_fail", ip, userAgent, `{"reason":"unknown_user"}`)
		return Identity{}, ErrInvalidCredentials
	}
	if err != nil {
		return Identity{}, err
	}
	if u.Status != "active" {
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_fail", ip, userAgent, `{"reason":"disabled"}`)
		return Identity{}, ErrUserDisabled
	}
	if !u.PasswordHash.Valid {
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_fail", ip, userAgent, `{"reason":"no_local_hash"}`)
		return Identity{}, ErrInvalidCredentials
	}
	if err := VerifyPassword(password, u.PasswordHash.String); err != nil {
		_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_fail", ip, userAgent, `{"reason":"bad_password"}`)
		return Identity{}, ErrInvalidCredentials
	}

	ident, err := s.issueSession(ctx, &u, ip, userAgent)
	if err != nil {
		return Identity{}, err
	}
	_ = s.repo.RecordEvent(ctx, u.ID, "auth.login_ok", ip, userAgent, "")
	return ident, nil
}

// Refresh exchanges a valid refresh token for a new access token and a
// rotated refresh token. The old refresh token is revoked.
func (s *Service) Refresh(ctx context.Context, refreshToken, ip, userAgent string) (Identity, error) {
	if refreshToken == "" {
		return Identity{}, ErrInvalidCredentials
	}
	sess, err := s.repo.GetActiveSessionByRefresh(ctx, refreshToken)
	if errors.Is(err, authrepo.ErrNotFound) {
		return Identity{}, ErrInvalidCredentials
	}
	if err != nil {
		return Identity{}, err
	}
	u, err := s.repo.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return Identity{}, err
	}
	if u.Status != "active" {
		_ = s.repo.RevokeSession(ctx, sess.ID)
		return Identity{}, ErrUserDisabled
	}

	if err := s.repo.RevokeSession(ctx, sess.ID); err != nil {
		return Identity{}, err
	}
	ident, err := s.issueSession(ctx, &u, ip, userAgent)
	if err != nil {
		return Identity{}, err
	}
	_ = s.repo.RecordEvent(ctx, u.ID, "auth.refresh", ip, userAgent, "")
	return ident, nil
}

// Logout revokes the refresh-token session. Always returns nil for unknown
// tokens so that callers cannot probe token validity via logout.
func (s *Service) Logout(ctx context.Context, refreshToken, ip, userAgent string) error {
	if refreshToken == "" {
		return nil
	}
	_ = s.repo.RevokeSessionByRefresh(ctx, refreshToken)
	_ = s.repo.RecordEvent(ctx, "", "auth.logout", ip, userAgent, "")
	return nil
}

// Signer exposes the underlying JWT signer so API middleware can share it.
func (s *Service) Signer() *Signer { return s.signer }

// Repo exposes the underlying auth repo so callers can query users.
func (s *Service) Repo() *authrepo.Repo { return s.repo }

func (s *Service) issueSession(ctx context.Context, u *authrepo.User, ip, userAgent string) (Identity, error) {
	access, ttl, err := s.signer.IssueAccess(u.ID, u.Username, u.Role)
	if err != nil {
		return Identity{}, fmt.Errorf("auth: issue access: %w", err)
	}
	refresh, refreshExp, err := s.signer.IssueRefresh()
	if err != nil {
		return Identity{}, fmt.Errorf("auth: issue refresh: %w", err)
	}
	if _, err := s.repo.CreateSession(ctx, u.ID, refresh, refreshExp, userAgent, ip); err != nil {
		return Identity{}, fmt.Errorf("auth: create session: %w", err)
	}
	return Identity{
		UserID:         u.ID,
		Username:       u.Username,
		Role:           u.Role,
		AccessToken:    access,
		AccessExpires:  ttl,
		RefreshToken:   refresh,
		RefreshExpires: refreshExp.Unix(),
	}, nil
}
