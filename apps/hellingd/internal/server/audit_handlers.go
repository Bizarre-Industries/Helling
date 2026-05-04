package server

import (
	"net/http"
	"strconv"
)

func (s *Server) handleAuditQuery(w http.ResponseWriter, r *http.Request) {
	actor := r.URL.Query().Get("actor")
	action := r.URL.Query().Get("action")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	// v0.1: audit is a no-op. systemd journal integration lands in v0.2.
	_ = actor
	_ = action
	writeJSON(w, http.StatusOK, map[string]any{
		"events": []map[string]any{},
		"total":  0,
		"limit":  limit,
	})
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		writeError(w, http.StatusBadRequest, "bad_request", "format must be json or csv")
		return
	}

	// v0.1: audit export is a no-op.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-export.json")
	writeJSON(w, http.StatusOK, []map[string]any{})
}
