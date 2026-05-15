package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestUserRoundtrip(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	u, err := st.CreateUser(ctx, "alice", "hash-of-something", true)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("CreateUser returned zero ID")
	}
	if !u.IsAdmin {
		t.Fatal("IsAdmin not persisted")
	}

	got, err := st.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.ID != u.ID || got.Username != "alice" || got.PasswordHash != "hash-of-something" {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", got, u)
	}

	byID, err := st.GetUserByID(ctx, u.ID)
	if err != nil || byID.Username != "alice" {
		t.Fatalf("GetUserByID: got %+v err %v", byID, err)
	}
}

func TestGetUserNotFound(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	_, err := st.GetUserByUsername(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCountUsers(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	n, err := st.CountUsers(ctx)
	if err != nil || n != 0 {
		t.Fatalf("CountUsers empty: got %d err %v", n, err)
	}
	if _, err := st.CreateUser(ctx, "u1", "h", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := st.CreateUser(ctx, "u2", "h", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	n, _ = st.CountUsers(ctx)
	if n != 2 {
		t.Fatalf("CountUsers: got %d want 2", n)
	}
}

func TestCreateFirstAdminOnlyWhenUsersTableIsEmpty(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	u, err := st.CreateFirstAdmin(ctx, "admin", "hash")
	if err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	if !u.IsAdmin || u.Username != "admin" {
		t.Fatalf("CreateFirstAdmin user = %+v", u)
	}

	if _, err := st.CreateFirstAdmin(ctx, "second", "hash"); !errors.Is(err, ErrUsersExist) {
		t.Fatalf("CreateFirstAdmin second err = %v, want ErrUsersExist", err)
	}
	n, err := st.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if n != 1 {
		t.Fatalf("CountUsers = %d, want 1", n)
	}
}

func TestSessionRoundtrip(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	u, err := st.CreateUser(ctx, "bob", "h", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	sess, err := st.CreateSession(ctx, "deadbeef", u.ID, time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := st.GetSessionByTokenHash(ctx, "deadbeef")
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}
	if got.UserID != u.ID || got.ID != sess.ID {
		t.Fatalf("session mismatch: %+v vs %+v", got, sess)
	}

	if err := st.TouchSession(ctx, "deadbeef"); err != nil {
		t.Fatalf("TouchSession: %v", err)
	}
	if err := st.DeleteSession(ctx, "deadbeef"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := st.GetSessionByTokenHash(ctx, "deadbeef"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSessionExpiryReturnsNotFound(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "carol", "h", false)
	if _, err := st.CreateSession(ctx, "expired", u.ID, -time.Hour); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := st.GetSessionByTokenHash(ctx, "expired"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session: got %v want ErrNotFound", err)
	}
}

func TestOpenCreatesPrivatePaths(t *testing.T) {
	t.Parallel()
	stateDir := filepath.Join(t.TempDir(), "state")
	st, err := Open(stateDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	dirInfo, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("stat state dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o750 {
		t.Fatalf("state dir mode = %o, want 750", got)
	}

	dbInfo, err := os.Stat(filepath.Join(stateDir, "helling.db"))
	if err != nil {
		t.Fatalf("stat db: %v", err)
	}
	if got := dbInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("db mode = %o, want 600", got)
	}
}

func TestOpenRepairsOverPermissiveDBMode(t *testing.T) {
	t.Parallel()
	stateDir := t.TempDir()
	dbPath := filepath.Join(stateDir, "helling.db")
	if err := os.WriteFile(dbPath, []byte(""), 0o666); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	st, err := Open(stateDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("db mode = %o, want 600", got)
	}
}
