package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type createNotificationChannelRequest struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

func (s *Server) handleListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	// v0.1: notifications are not yet implemented. Return empty list.
	writeJSON(w, http.StatusOK, []map[string]any{})
}

func (s *Server) handleCreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var req createNotificationChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Type == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and type are required")
		return
	}
	// v0.1: notification delivery lands in v0.3.
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     "nc-placeholder",
		"name":   req.Name,
		"type":   req.Type,
		"status": "created",
	})
}

func (s *Server) handleDeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "channel id required")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "channel id required")
		return
	}
	// v0.1: test delivery is a no-op.
	writeJSON(w, http.StatusAccepted, map[string]any{
		"channel_id": id,
		"status":     "queued",
	})
}
