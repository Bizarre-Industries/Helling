package server

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventsSSEStreamsNewEvents(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	seedRegularUser(t, st, "event-user")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "event-user", testPassword)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/v1/events", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status: got %d want 200", resp.StatusCode)
	}

	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				lines <- line
				return
			}
		}
	}()

	if _, err := srv.emitEvent(context.Background(), "schedule.created", "schedule-1", nil); err != nil {
		t.Fatalf("emitEvent: %v", err)
	}

	select {
	case line := <-lines:
		if line != "event: schedule.created" {
			t.Fatalf("SSE line: got %q want schedule.created event", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for live SSE event")
	}
}
