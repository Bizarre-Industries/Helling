// Package authrepo provides SQLite-backed persistence for hellingd identity
// tables: users, sessions, auth_events. See docs/spec/sqlite-schema.md.
package authrepo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a lookup yields zero rows.
var ErrNotFound = errors.New("authrepo: not found")

// ErrDuplicate is returned when an insert violates uniqueness (e.g. username).
var ErrDuplicate = errors.New("authrepo: duplicate")

// User mirrors the users table row shape used by hellingd.
type User struct {
	ID           string
	Username     string
	Role         string
	Status       string
	PasswordHash sql.NullString
	CreatedAt    int64
	UpdatedAt    int64
}

// Session mirrors the sessions table row.
type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	UserAgent        sql.NullString
	IPAddress        sql.NullString
	ExpiresAt        int64
	RevokedAt        sql.NullInt64
	CreatedAt        int64
}

// Repo wraps *sql.DB with auth-oriented queries.
type Repo struct {
	db  *sql.DB
	now func() time.Time
}

// New returns a Repo on the given pool using time.Now for timestamps.
func New(db *sql.DB) *Repo { return &Repo{db: db, now: time.Now} }

// SetClock lets tests inject a deterministic clock.
func (r *Repo) SetClock(now func() time.Time) { r.now = now }

// ── Users ──────────────────────────────────────────────────────────────────

// CountAdmins returns the number of users whose role=admin and status=active.
// Used by authSetup to detect first-boot state.
func (r *Repo) CountAdmins(ctx context.Context) (int, error) {
	const q = `SELECT count(*) FROM users WHERE role = 'admin' AND status = 'active'`
	var n int
	if err := r.db.QueryRowContext(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("authrepo: count admins: %w", err)
	}
	return n, nil
}

// CreateUser inserts a new user row. passwordHash may be empty for PAM-only users.
func (r *Repo) CreateUser(ctx context.Context, username, role, passwordHash string) (User, error) {
	now := r.now().Unix()
	id := uuid.NewString()
	const q = `INSERT INTO users (id, username, role, status, password_hash, created_at, updated_at)
	           VALUES (?, ?, ?, 'active', NULLIF(?, ''), ?, ?)`
	if _, err := r.db.ExecContext(ctx, q, id, username, role, passwordHash, now, now); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrDuplicate
		}
		return User{}, fmt.Errorf("authrepo: insert user: %w", err)
	}
	u := User{
		ID: id, Username: username, Role: role, Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if passwordHash != "" {
		u.PasswordHash = sql.NullString{String: passwordHash, Valid: true}
	}
	return u, nil
}

// GetUserByUsername returns the user row for a username or ErrNotFound.
func (r *Repo) GetUserByUsername(ctx context.Context, username string) (User, error) {
	const q = `SELECT id, username, role, status, password_hash, created_at, updated_at
	           FROM users WHERE username = ? LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, username)
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("authrepo: get user: %w", err)
	}
	return u, nil
}

// GetUserByID returns the user row for a user id or ErrNotFound.
func (r *Repo) GetUserByID(ctx context.Context, id string) (User, error) {
	const q = `SELECT id, username, role, status, password_hash, created_at, updated_at
	           FROM users WHERE id = ? LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, id)
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("authrepo: get user by id: %w", err)
	}
	return u, nil
}

// DeleteUser removes a user row. ON DELETE CASCADE in the schema cleans up
// sessions, api_tokens, totp_secrets, and recovery_codes (see
// docs/spec/sqlite-schema.md §7).
func (r *Repo) DeleteUser(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return fmt.Errorf("authrepo: delete user: %w", err)
	}
	return nil
}

// SetUserScope writes the Incus trust-scope hint. Column provisioned in
// migration 0003_user_scope.sql.
func (r *Repo) SetUserScope(ctx context.Context, id, scope string) error {
	const q = `UPDATE users SET scope = ?, updated_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, scope, r.now().Unix(), id); err != nil {
		return fmt.Errorf("authrepo: set user scope: %w", err)
	}
	return nil
}

// GetSystemConfig fetches a runtime config value, or ErrNotFound.
func (r *Repo) GetSystemConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM system_config WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("authrepo: get system config: %w", err)
	}
	return value, nil
}

// PutSystemConfig upserts a runtime config value. updatedBy may be empty for
// boot-time writes.
func (r *Repo) PutSystemConfig(ctx context.Context, key, value, updatedBy string) error {
	const q = `INSERT INTO system_config (key, value, updated_at, updated_by)
	           VALUES (?, ?, ?, NULLIF(?, ''))
	           ON CONFLICT(key) DO UPDATE SET
	             value      = excluded.value,
	             updated_at = excluded.updated_at,
	             updated_by = excluded.updated_by`
	if _, err := r.db.ExecContext(ctx, q, key, value, r.now().Unix(), updatedBy); err != nil {
		return fmt.Errorf("authrepo: put system config: %w", err)
	}
	return nil
}

// ListUsers returns users ordered by created_at ASC starting at the given
// offset, up to limit rows. Used by userList API stub replacement.
func (r *Repo) ListUsers(ctx context.Context, offset, limit int) ([]User, int, error) {
	const countQ = `SELECT count(*) FROM users`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("authrepo: count users: %w", err)
	}

	const q = `SELECT id, username, role, status, password_hash, created_at, updated_at
	           FROM users ORDER BY created_at ASC, id ASC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("authrepo: list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Status, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("authrepo: scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("authrepo: rows err: %w", err)
	}
	return out, total, nil
}

// ── Sessions ───────────────────────────────────────────────────────────────

// CreateSession persists a new session with the SHA-256 hash of the refresh
// token. The raw token is never stored server-side.
func (r *Repo) CreateSession(ctx context.Context, userID, refreshToken string, expiresAt time.Time, userAgent, ip string) (Session, error) {
	hash := Sha256Hex(refreshToken)
	now := r.now().Unix()
	id := uuid.NewString()
	const q = `INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at)
	           VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)`
	if _, err := r.db.ExecContext(ctx, q, id, userID, hash, userAgent, ip, expiresAt.Unix(), now); err != nil {
		return Session{}, fmt.Errorf("authrepo: insert session: %w", err)
	}
	return Session{
		ID: id, UserID: userID, RefreshTokenHash: hash,
		ExpiresAt: expiresAt.Unix(), CreatedAt: now,
	}, nil
}

// GetActiveSessionByRefresh returns the active session matching the hashed
// refresh token. An active session has revoked_at IS NULL and expires_at in
// the future.
func (r *Repo) GetActiveSessionByRefresh(ctx context.Context, refreshToken string) (Session, error) {
	hash := Sha256Hex(refreshToken)
	now := r.now().Unix()
	const q = `SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at
	           FROM sessions
	           WHERE refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?
	           LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, hash, now)
	var s Session
	err := row.Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("authrepo: get session: %w", err)
	}
	return s, nil
}

// RevokeSession marks a session revoked by its id.
func (r *Repo) RevokeSession(ctx context.Context, id string) error {
	const q = `UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, r.now().Unix(), id); err != nil {
		return fmt.Errorf("authrepo: revoke session: %w", err)
	}
	return nil
}

// RevokeSessionByRefresh revokes the session identified by the given raw
// refresh token. No-op if no matching active session exists.
func (r *Repo) RevokeSessionByRefresh(ctx context.Context, refreshToken string) error {
	hash := Sha256Hex(refreshToken)
	const q = `UPDATE sessions SET revoked_at = ? WHERE refresh_token_hash = ? AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, r.now().Unix(), hash); err != nil {
		return fmt.Errorf("authrepo: revoke session by refresh: %w", err)
	}
	return nil
}

// ── Auth events ────────────────────────────────────────────────────────────

// RecordEvent writes a row to auth_events. userID may be empty for failed
// pre-auth events; metadataJSON may be empty.
func (r *Repo) RecordEvent(ctx context.Context, userID, eventType, sourceIP, userAgent, metadataJSON string) error {
	const q = `INSERT INTO auth_events (id, user_id, event_type, source_ip, user_agent, metadata_json, created_at)
	           VALUES (?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?)`
	_, err := r.db.ExecContext(ctx, q,
		uuid.NewString(), userID, eventType, sourceIP, userAgent, metadataJSON, r.now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("authrepo: record event: %w", err)
	}
	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────

// Sha256Hex returns the lowercase hex SHA-256 digest of s.
func Sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
}
