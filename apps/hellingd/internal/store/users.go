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

// CreateUser inserts a new user and returns the persisted row.
// Caller is responsible for hashing the password (use internal/auth.Hash).
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, isAdmin bool) (User, error) {
	now := time.Now().Unix()
	admin := 0
	if isAdmin {
		admin = 1
	}
	res, err := s.db.ExecContext(ctx,
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

// GetUserByUsername returns the user with the given username, or ErrNotFound.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	var u User
	var createdAt int64
	var isAdmin int
	err := s.db.QueryRowContext(ctx,
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
	err := s.db.QueryRowContext(ctx,
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
