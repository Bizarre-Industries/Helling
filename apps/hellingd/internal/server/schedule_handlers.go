package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
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
	kind := r.URL.Query().Get("kind")
	schedules, err := s.cfg.Store.ListSchedules(r.Context(), kind)
	if err != nil {
		s.cfg.Logger.Error("list schedules", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, schedules)
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Kind == "" || req.Target == "" || req.CronExpr == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name, kind, target, and cron_expr are required")
		return
	}

	sch, err := s.cfg.Store.CreateSchedule(r.Context(), u.ID, req.Name, req.Kind, req.Target, req.CronExpr)
	if err != nil {
		s.cfg.Logger.Error("create schedule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, sch)
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sch, err := s.cfg.Store.GetSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("get schedule", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, sch)
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sch, err := s.cfg.Store.GetSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("update schedule: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	name := sch.Name
	if req.Name != nil {
		name = *req.Name
	}
	cronExpr := sch.CronExpr
	if req.CronExpr != nil {
		cronExpr = *req.CronExpr
	}
	enabled := sch.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := s.cfg.Store.UpdateSchedule(r.Context(), id, name, cronExpr, enabled); err != nil {
		s.cfg.Logger.Error("update schedule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.cfg.Store.DeleteSchedule(r.Context(), id); err != nil {
		s.cfg.Logger.Error("delete schedule", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.cfg.Store.GetSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("run schedule: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	// v0.1: trigger is a no-op; actual systemd timer triggering lands in v0.2.
	writeJSON(w, http.StatusAccepted, map[string]any{
		"schedule_id": id,
		"status":      "triggered",
	})
}
