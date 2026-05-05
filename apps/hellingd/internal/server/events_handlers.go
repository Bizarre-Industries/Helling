package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

type eventResponse struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Time    time.Time      `json:"time"`
	Source  string         `json:"source"`
	Subject string         `json:"subject"`
	Data    map[string]any `json:"data"`
}

type eventHub struct {
	mu          sync.Mutex
	subscribers map[chan *store.Event]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[chan *store.Event]struct{})}
}

func (h *eventHub) subscribe() chan *store.Event {
	ch := make(chan *store.Event, 16)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *eventHub) unsubscribe(ch chan *store.Event) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *eventHub) publish(ev *store.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (s *Server) emitEvent(ctx context.Context, eventType, subject string, data map[string]any) (store.Event, error) {
	ev, err := s.cfg.Store.CreateEvent(ctx, eventType, "helling", subject, data)
	if err != nil {
		return store.Event{}, err
	}
	s.events.publish(&ev)
	if eventType != "webhook.test" {
		s.enqueueWebhookEvent(ctx, &ev)
	} else if err := s.cfg.Store.MarkEventOutboxDelivered(ctx, ev.ID); err != nil {
		s.cfg.Logger.WarnContext(ctx, "mark webhook test event delivered", slog.Any("err", err), slog.String("event", ev.ID))
	}
	return ev, nil
}

func (s *Server) emitExternalEvent(ctx context.Context, eventType, source, subject string, data map[string]any) (store.Event, error) {
	ev, err := s.cfg.Store.CreateEvent(ctx, eventType, source, subject, data)
	if err != nil {
		return store.Event{}, err
	}
	s.events.publish(&ev)
	s.enqueueWebhookEvent(ctx, &ev)
	return ev, nil
}

// MirrorIncusLifecycleEvent maps one Incus lifecycle frame into Helling events.
func (s *Server) MirrorIncusLifecycleEvent(ctx context.Context, raw []byte) error {
	name, subject, err := incus.MapLifecycleEvent(raw)
	if err != nil {
		return err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		data = map[string]any{}
	}
	_, err = s.emitExternalEvent(ctx, name, "incus", subject, data)
	return err
}

func (s *Server) enqueueWebhookEvent(ctx context.Context, ev *store.Event) {
	if ev == nil {
		return
	}
	_ = ctx
	select {
	case s.webhookWake <- struct{}{}:
	default:
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if !wantsSSE(r) {
		s.handleEventsSnapshot(w, r)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	if !s.replayStoredEvents(w, r) {
		return
	}
	flushSSE(w, flusher)
	s.streamLiveEvents(w, r, flusher)
}

func (s *Server) replayStoredEvents(w http.ResponseWriter, r *http.Request) bool {
	typeFilter := r.URL.Query().Get("type")
	cursor := r.Header.Get("Last-Event-ID")
	if cursor != "" {
		sent := 0
		for sent < eventReplayMaxEvents {
			limit := eventReplayPageSize
			if remaining := eventReplayMaxEvents - sent; remaining < limit {
				limit = remaining
			}
			events, err := s.cfg.Store.ListEvents(r.Context(), limit, typeFilter, cursor)
			if err != nil {
				return true
			}
			if !writeEventsForward(w, events) {
				return false
			}
			sent += len(events)
			if len(events) < limit {
				return true
			}
			cursor = events[len(events)-1].ID
		}
	}
	events, err := s.cfg.Store.ListEvents(r.Context(), 50, typeFilter, "")
	if err != nil {
		return true
	}
	return writeEventsReverse(w, events)
}

func (s *Server) streamLiveEvents(w http.ResponseWriter, r *http.Request, flusher http.Flusher) {
	live := s.events.subscribe()
	defer s.events.unsubscribe(live)
	typeFilter := r.URL.Query().Get("type")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-live:
			if !eventMatchesFilter(ev, typeFilter) {
				continue
			}
			if err := writeSSE(w, ev); err != nil {
				return
			}
			flushSSE(w, flusher)
		case <-ticker.C:
			if err := writeSSEHeartbeat(w); err != nil {
				return
			}
			flushSSE(w, flusher)
		}
	}
}

func writeEventsForward(w http.ResponseWriter, events []store.Event) bool {
	for i := range events {
		if err := writeSSE(w, &events[i]); err != nil {
			return false
		}
	}
	return true
}

func writeEventsReverse(w http.ResponseWriter, events []store.Event) bool {
	for i := len(events) - 1; i >= 0; i-- {
		if err := writeSSE(w, &events[i]); err != nil {
			return false
		}
	}
	return true
}

func (s *Server) handleEventsSnapshot(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	rows, err := s.cfg.Store.ListEvents(r.Context(), limit, r.URL.Query().Get("type"), r.URL.Query().Get("since"))
	if err != nil {
		s.cfg.Logger.Error("list events", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]eventResponse, 0, len(rows))
	for i := range rows {
		out = append(out, eventToResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func wantsSSE(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

func eventMatchesFilter(ev *store.Event, typeFilter string) bool {
	if typeFilter == "" {
		return true
	}
	if strings.HasSuffix(typeFilter, ".") {
		return strings.HasPrefix(ev.Type, typeFilter)
	}
	return ev.Type == typeFilter
}

func writeSSE(w http.ResponseWriter, ev *store.Event) error {
	body, err := json.Marshal(eventToResponse(ev))
	if err != nil {
		return err
	}
	setSSEWriteDeadline(w)
	_, err = fmt.Fprintf(w, "event: %s\nid: %s\ndata: %s\n\n", ev.Type, ev.ID, body)
	return err
}

const (
	eventReplayPageSize  = 500
	eventReplayMaxEvents = 2000
	sseWriteTimeout      = 10 * time.Second
)

func writeSSEHeartbeat(w http.ResponseWriter) error {
	setSSEWriteDeadline(w)
	_, err := fmt.Fprintf(w, ": heartbeat\n\n")
	return err
}

func flushSSE(w http.ResponseWriter, flusher http.Flusher) {
	setSSEWriteDeadline(w)
	flusher.Flush()
}

func setSSEWriteDeadline(w http.ResponseWriter) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(sseWriteTimeout))
}

func eventToResponse(ev *store.Event) eventResponse {
	return eventResponse{
		ID:      ev.ID,
		Type:    ev.Type,
		Time:    ev.Time,
		Source:  ev.Source,
		Subject: ev.Subject,
		Data:    ev.Data,
	}
}
