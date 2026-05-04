package server

import (
	"encoding/json"
	"net/http"
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "webhooks are deferred")
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "webhooks are deferred")
}

func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "webhooks are deferred")
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "webhooks are deferred")
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "webhooks are deferred")
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "webhooks are deferred")
}
