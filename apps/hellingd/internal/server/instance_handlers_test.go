package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	incusapi "github.com/lxc/incus/v6/shared/api"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// fakeIncusClient is a minimal incus.Client used to drive instance handlers
// from tests without needing a real Incus daemon.
type fakeIncusClient struct {
	instances   []incusapi.Instance
	createErr   error
	deleteErr   error
	stateErr    error
	createdName string
	stateCalls  []string
	deletedName string
	nextOpID    string
}

func (f *fakeIncusClient) ListInstances(_ context.Context) ([]incusapi.Instance, error) {
	return f.instances, nil
}

func (f *fakeIncusClient) GetInstance(_ context.Context, name string) (*incusapi.Instance, error) {
	for i := range f.instances {
		if f.instances[i].Name == name {
			return &f.instances[i], nil
		}
	}
	return nil, errors.New("not found")
}

//nolint:gocritic // interface method; struct passed by value upstream
func (f *fakeIncusClient) CreateInstance(_ context.Context, req incusapi.InstancesPost) (incus.OperationHandle, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.createdName = req.Name
	return &fakeOpHandle{id: f.nextOpID}, nil
}

func (f *fakeIncusClient) UpdateInstanceState(_ context.Context, name, action string, _ bool, _ int) (incus.OperationHandle, error) {
	if f.stateErr != nil {
		return nil, f.stateErr
	}
	f.stateCalls = append(f.stateCalls, name+":"+action)
	return &fakeOpHandle{id: f.nextOpID}, nil
}

func (f *fakeIncusClient) DeleteInstance(_ context.Context, name string) (incus.OperationHandle, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	f.deletedName = name
	return &fakeOpHandle{id: f.nextOpID}, nil
}

func (f *fakeIncusClient) GetOperation(_ context.Context, _ string) (incus.OperationStatus, string, error) {
	return incus.OpRunning, "", nil
}

type fakeOpHandle struct{ id string }

func (h *fakeOpHandle) ID() string  { return h.id }
func (h *fakeOpHandle) Wait() error { return nil }

func newServerWithIncus(t *testing.T, fake incus.Client) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := New(&Config{
		Store:   st,
		Logger:  logger,
		Version: VersionInfo{Version: "test"},
		Auth: AuthSettings{
			SessionTTL:     time.Hour,
			UsernameLimit:  5,
			UsernameWindow: time.Minute,
			IPLimit:        20,
			IPWindow:       time.Minute,
			Argon2:         auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32},
		},
		Incus: fake,
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return srv, st
}

func TestListInstancesShapesAndFilters(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{
		instances: []incusapi.Instance{
			{Name: "web-1", Type: "container", Status: "Running", InstancePut: incusapi.InstancePut{Config: map[string]string{"image.alias": "images:debian/13"}}},
			{Name: "db-1", Type: "container", Status: "Stopped"},
		},
	}
	srv, st := newServerWithIncus(t, fake)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cookie := loginCookie(t, ts, "alice", "secret-password-123")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/instances", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/instances: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}
	var got []incus.Instance
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	_ = resp.Body.Close()
	if len(got) != 2 {
		t.Fatalf("len: got %d want 2", len(got))
	}
	if got[0].Status != "running" || got[1].Status != "stopped" {
		t.Fatalf("status casing: %+v", got)
	}

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/instances?status=running", http.NoBody)
	req.AddCookie(cookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("filtered GET: %v", err)
	}
	got = nil
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode filtered: %v", err)
	}
	_ = resp.Body.Close()
	if len(got) != 1 || got[0].Name != "web-1" {
		t.Fatalf("filtered result: %+v", got)
	}
}

func TestCreateInstanceCreatesOperationRow(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{nextOpID: "incus-op-123"}
	srv, st := newServerWithIncus(t, fake)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cookie := loginCookie(t, ts, "alice", "secret-password-123")

	body, _ := json.Marshal(map[string]any{"name": "web-1", "image": "images:debian/13"})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/instances", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/instances: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d want 202", resp.StatusCode)
	}
	var op map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		t.Fatalf("decode: %v", err)
	}
	_ = resp.Body.Close()
	if op["kind"] != "instance.create" || op["target"] != "web-1" || op["status"] != "pending" {
		t.Fatalf("op response: %+v", op)
	}
	if fake.createdName != "web-1" {
		t.Fatalf("Incus.CreateInstance not called for web-1; got %q", fake.createdName)
	}
}

func TestStartStopInstanceQueuesOps(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{nextOpID: "incus-op-state"}
	srv, st := newServerWithIncus(t, fake)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cookie := loginCookie(t, ts, "alice", "secret-password-123")

	for _, action := range []string{"start", "stop"} {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/instances/web-1/"+action, http.NoBody)
		req.AddCookie(cookie)
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatalf("%s POST: %v", action, err)
		}
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("%s status: got %d want 202", action, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	if len(fake.stateCalls) != 2 {
		t.Fatalf("expected 2 state calls, got %v", fake.stateCalls)
	}
	if fake.stateCalls[0] != "web-1:start" || fake.stateCalls[1] != "web-1:stop" {
		t.Fatalf("state calls: got %v", fake.stateCalls)
	}
}

func TestDeleteInstanceQueuesOp(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{nextOpID: "incus-op-del"}
	srv, st := newServerWithIncus(t, fake)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cookie := loginCookie(t, ts, "alice", "secret-password-123")
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodDelete, ts.URL+"/v1/instances/web-1", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d want 202", resp.StatusCode)
	}
	_ = resp.Body.Close()
	if fake.deletedName != "web-1" {
		t.Fatalf("Incus.DeleteInstance not called for web-1; got %q", fake.deletedName)
	}
}

func TestListAndGetOperationsThroughHandlers(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{nextOpID: "incus-op-list"}
	srv, st := newServerWithIncus(t, fake)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "alice", "secret-password-123")

	body, _ := json.Marshal(map[string]any{"name": "web-1", "image": "images:debian/13"})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/instances", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, _ := ts.Client().Do(req)
	var created map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()
	opID, _ := created["id"].(string)
	if opID == "" {
		t.Fatal("created op missing id")
	}

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/operations", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/operations: %v", err)
	}
	var ops []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&ops); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	_ = resp.Body.Close()
	if len(ops) != 1 || ops[0]["id"] != opID {
		t.Fatalf("list: %+v", ops)
	}

	req, _ = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/operations/"+opID, http.NoBody)
	req.AddCookie(cookie)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /v1/operations/{id}: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get-by-id status: %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestListInstancesWithoutIncusReturns503(t *testing.T) {
	t.Parallel()
	srv, st := newServerWithIncus(t, nil)
	seedUser(t, st, "alice", "secret-password-123")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "alice", "secret-password-123")

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/v1/instances", http.NoBody)
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d want 503", resp.StatusCode)
	}
}
