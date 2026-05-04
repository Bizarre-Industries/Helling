package poller

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// fakeIncus implements incus.Client for the poller tests. Only GetOperation
// is exercised; the rest panic so unexpected calls fail loudly.
type fakeIncus struct {
	state map[string]struct {
		status incus.OperationStatus
		errMsg string
	}
}

func (f *fakeIncus) ListInstances(_ context.Context) ([]incus.Instance, error) {
	panic("ListInstances not expected in poller tests")
}

func (f *fakeIncus) GetInstance(_ context.Context, _ string) (*incus.Instance, error) {
	panic("GetInstance not expected in poller tests")
}

func (f *fakeIncus) CreateInstance(_ context.Context, _ incus.InstanceCreate) (incus.OperationHandle, error) {
	panic("CreateInstance not expected in poller tests")
}

func (f *fakeIncus) UpdateInstanceState(_ context.Context, _, _ string, _ bool, _ int) (incus.OperationHandle, error) {
	panic("UpdateInstanceState not expected in poller tests")
}

func (f *fakeIncus) DeleteInstance(_ context.Context, _ string) (incus.OperationHandle, error) {
	panic("DeleteInstance not expected in poller tests")
}

func (f *fakeIncus) GetOperation(_ context.Context, id string) (incus.OperationStatus, string, error) {
	if v, ok := f.state[id]; ok {
		return v.status, v.errMsg, nil
	}
	return incus.OpUnknown, "", nil
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPollerAdvancesPendingToSuccess(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "alice", "h", false)
	op, _ := st.CreateOperation(ctx, u.ID, "instance.create", "web-1", "incus-uuid")

	fake := &fakeIncus{state: map[string]struct {
		status incus.OperationStatus
		errMsg string
	}{
		"incus-uuid": {incus.OpSuccess, ""},
	}}

	tick(ctx, st, fake, discardLogger())

	got, _ := st.GetOperation(ctx, u.ID, op.ID)
	if got.Status != store.OpStatusSuccess {
		t.Fatalf("status: got %q want success", got.Status)
	}
}

func TestPollerRecordsFailureMessage(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "alice", "h", false)
	op, _ := st.CreateOperation(ctx, u.ID, "instance.create", "x", "incus-fail")

	fake := &fakeIncus{state: map[string]struct {
		status incus.OperationStatus
		errMsg string
	}{
		"incus-fail": {incus.OpFailure, "image fetch failed"},
	}}

	tick(ctx, st, fake, discardLogger())

	got, _ := st.GetOperation(ctx, u.ID, op.ID)
	if got.Status != store.OpStatusFailure {
		t.Fatalf("status: got %q want failure", got.Status)
	}
	if got.Error != "image fetch failed" {
		t.Fatalf("error: got %q", got.Error)
	}
}

func TestPollerNoIncusOpMarksSuccess(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "alice", "h", false)
	op, _ := st.CreateOperation(ctx, u.ID, "bookkeeping", "x", "")

	fake := &fakeIncus{state: nil}
	tick(ctx, st, fake, discardLogger())

	got, _ := st.GetOperation(ctx, u.ID, op.ID)
	if got.Status != store.OpStatusSuccess {
		t.Fatalf("status: got %q want success (bookkeeping)", got.Status)
	}
}

func TestRunExitsOnContextCancel(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		Run(ctx, st, &fakeIncus{}, discardLogger(), 10*time.Millisecond)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("poller did not exit within 1s of ctx cancel")
	}
}
