package authrepo_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

func newRepo(t *testing.T) *authrepo.Repo {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "repo.db") + "?cache=shared"
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	return authrepo.New(pool)
}

func TestCountAdmins_FreshDB(t *testing.T) {
	r := newRepo(t)
	n, err := r.CountAdmins(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestCreateUser_AndCount(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	u, err := r.CreateUser(ctx, "alice", "admin", "$argon2id$v=19$m=65536,t=3,p=4$AA$BB")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == "" || u.Username != "alice" || u.Role != "admin" || u.Status != "active" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if !u.PasswordHash.Valid {
		t.Error("password hash should be stored")
	}

	n, err := r.CountAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 admin, got %d", n)
	}
}

func TestCreateUser_NoPasswordKeepsNullHash(t *testing.T) {
	r := newRepo(t)
	u, err := r.CreateUser(context.Background(), "pam-user", "user", "")
	if err != nil {
		t.Fatal(err)
	}
	if u.PasswordHash.Valid {
		t.Errorf("PAM-backed user must have NULL password_hash")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	if _, err := r.CreateUser(ctx, "alice", "admin", "hash1"); err != nil {
		t.Fatal(err)
	}
	_, err := r.CreateUser(ctx, "alice", "user", "hash2")
	if !errors.Is(err, authrepo.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestGetUserByUsername_AndByID(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	created, err := r.CreateUser(ctx, "bob", "auditor", "hash")
	if err != nil {
		t.Fatal(err)
	}

	got, err := r.GetUserByUsername(ctx, "bob")
	if err != nil {
		t.Fatalf("by username: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id mismatch: %s vs %s", got.ID, created.ID)
	}

	got2, err := r.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if got2.Username != "bob" {
		t.Errorf("username mismatch: %q", got2.Username)
	}

	if _, err := r.GetUserByUsername(ctx, "ghost"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := r.GetUserByID(ctx, "missing"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestListUsers_PaginatesAndCounts(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	for _, name := range []string{"a", "b", "c", "d"} {
		if _, err := r.CreateUser(ctx, name, "user", "h"); err != nil {
			t.Fatal(err)
		}
	}
	page, total, err := r.ListUsers(ctx, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page) != 2 {
		t.Errorf("page size = %d, want 2", len(page))
	}
	page2, _, err := r.ListUsers(ctx, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 size = %d, want 2", len(page2))
	}
}

func TestSessionCreate_GetByRefresh_Revoke(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "sessionuser", "user", "h")
	if err != nil {
		t.Fatal(err)
	}

	expires := time.Now().Add(24 * time.Hour)
	s, err := r.CreateSession(ctx, u.ID, "raw-token-abc", expires, "curl/8", "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == "" {
		t.Fatal("session id required")
	}
	if s.RefreshTokenHash == "" || s.RefreshTokenHash == "raw-token-abc" {
		t.Fatal("refresh_token_hash must be hashed, not raw")
	}

	got, err := r.GetActiveSessionByRefresh(ctx, "raw-token-abc")
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if got.ID != s.ID {
		t.Fatalf("id mismatch: %s vs %s", got.ID, s.ID)
	}

	if _, err := r.GetActiveSessionByRefresh(ctx, "bogus"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	if err := r.RevokeSession(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetActiveSessionByRefresh(ctx, "raw-token-abc"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound after revoke, got %v", err)
	}
}

func TestRevokeSessionByRefresh(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "u", "user", "h")
	if _, err := r.CreateSession(ctx, u.ID, "tok", time.Now().Add(time.Hour), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := r.RevokeSessionByRefresh(ctx, "tok"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetActiveSessionByRefresh(ctx, "tok"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := r.RevokeSessionByRefresh(ctx, "tok"); err != nil {
		t.Fatalf("second revoke: %v", err)
	}
}

func TestGetActiveSessionByRefresh_ExpiredSessionNotReturned(t *testing.T) {
	r := newRepo(t)
	r.SetClock(func() time.Time { return time.Unix(1_700_000_000, 0) })
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "u", "user", "h")
	if _, err := r.CreateSession(ctx, u.ID, "past", time.Unix(1_700_000_000, 0).Add(-time.Hour), "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetActiveSessionByRefresh(ctx, "past"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound for expired session, got %v", err)
	}
}

func TestRecordEvent(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "u", "user", "h")

	if err := r.RecordEvent(ctx, u.ID, "auth.login_ok", "10.0.0.1", "curl", `{"a":1}`); err != nil {
		t.Fatalf("record with user: %v", err)
	}
	if err := r.RecordEvent(ctx, "", "auth.login_fail", "", "", ""); err != nil {
		t.Fatalf("record anon: %v", err)
	}
}

func TestSha256HexStable(t *testing.T) {
	a := authrepo.Sha256Hex("same")
	b := authrepo.Sha256Hex("same")
	if a != b {
		t.Fatal("hash must be deterministic")
	}
	if authrepo.Sha256Hex("other") == a {
		t.Fatal("different inputs must hash differently")
	}
}

func TestSetClockAffectsEventTimestamps(t *testing.T) {
	r := newRepo(t)
	fixed := time.Unix(1_700_000_000, 0)
	r.SetClock(func() time.Time { return fixed })
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "clocktest", "admin", "h")
	if err != nil {
		t.Fatal(err)
	}
	if u.CreatedAt != fixed.Unix() {
		t.Errorf("CreatedAt = %d, want %d", u.CreatedAt, fixed.Unix())
	}
}

func TestTOTPSecret_UpsertGetSetDelete(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "alice", "user", "h")

	if err := r.UpsertTOTPSecret(ctx, u.ID, []byte("secret123"), false); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetTOTPSecret(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.EncryptedSecret) != "secret123" || got.Enabled {
		t.Fatalf("bad totp row: %+v", got)
	}

	if err := r.SetTOTPEnabled(ctx, u.ID, true); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetTOTPSecret(ctx, u.ID)
	if !got.Enabled {
		t.Fatal("expected enabled=true")
	}

	// Upsert replaces secret.
	if err := r.UpsertTOTPSecret(ctx, u.ID, []byte("new-secret"), true); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetTOTPSecret(ctx, u.ID)
	if string(got.EncryptedSecret) != "new-secret" {
		t.Errorf("upsert did not replace: %q", string(got.EncryptedSecret))
	}

	if err := r.DeleteTOTPSecret(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetTOTPSecret(ctx, u.ID); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}

func TestRecoveryCodes_InsertListMark(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "bob", "user", "h")

	if err := r.InsertRecoveryCodes(ctx, u.ID, []string{"h1", "h2", "h3"}); err != nil {
		t.Fatal(err)
	}
	codes, err := r.ListUnusedRecoveryCodes(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 3 {
		t.Fatalf("want 3 codes, got %d", len(codes))
	}
	if err := r.MarkRecoveryCodeUsed(ctx, codes[0].ID); err != nil {
		t.Fatal(err)
	}
	codes, _ = r.ListUnusedRecoveryCodes(ctx, u.ID)
	if len(codes) != 2 {
		t.Fatalf("want 2 unused, got %d", len(codes))
	}
}

func TestRecoveryCodes_InsertEmptyIsNoop(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "c", "user", "h")
	if err := r.InsertRecoveryCodes(ctx, u.ID, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAPITokens_CreateListRevokeLookup(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "u", "user", "h")

	exp := time.Now().Add(24 * time.Hour)
	tok, err := r.CreateAPIToken(ctx, u.ID, "ci", "deadbeef", "read", exp)
	if err != nil {
		t.Fatal(err)
	}
	if tok.ID == "" {
		t.Fatal("missing id")
	}

	list, err := r.ListAPITokens(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 token, got %d", len(list))
	}

	got, err := r.GetAPITokenByHash(ctx, "deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != tok.ID {
		t.Fatal("id mismatch on lookup by hash")
	}

	if err := r.TouchAPITokenLastUsed(ctx, tok.ID); err != nil {
		t.Fatal(err)
	}

	if err := r.RevokeAPIToken(ctx, u.ID, tok.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.RevokeAPIToken(ctx, u.ID, tok.ID); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound on re-revoke, got %v", err)
	}
	if _, err := r.GetAPITokenByHash(ctx, "deadbeef"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("want ErrNotFound after revoke, got %v", err)
	}
}

func TestAPITokens_ExpiredNotReturned(t *testing.T) {
	r := newRepo(t)
	r.SetClock(func() time.Time { return time.Unix(1_700_000_000, 0) })
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "u", "user", "h")
	_, _ = r.CreateAPIToken(ctx, u.ID, "expired", "oldhash", "read", time.Unix(1_700_000_000, 0).Add(-time.Hour))

	if _, err := r.GetAPITokenByHash(ctx, "oldhash"); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired token, got %v", err)
	}
}

func TestWebhook_CreateGetListUpdateDelete(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, _ := r.CreateUser(ctx, "wh", "admin", "h")

	w, err := r.CreateWebhook(ctx, "ci", "https://example.com/h", []string{"instance.created"}, []byte("secret-bytes"), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if w.ID == "" || !w.Enabled {
		t.Fatalf("bad webhook: %+v", w)
	}

	got, err := r.GetWebhook(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://example.com/h" {
		t.Fatalf("url mismatch: %s", got.URL)
	}
	if len(got.Events) != 1 || got.Events[0] != "instance.created" {
		t.Fatalf("events mismatch: %+v", got.Events)
	}

	list, total, err := r.ListWebhooks(ctx, u.ID, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("total=%d list=%d", total, len(list))
	}

	newName := "renamed"
	enabled := false
	if err := r.UpdateWebhook(ctx, w.ID, &newName, nil, []string{"instance.deleted"}, &enabled); err != nil {
		t.Fatal(err)
	}
	got2, _ := r.GetWebhook(ctx, w.ID)
	if got2.Name != "renamed" || got2.Enabled {
		t.Fatalf("update did not apply: %+v", got2)
	}

	if err := r.UpdateWebhook(ctx, w.ID, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateWebhook(ctx, "missing", &newName, nil, nil, nil); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := r.MarkWebhookDelivered(ctx, w.ID, time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := r.DeleteWebhook(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteWebhook(ctx, w.ID); !errors.Is(err, authrepo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on re-delete, got %v", err)
	}
}

func TestWebhook_CreateRequiresCreator(t *testing.T) {
	r := newRepo(t)
	if _, err := r.CreateWebhook(context.Background(), "x", "https://x", []string{"e"}, nil, ""); err == nil {
		t.Fatal("expected error on empty createdBy")
	}
}

func TestUser_DeleteAndSetScope(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "scopey", "admin", "")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := r.SetUserScope(ctx, u.ID, "incus:admin"); err != nil {
		t.Fatalf("set scope: %v", err)
	}
	if err := r.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := r.GetUserByID(ctx, u.ID); err == nil {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestSystemConfig_PutGetUpsert(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	if _, err := r.GetSystemConfig(ctx, "missing"); err == nil {
		t.Fatal("expected ErrNotFound for missing key")
	}
	if err := r.PutSystemConfig(ctx, "log.level", "info", ""); err != nil {
		t.Fatalf("put: %v", err)
	}
	v, err := r.GetSystemConfig(ctx, "log.level")
	if err != nil || v != "info" {
		t.Fatalf("get: v=%q err=%v", v, err)
	}
	u, err := r.CreateUser(ctx, "cfg-actor", "admin", "")
	if err != nil {
		t.Fatalf("actor: %v", err)
	}
	if err := r.PutSystemConfig(ctx, "log.level", "debug", u.ID); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	v, _ = r.GetSystemConfig(ctx, "log.level")
	if v != "debug" {
		t.Fatalf("upsert value: got %q", v)
	}
}

func TestWebhook_MarkDelivered(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()
	u, err := r.CreateUser(ctx, "whmark", "admin", "")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	w, err := r.CreateWebhook(ctx, "mk", "https://x", []string{"e"}, []byte("s"), u.ID)
	if err != nil {
		t.Fatalf("wh: %v", err)
	}
	when := time.Unix(1700000000, 0)
	if err := r.MarkWebhookDelivered(ctx, w.ID, when); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, err := r.GetWebhook(ctx, w.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.LastDeliveryAt.Valid || got.LastDeliveryAt.Int64 != when.Unix() {
		t.Fatalf("last_delivery_at not stored: %+v", got.LastDeliveryAt)
	}
}
