package store

import (
	"context"
	"errors"
	"testing"
)

func TestOperationCreateAndGet(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	u, err := st.CreateUser(ctx, "alice", "h", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	op, err := st.CreateOperation(ctx, u.ID, "instance.create", "web-1", "incus-uuid")
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	if op.Status != OpStatusPending {
		t.Fatalf("status: got %q want pending", op.Status)
	}

	got, err := st.GetOperation(ctx, u.ID, op.ID)
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}
	if got.Kind != "instance.create" || got.Target != "web-1" || got.IncusOpID != "incus-uuid" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestGetOperationOwnershipIsolation(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()

	a, _ := st.CreateUser(ctx, "alice", "h", false)
	b, _ := st.CreateUser(ctx, "bob", "h", false)
	op, _ := st.CreateOperation(ctx, a.ID, "instance.create", "x", "")

	if _, err := st.GetOperation(ctx, b.ID, op.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bob fetched alice's op: got %v want ErrNotFound", err)
	}
}

func TestUpdateOperationStatus(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "alice", "h", false)
	op, _ := st.CreateOperation(ctx, u.ID, "instance.delete", "x", "incus-uuid")

	if err := st.UpdateOperationStatus(ctx, op.ID, OpStatusRunning, ""); err != nil {
		t.Fatalf("UpdateOperationStatus running: %v", err)
	}
	got, _ := st.GetOperation(ctx, u.ID, op.ID)
	if got.Status != OpStatusRunning {
		t.Fatalf("status: got %q want running", got.Status)
	}

	if err := st.UpdateOperationStatus(ctx, op.ID, OpStatusFailure, "boom"); err != nil {
		t.Fatalf("UpdateOperationStatus failure: %v", err)
	}
	got, _ = st.GetOperation(ctx, u.ID, op.ID)
	if got.Status != OpStatusFailure || got.Error != "boom" {
		t.Fatalf("failure update: %+v", got)
	}
}

func TestListOperationsFilterAndLimit(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "alice", "h", false)

	op1, _ := st.CreateOperation(ctx, u.ID, "instance.create", "a", "")
	_, _ = st.CreateOperation(ctx, u.ID, "instance.create", "b", "")
	_ = st.UpdateOperationStatus(ctx, op1.ID, OpStatusSuccess, "")

	all, err := st.ListOperations(ctx, u.ID, "", 0)
	if err != nil {
		t.Fatalf("ListOperations: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all: got %d want 2", len(all))
	}

	successOnly, _ := st.ListOperations(ctx, u.ID, OpStatusSuccess, 0)
	if len(successOnly) != 1 || successOnly[0].ID != op1.ID {
		t.Fatalf("success filter: got %+v", successOnly)
	}

	limited, _ := st.ListOperations(ctx, u.ID, "", 1)
	if len(limited) != 1 {
		t.Fatalf("limit: got %d want 1", len(limited))
	}
}

func TestListActiveOperations(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	u, _ := st.CreateUser(ctx, "alice", "h", false)

	pending, _ := st.CreateOperation(ctx, u.ID, "instance.create", "a", "")
	running, _ := st.CreateOperation(ctx, u.ID, "instance.create", "b", "")
	done, _ := st.CreateOperation(ctx, u.ID, "instance.create", "c", "")
	_ = st.UpdateOperationStatus(ctx, running.ID, OpStatusRunning, "")
	_ = st.UpdateOperationStatus(ctx, done.ID, OpStatusSuccess, "")

	active, err := st.ListActiveOperations(ctx)
	if err != nil {
		t.Fatalf("ListActiveOperations: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("active: got %d want 2", len(active))
	}
	seen := map[string]bool{}
	for _, op := range active {
		seen[op.ID] = true
	}
	if !seen[pending.ID] || !seen[running.ID] {
		t.Fatalf("missing pending/running ops; got %+v", active)
	}
}
