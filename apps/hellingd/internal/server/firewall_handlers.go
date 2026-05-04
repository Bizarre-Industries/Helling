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
	writeError(w, http.StatusNotImplemented, "not_implemented", "firewall management is deferred")
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "firewall management is deferred")
}

func (s *Server) handleDeleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "rule id required")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "firewall management is deferred")
}
