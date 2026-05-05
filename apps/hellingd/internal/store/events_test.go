package store

import (
	"context"
	"testing"
	"time"
)

func TestClaimPendingOutboxEventsPreventsDuplicateClaim(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	hook := seedWebhookForEvents(t, st, "instance.*")
	ev, err := st.CreateEvent(ctx, "instance.created", "test", "vm1", nil)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	first, err := st.ClaimPendingOutboxEvents(ctx, 100)
	if err != nil {
		t.Fatalf("ClaimPendingOutboxEvents first: %v", err)
	}
	if len(first) != 1 || first[0].ID != ev.ID || first[0].WebhookID != hook.ID || first[0].OutboxID == "" {
		t.Fatalf("first claim = %#v, want event %s", first, ev.ID)
	}
	second, err := st.ClaimPendingOutboxEvents(ctx, 100)
	if err != nil {
		t.Fatalf("ClaimPendingOutboxEvents second: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second claim = %#v, want none", second)
	}
}

func TestMarkEventOutboxProcessingClaimsOnce(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	seedWebhookForEvents(t, st, "instance.created")
	ev, err := st.CreateEvent(ctx, "instance.created", "test", "vm1", nil)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	claimed, err := st.MarkEventOutboxProcessing(ctx, ev.ID)
	if err != nil {
		t.Fatalf("MarkEventOutboxProcessing first: %v", err)
	}
	if !claimed {
		t.Fatal("first processing claim returned false")
	}
	claimed, err = st.MarkEventOutboxProcessing(ctx, ev.ID)
	if err != nil {
		t.Fatalf("MarkEventOutboxProcessing second: %v", err)
	}
	if claimed {
		t.Fatal("second processing claim returned true")
	}
}

func TestClaimPendingOutboxEventsReclaimsStaleProcessing(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	seedWebhookForEvents(t, st, "instance.created")
	ev, err := st.CreateEvent(ctx, "instance.created", "test", "vm1", nil)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	first, err := st.ClaimPendingOutboxEvents(ctx, 100)
	if err != nil {
		t.Fatalf("ClaimPendingOutboxEvents first: %v", err)
	}
	if len(first) != 1 || first[0].Attempt != 1 {
		t.Fatalf("first claim = %#v, want one attempt 1", first)
	}
	stale := time.Now().UTC().Add(-3 * time.Minute).Unix()
	if _, err := st.db.ExecContext(ctx, `UPDATE event_outbox SET updated_at = ? WHERE event_id = ?`, stale, ev.ID); err != nil {
		t.Fatalf("mark stale processing: %v", err)
	}
	second, err := st.ClaimPendingOutboxEvents(ctx, 100)
	if err != nil {
		t.Fatalf("ClaimPendingOutboxEvents stale: %v", err)
	}
	if len(second) != 1 || second[0].ID != ev.ID || second[0].Attempt != 2 {
		t.Fatalf("stale claim = %#v, want event %s attempt 2", second, ev.ID)
	}
}

func TestCreateEventQueuesOnlyMatchingWebhooks(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	matching := seedWebhookForEvents(t, st, "schedule.*")
	_ = seedWebhookForEvents(t, st, "instance.*")

	ev, err := st.CreateEvent(ctx, "schedule.created", "test", "schedule-1", nil)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	jobs, err := st.ClaimEventOutbox(ctx, ev.ID)
	if err != nil {
		t.Fatalf("ClaimEventOutbox: %v", err)
	}
	if len(jobs) != 1 || jobs[0].WebhookID != matching.ID || jobs[0].OutboxID == "" {
		t.Fatalf("claimed jobs = %#v, want one job for webhook %s", jobs, matching.ID)
	}
}

func TestListEventsSinceReturnsAscendingAfterID(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	first, err := st.CreateEvent(ctx, "instance.created", "test", "vm1", nil)
	if err != nil {
		t.Fatalf("CreateEvent first: %v", err)
	}
	second, err := st.CreateEvent(ctx, "instance.updated", "test", "vm1", nil)
	if err != nil {
		t.Fatalf("CreateEvent second: %v", err)
	}
	third, err := st.CreateEvent(ctx, "instance.deleted", "test", "vm1", nil)
	if err != nil {
		t.Fatalf("CreateEvent third: %v", err)
	}
	got, err := st.ListEvents(ctx, 10, "", first.ID)
	if err != nil {
		t.Fatalf("ListEvents since: %v", err)
	}
	if len(got) != 2 || got[0].ID != second.ID || got[1].ID != third.ID {
		t.Fatalf("ListEvents since = %#v, want %s then %s", got, second.ID, third.ID)
	}
}

func TestPruneEventsKeepsNewestRows(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	ctx := context.Background()
	base := time.Now().UTC().Unix()
	ids := make([]string, 5)
	for i := range ids {
		ev, err := st.CreateEvent(ctx, "instance.updated", "test", "vm1", nil)
		if err != nil {
			t.Fatalf("CreateEvent %d: %v", i, err)
		}
		ids[i] = ev.ID
		if _, err := st.db.ExecContext(ctx, `UPDATE events SET created_at = ? WHERE id = ?`, base+int64(i), ev.ID); err != nil {
			t.Fatalf("set created_at %d: %v", i, err)
		}
	}
	if err := st.PruneEvents(ctx, 2, 365*24*time.Hour); err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}
	got, err := st.ListEvents(ctx, 10, "", "")
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(got) != 2 || got[0].ID != ids[4] || got[1].ID != ids[3] {
		t.Fatalf("remaining events = %#v, want newest two %s, %s", got, ids[4], ids[3])
	}
}

func seedWebhookForEvents(t *testing.T, st *Store, events ...string) Webhook {
	t.Helper()
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "user-"+events[0], "hash", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	hook, err := st.CreateWebhook(ctx, u.ID, "ops-"+events[0], "https://example.com/hook", "top-secret-value", events)
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	return hook
}
