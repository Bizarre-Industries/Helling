package store

import (
	"context"
	"errors"
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
