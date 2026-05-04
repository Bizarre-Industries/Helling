package store

// Users CRUD. Sessions live in sessions.go.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User mirrors the users table.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

// ErrNotFound is returned when a lookup matches no rows.
var ErrNotFound = errors.New("store: not found")

// ErrUsersExist is returned when first-admin bootstrap is attempted after any
// user already exists.
var ErrUsersExist = errors.New("store: users already exist")

// CreateUser inserts a new user and returns the persisted row.
// Caller is responsible for hashing the password (use internal/auth.Hash).
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, isAdmin bool) (User, error) {
	now := time.Now().Unix()
	admin := 0
	if isAdmin {
		admin = 1
	}
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (username, password_hash, created_at, is_admin) VALUES (?, ?, ?, ?)`,
		username, passwordHash, now, admin,
	)
	if err != nil {
		return User{}, fmt.Errorf("inserting user %q: %w", username, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("getting user id: %w", err)
	}
	return User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		IsAdmin:      isAdmin,
		CreatedAt:    time.Unix(now, 0).UTC(),
	}, nil
}

// CreateFirstAdmin inserts the first user as an admin only when the users table
// is still empty. The emptiness check and insert share one SQLite transaction so
// concurrent first-boot setup requests cannot create multiple bootstrap admins.
func (s *Store) CreateFirstAdmin(ctx context.Context, username, passwordHash string) (User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin first-admin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return User{}, fmt.Errorf("counting users for first admin: %w", err)
	}
	if n > 0 {
		return User{}, ErrUsersExist
	}

	now := time.Now().Unix()
	res, err := tx.ExecContext(
		ctx,
		`INSERT INTO users (username, password_hash, created_at, is_admin) VALUES (?, ?, ?, 1)`,
		username, passwordHash, now,
	)
	if err != nil {
		return User{}, fmt.Errorf("inserting first admin %q: %w", username, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("getting first admin id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit first-admin tx: %w", err)
	}

	return User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		IsAdmin:      true,
		CreatedAt:    time.Unix(now, 0).UTC(),
	}, nil
}

// GetUserByUsername returns the user with the given username, or ErrNotFound.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	var u User
	var createdAt int64
	var isAdmin int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, password_hash, created_at, is_admin FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt, &isAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("loading user %q: %w", username, err)
	}
	u.CreatedAt = time.Unix(createdAt, 0).UTC()
	u.IsAdmin = isAdmin != 0
	return u, nil
}

// GetUserByID returns the user with the given id, or ErrNotFound.
func (s *Store) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	var createdAt int64
	var isAdmin int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, password_hash, created_at, is_admin FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt, &isAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("loading user id %d: %w", id, err)
	}
	u.CreatedAt = time.Unix(createdAt, 0).UTC()
	u.IsAdmin = isAdmin != 0
	return u, nil
}

// CountUsers returns the total number of users. Used for first-boot bootstrap.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return n, nil
}

// ListUsers returns all users ordered by creation time.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, created_at, is_admin FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []User
	for rows.Next() {
		var u User
		var createdAt int64
		var isAdmin int
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &createdAt, &isAdmin); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		u.CreatedAt = time.Unix(createdAt, 0).UTC()
		u.IsAdmin = isAdmin != 0
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateUser updates the password hash and admin flag for a user.
func (s *Store) UpdateUser(ctx context.Context, id int64, passwordHash string, isAdmin bool) error {
	admin := 0
	if isAdmin {
		admin = 1
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET password_hash = ?, is_admin = ? WHERE id = ?`,
		passwordHash, admin, id,
	)
	if err != nil {
		return fmt.Errorf("updating user %d: %w", id, err)
	}
	return nil
}

// DeleteUser removes a user and cascades to sessions, tokens, etc.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting user %d: %w", id, err)
	}
	return nil
}
