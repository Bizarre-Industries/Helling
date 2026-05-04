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
	writeError(w, http.StatusNotImplemented, "not_implemented", "notifications are deferred")
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "notifications are deferred")
}

func (s *Server) handleDeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "channel id required")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "notifications are deferred")
}

func (s *Server) handleTestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "channel id required")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "notifications are deferred")
}
