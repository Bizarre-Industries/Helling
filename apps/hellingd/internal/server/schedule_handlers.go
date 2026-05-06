package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/systemd"
)

const (
	scheduleKindBackup   = "backup"
	scheduleKindSnapshot = "snapshot"
)

type createScheduleRequest struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Target     string `json:"target"`
	OnCalendar string `json:"on_calendar"`
	Enabled    *bool  `json:"enabled"`
}

type updateScheduleRequest struct {
	Name       *string `json:"name"`
	OnCalendar *string `json:"on_calendar"`
	Enabled    *bool   `json:"enabled"`
}

type scheduleResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Target     string     `json:"target"`
	OnCalendar string     `json:"on_calendar"`
	Enabled    bool       `json:"enabled"`
	UnitName   string     `json:"unit_name"`
	LastRunAt  *time.Time `json:"last_run_at,omitempty"`
	NextRunAt  *time.Time `json:"next_run_at,omitempty"`
	LastStatus *string    `json:"last_status,omitempty"`
	LastError  *string    `json:"last_error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	rows, err := s.cfg.Store.ListSchedules(r.Context(), r.URL.Query().Get("kind"))
	if err != nil {
		s.cfg.Logger.Error("list schedules", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]scheduleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, scheduleToResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Kind = strings.TrimSpace(req.Kind)
	req.Target = strings.TrimSpace(req.Target)
	req.OnCalendar = strings.TrimSpace(req.OnCalendar)
	if req.Name == "" || req.Kind == "" || req.Target == "" || req.OnCalendar == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name, kind, target, and on_calendar are required")
		return
	}
	if !validScheduleKind(req.Kind) {
		writeError(w, http.StatusBadRequest, "bad_request", "kind must be backup or snapshot")
		return
	}
	if err := systemd.ValidateScheduleInput(req.Kind, req.Target, req.OnCalendar); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	row, err := s.cfg.Store.CreateSchedule(r.Context(), user.ID, req.Name, req.Kind, req.Target, req.OnCalendar)
	if err != nil {
		s.cfg.Logger.Error("create schedule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if req.Enabled != nil && !*req.Enabled {
		if err := s.cfg.Store.UpdateSchedule(r.Context(), row.ID, row.Name, row.OnCalendar, false); err != nil {
			s.cfg.Logger.Error("create schedule: disable", slog.Any("err", err))
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		row.Enabled = false
	}
	if err := s.installScheduleUnits(r.Context(), &row); err != nil {
		_ = s.cfg.Store.DeleteSchedule(context.WithoutCancel(r.Context()), row.ID)
		s.cfg.Logger.Error("create schedule: install units", slog.Any("err", err), slog.String("schedule", row.ID))
		writeError(w, http.StatusInternalServerError, "internal", "could not install schedule units")
		return
	}
	_, _ = s.emitEvent(r.Context(), "schedule.created", row.ID, map[string]any{"name": row.Name})
	s.audit(r, "schedule.create", outcomeSuccess, "schedule", row.ID, "schedule created")
	writeJSON(w, http.StatusCreated, map[string]any{"data": scheduleToResponse(&row)})
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	row, err := s.cfg.Store.GetSchedule(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("get schedule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": scheduleToResponse(&row)})
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	row, err := s.cfg.Store.GetSchedule(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("update schedule: get", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	var req updateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name != nil {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.OnCalendar != nil {
		row.OnCalendar = strings.TrimSpace(*req.OnCalendar)
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if row.Name == "" || row.OnCalendar == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and on_calendar cannot be empty")
		return
	}
	if err := systemd.ValidateScheduleInput(row.Kind, row.Target, row.OnCalendar); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := s.refreshScheduleUnits(r.Context(), &row); err != nil {
		s.cfg.Logger.Error("update schedule: install units", slog.Any("err", err), slog.String("schedule", row.ID))
		writeError(w, http.StatusInternalServerError, "internal", "could not install schedule units")
		return
	}
	if err := s.cfg.Store.UpdateSchedule(r.Context(), row.ID, row.Name, row.OnCalendar, row.Enabled); err != nil {
		s.cfg.Logger.Error("update schedule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	row, _ = s.cfg.Store.GetSchedule(r.Context(), row.ID)
	_, _ = s.emitEvent(r.Context(), "schedule.updated", row.ID, map[string]any{"name": row.Name})
	s.audit(r, "schedule.update", outcomeSuccess, "schedule", row.ID, "schedule updated")
	writeJSON(w, http.StatusOK, map[string]any{"data": scheduleToResponse(&row)})
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row, err := s.cfg.Store.GetSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("delete schedule: get", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := s.removeScheduleUnits(r.Context(), row.ID); err != nil {
		s.cfg.Logger.Error("delete schedule: remove units", slog.Any("err", err), slog.String("schedule", row.ID))
		writeError(w, http.StatusInternalServerError, "internal", "could not remove schedule units")
		return
	}
	if err := s.cfg.Store.DeleteSchedule(r.Context(), row.ID); err != nil {
		s.cfg.Logger.Error("delete schedule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	_, _ = s.emitEvent(r.Context(), "schedule.deleted", row.ID, nil)
	s.audit(r, "schedule.delete", outcomeSuccess, "schedule", row.ID, "schedule deleted")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	row, err := s.cfg.Store.GetSchedule(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "schedule not found")
			return
		}
		s.cfg.Logger.Error("run schedule: get", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if s.cfg.Incus == nil {
		s.audit(r, "schedule.run", outcomeFailed, "schedule", row.ID, "incus unavailable")
		writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "incus unavailable")
		return
	}
	userID := row.UserID
	if user, ok := UserFromContext(r.Context()); ok {
		userID = user.ID
	} else if !scheduleRunnerFromContext(r.Context()) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	artifactName := scheduleArtifactName(&row, time.Now().UTC())
	op, err := s.runScheduleIncusOperation(r.Context(), &row, artifactName)
	if err != nil {
		s.cfg.Logger.Error("run schedule: submit incus operation", slog.Any("err", err), slog.String("schedule", row.ID))
		s.audit(r, "schedule.run", outcomeFailed, "schedule", row.ID, "schedule run failed")
		writeError(w, http.StatusBadGateway, "incus_error", "could not submit schedule operation")
		return
	}
	dbOp, err := s.cfg.Store.CreateOperation(r.Context(), userID, "schedule."+row.Kind, row.Target, op.ID())
	if err != nil {
		s.cfg.Logger.Error("run schedule: create operation", slog.Any("err", err), slog.String("schedule", row.ID))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := s.cfg.Store.MarkScheduleRunStarted(r.Context(), row.ID); err != nil {
		s.cfg.Logger.Error("run schedule: touch schedule", slog.Any("err", err), slog.String("schedule", row.ID))
	}
	ev, err := s.emitEvent(r.Context(), "schedule.started", row.ID, map[string]any{
		"artifact":           artifactName,
		"incus_operation_id": op.ID(),
		"kind":               row.Kind,
		"operation_id":       dbOp.ID,
		"target":             row.Target,
	})
	if err != nil {
		s.cfg.Logger.Error("run schedule: event", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	s.audit(r, "schedule.run", outcomeSuccess, "schedule", row.ID, "schedule run requested")
	writeJSON(w, http.StatusAccepted, map[string]any{"data": eventToResponse(&ev)})
}

func (s *Server) runScheduleIncusOperation(ctx context.Context, row *store.Schedule, artifactName string) (incus.OperationHandle, error) {
	switch row.Kind {
	case scheduleKindBackup:
		return s.cfg.Incus.CreateInstanceBackup(ctx, row.Target, artifactName)
	case scheduleKindSnapshot:
		return s.cfg.Incus.CreateInstanceSnapshot(ctx, row.Target, artifactName)
	default:
		return nil, fmt.Errorf("unsupported schedule kind %q", row.Kind)
	}
}

func scheduleArtifactName(row *store.Schedule, now time.Time) string {
	return "helling-" + row.ID + "-" + now.UTC().Format("20060102T150405Z")
}

func validScheduleKind(kind string) bool {
	return kind == scheduleKindBackup || kind == scheduleKindSnapshot
}

func (s *Server) installScheduleUnits(ctx context.Context, row *store.Schedule) error {
	if s.cfg.ScheduleUnits == nil {
		return nil
	}
	return s.cfg.ScheduleUnits.Install(ctx, scheduleUnitSpec(row))
}

func (s *Server) refreshScheduleUnits(ctx context.Context, row *store.Schedule) error {
	if s.cfg.ScheduleUnits == nil {
		return nil
	}
	if !row.Enabled {
		if err := s.cfg.ScheduleUnits.Remove(ctx, row.ID); err != nil {
			return err
		}
	}
	return s.cfg.ScheduleUnits.Install(ctx, scheduleUnitSpec(row))
}

func (s *Server) removeScheduleUnits(ctx context.Context, id string) error {
	if s.cfg.ScheduleUnits == nil {
		return nil
	}
	return s.cfg.ScheduleUnits.Remove(ctx, id)
}

func scheduleUnitSpec(row *store.Schedule) systemd.ScheduleSpec {
	return systemd.ScheduleSpec{
		ID:         row.ID,
		Kind:       row.Kind,
		Target:     row.Target,
		OnCalendar: row.OnCalendar,
		Enabled:    row.Enabled,
	}
}

func scheduleToResponse(row *store.Schedule) scheduleResponse {
	return scheduleResponse{
		ID:         row.ID,
		Name:       row.Name,
		Kind:       row.Kind,
		Target:     row.Target,
		OnCalendar: row.OnCalendar,
		Enabled:    row.Enabled,
		UnitName:   row.UnitName,
		LastRunAt:  row.LastRunAt,
		NextRunAt:  row.NextRunAt,
		LastStatus: row.LastStatus,
		LastError:  row.LastError,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
