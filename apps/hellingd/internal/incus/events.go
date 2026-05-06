package incus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// LifecycleEvent is the Incus lifecycle event shape Helling consumes.
type LifecycleEvent struct {
	Action   string         `json:"action"`
	Source   string         `json:"source"`
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata"`
}

// DialEvents opens an Incus events WebSocket connection.
func DialEvents(ctx context.Context, endpoint string) (*websocket.Conn, error) {
	return DialEventsWithClient(ctx, endpoint, nil)
}

// DialEventsWithClient opens an Incus events WebSocket using a custom HTTP client.
func DialEventsWithClient(ctx context.Context, endpoint string, client *http.Client) (*websocket.Conn, error) {
	conn, resp, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPClient: client})
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("dialing incus events websocket: %w", err)
	}
	return conn, nil
}

// WatchLifecycleEvents mirrors Incus lifecycle messages until ctx is canceled.
func WatchLifecycleEvents(ctx context.Context, endpoint string, client *http.Client, handle func([]byte)) {
	if endpoint == "" {
		endpoint = "ws://incus/1.0/events?type=lifecycle"
	}
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		conn, err := DialEventsWithClient(ctx, endpoint, client)
		if err != nil {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		for {
			_, body, err := conn.Read(ctx)
			if err != nil {
				_ = conn.Close(websocket.StatusNormalClosure, "")
				break
			}
			handle(body)
		}
	}
}

// MapLifecycleEvent maps raw Incus lifecycle JSON to a Helling event name and subject.
func MapLifecycleEvent(raw []byte) (name string, subject string, err error) {
	var ev LifecycleEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return "", "", fmt.Errorf("decoding incus lifecycle event: %w", err)
	}
	name = "instance.updated"
	switch ev.Action {
	case "created":
		name = "instance.created"
	case "deleted":
		name = "instance.deleted"
	case "started", "resumed":
		name = "instance.started"
	case "stopped", "paused":
		name = "instance.stopped"
	}
	return name, ev.Source, nil
}
