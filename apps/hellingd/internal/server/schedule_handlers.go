package server

import (
	"encoding/json"
	"net/http"
)

type createScheduleRequest struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	CronExpr string `json:"cron_expr"`
}

type updateScheduleRequest struct {
	Name     *string `json:"name"`
	CronExpr *string `json:"cron_expr"`
	Enabled  *bool   `json:"enabled"`
}

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "schedules are deferred")
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Kind == "" || req.Target == "" || req.CronExpr == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name, kind, target, and cron_expr are required")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "schedules are deferred")
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "schedules are deferred")
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "schedules are deferred")
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "schedules are deferred")
}

func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "schedules are deferred")
}
