package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

type createWebhookRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

type updateWebhookRequest struct {
	Name    *string   `json:"name"`
	URL     *string   `json:"url"`
	Secret  *string   `json:"secret"`
	Events  *[]string `json:"events"`
	Enabled *bool     `json:"enabled"`
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	webhooks, err := s.cfg.Store.ListWebhooks(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list webhooks", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, webhooks)
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and url are required")
		return
	}
	if len(req.Events) == 0 {
		req.Events = []string{"*"}
	}

	wb, err := s.cfg.Store.CreateWebhook(r.Context(), u.ID, req.Name, req.URL, req.Secret, req.Events)
	if err != nil {
		s.cfg.Logger.Error("create webhook", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, wb)
}

func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wb, err := s.cfg.Store.GetWebhook(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		s.cfg.Logger.Error("get webhook", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	deliveries, _ := s.cfg.Store.ListWebhookDeliveries(r.Context(), id, 20)
	writeJSON(w, http.StatusOK, map[string]any{
		"webhook":    wb,
		"deliveries": deliveries,
	})
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wb, err := s.cfg.Store.GetWebhook(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		s.cfg.Logger.Error("update webhook: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	name := wb.Name
	if req.Name != nil {
		name = *req.Name
	}
	url := wb.URL
	if req.URL != nil {
		url = *req.URL
	}
	secret := wb.Secret
	if req.Secret != nil {
		secret = *req.Secret
	}
	events := wb.Events
	if req.Events != nil {
		events = *req.Events
	}
	enabled := wb.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := s.cfg.Store.UpdateWebhook(r.Context(), id, name, url, secret, events, enabled); err != nil {
		s.cfg.Logger.Error("update webhook", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.cfg.Store.DeleteWebhook(r.Context(), id); err != nil {
		s.cfg.Logger.Error("delete webhook", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wb, err := s.cfg.Store.GetWebhook(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		s.cfg.Logger.Error("test webhook: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	// v0.1: test delivery is a no-op. Actual HTTP delivery lands in v0.2.
	_, _ = s.cfg.Store.CreateWebhookDelivery(r.Context(), id, "test", "pending", nil, nil, nil, 1)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"webhook_id": id,
		"url":        wb.URL,
		"status":     "queued",
	})
}
