package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

func (s *Server) dispatchEventWebhooks(ctx context.Context, job *store.OutboxEvent) {
	ev := &job.Event
	hook, err := s.cfg.Store.GetWebhook(ctx, job.WebhookID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			if err := s.cfg.Store.MarkOutboxDelivered(ctx, job.OutboxID); err != nil {
				s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: mark missing webhook delivered", slog.Any("err", err), slog.String("event", ev.ID), slog.String("webhook", job.WebhookID))
			}
			return
		}
		s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: load webhook", slog.Any("err", err), slog.String("event", ev.ID), slog.String("webhook", job.WebhookID))
		_ = s.cfg.Store.MarkOutboxPending(ctx, job.OutboxID, time.Now().UTC().Add(30*time.Second))
		return
	}
	if !hook.Enabled || !webhookSubscribesToEvent(hook.Events, ev.Type) {
		if err := s.cfg.Store.MarkOutboxDelivered(ctx, job.OutboxID); err != nil {
			s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: mark disabled webhook delivered", slog.Any("err", err), slog.String("event", ev.ID), slog.String("webhook", job.WebhookID))
		}
		return
	}
	body, err := json.Marshal(eventToResponse(ev))
	if err != nil {
		s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: marshal event", slog.Any("err", err), slog.String("event", ev.ID))
		_ = s.cfg.Store.MarkOutboxPending(ctx, job.OutboxID, time.Now().UTC().Add(30*time.Second))
		return
	}
	secret, err := s.cfg.Store.DecryptSecret(hook.SecretEncrypted)
	if err != nil {
		s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: decrypt secret", slog.Any("err", err), slog.String("webhook", hook.ID))
		msg := err.Error()
		if _, createErr := s.cfg.Store.CreateWebhookDelivery(ctx, hook.ID, ev.ID, ev.Type, outcomeFailed, nil, nil, &msg, job.Attempt); createErr != nil {
			s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: record decrypt failure", slog.Any("err", createErr), slog.String("webhook", hook.ID), slog.String("event", ev.ID))
		}
		if s.rescheduleWebhookEvent(ctx, job) {
			return
		}
	}
	if err == nil && !s.deliverWebhookAttempt(ctx, &hook, ev, secret, body, job.Attempt) {
		if s.rescheduleWebhookEvent(ctx, job) {
			return
		}
	}
	if err := s.cfg.Store.MarkOutboxDelivered(ctx, job.OutboxID); err != nil {
		s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: mark delivered", slog.Any("err", err), slog.String("event", ev.ID), slog.String("webhook", hook.ID))
	}
}

func (s *Server) webhookWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-s.webhookQueue:
			s.dispatchEventWebhooks(ctx, &ev)
		}
	}
}

func (s *Server) outboxDrainer(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.drainWebhookOutbox(ctx)
		case <-s.webhookWake:
			s.drainWebhookOutbox(ctx)
		}
	}
}

func (s *Server) drainWebhookOutbox(ctx context.Context) {
	limit := cap(s.webhookQueue) - len(s.webhookQueue)
	if limit <= 0 {
		return
	}
	if limit > 1000 {
		limit = 1000
	}
	claimCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	events, err := s.cfg.Store.ClaimPendingOutboxEvents(claimCtx, limit)
	if err != nil {
		s.cfg.Logger.ErrorContext(ctx, "webhook outbox: list pending", slog.Any("err", err))
		return
	}
	for i := range events {
		select {
		case s.webhookQueue <- events[i]:
		default:
			_ = s.cfg.Store.MarkOutboxPending(ctx, events[i].OutboxID, time.Now().UTC().Add(5*time.Second))
			s.cfg.Logger.WarnContext(ctx, "webhook event queue full; rescheduled outbox event", slog.String("event", events[i].ID), slog.String("webhook", events[i].WebhookID))
		}
	}
}

func (s *Server) eventRetentionWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.cfg.Store.PruneEvents(ctx, s.cfg.EventRetentionRows, s.cfg.EventRetentionAge); err != nil {
				s.cfg.Logger.ErrorContext(ctx, "events: prune", slog.Any("err", err))
			}
		}
	}
}

func (s *Server) deliverWebhookAttempt(ctx context.Context, hook *store.Webhook, ev *store.Event, secret string, body []byte, attempt int) bool {
	statusCode, responseBody, deliveryErr := s.cfg.WebhookDelivery(ctx, hook.URL, secret, body)
	status := outcomeSuccess
	var errMsg *string
	if deliveryErr != nil {
		status = outcomeFailed
		msg := deliveryErr.Error()
		errMsg = &msg
	}
	if _, err := s.cfg.Store.CreateWebhookDelivery(ctx, hook.ID, ev.ID, ev.Type, status, statusCode, responseBody, errMsg, attempt); err != nil {
		s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: record delivery", slog.Any("err", err), slog.String("webhook", hook.ID), slog.String("event", ev.ID))
	}
	return deliveryErr == nil
}

func (s *Server) rescheduleWebhookEvent(ctx context.Context, job *store.OutboxEvent) bool {
	if job.Attempt <= 0 || job.Attempt > len(s.cfg.WebhookRetryDelays) {
		return false
	}
	delay := s.cfg.WebhookRetryDelays[job.Attempt-1]
	if err := s.cfg.Store.MarkOutboxPending(ctx, job.OutboxID, time.Now().UTC().Add(delay)); err != nil {
		s.cfg.Logger.ErrorContext(ctx, "webhook dispatch: reschedule failed event", slog.Any("err", err), slog.String("event", job.ID), slog.String("webhook", job.WebhookID))
		return false
	}
	return true
}

func webhookSubscribesToEvent(patterns []string, eventType string) bool {
	for _, pattern := range patterns {
		switch {
		case pattern == "*" || pattern == eventType:
			return true
		case strings.HasSuffix(pattern, ".*"):
			if strings.HasPrefix(eventType, strings.TrimSuffix(pattern, "*")) {
				return true
			}
		case strings.HasSuffix(pattern, "."):
			if strings.HasPrefix(eventType, pattern) {
				return true
			}
		}
	}
	return false
}
