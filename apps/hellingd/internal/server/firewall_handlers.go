package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type createFirewallRuleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	Source      string `json:"source"`
	SourcePort  string `json:"source_port"`
	Destination string `json:"destination"`
	DestPort    string `json:"dest_port"`
}

func (s *Server) handleListFirewallRules(w http.ResponseWriter, r *http.Request) {
	// v0.1: firewall rules are not yet implemented. Return empty list.
	writeJSON(w, http.StatusOK, []map[string]any{})
}

func (s *Server) handleCreateFirewallRule(w http.ResponseWriter, r *http.Request) {
	var req createFirewallRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Action == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and action are required")
		return
	}
	// v0.1: nftables integration lands in v0.2.
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     "fw-placeholder",
		"name":   req.Name,
		"status": "staged",
	})
}

func (s *Server) handleDeleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "rule id required")
		return
	}
	// v0.1: nftables integration lands in v0.2.
	w.WriteHeader(http.StatusNoContent)
}
