// Package events defines the Helling event payloads mirrored to SSE,
// webhooks, and the WebUI.
package events

import "time"

// Source identifies the subsystem that emitted an event.
type Source string

// Helling event source constants.
const (
	SourceHelling Source = "helling"
	SourceIncus   Source = "incus"
	SourcePodman  Source = "podman"
)

// Event is the shared payload shape mirrored to SSE, webhooks, and the WebUI.
type Event struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Time    time.Time      `json:"time"`
	Source  Source         `json:"source"`
	Subject string         `json:"subject"`
	Data    map[string]any `json:"data" tstype:"EventData"`
}

// IncusLifecycleEvent is the subset of Incus lifecycle events Helling mirrors.
type IncusLifecycleEvent struct {
	Action   string         `json:"action"`
	Source   string         `json:"source"`
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata" tstype:"EventData"`
}
