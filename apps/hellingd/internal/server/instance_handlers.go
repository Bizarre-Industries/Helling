package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// instanceCreateRequest mirrors components.schemas.InstanceCreate in
// api/openapi.yaml.
type instanceCreateRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Image string `json:"image"`
	Start bool   `json:"start"`
}

type instanceStopRequest struct {
	Force          bool `json:"force"`
	TimeoutSeconds int  `json:"timeout_seconds"`
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Incus == nil {
		writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "Incus client not configured")
		return
	}

	want := strings.ToLower(r.URL.Query().Get("status"))
	raw, err := s.cfg.Incus.ListInstances(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list instances", slog.Any("err", err))
		writeError(w, http.StatusBadGateway, "incus_error", "could not list instances")
		return
	}
	out := make([]incus.Instance, 0, len(raw))
	for i := range raw {
		shaped := incus.ToInstance(&raw[i])
		if want != "" && shaped.Status != want {
			continue
		}
		out = append(out, shaped)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Incus == nil {
		writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "Incus client not configured")
		return
	}
	name := chi.URLParam(r, "name")
	inst, err := s.cfg.Incus.GetInstance(r.Context(), name)
	if err != nil {
		s.cfg.Logger.Error("get instance", slog.String("name", name), slog.Any("err", err))
		writeError(w, http.StatusNotFound, "not_found", "instance not found")
		return
	}
	writeJSON(w, http.StatusOK, incus.ToInstance(inst))
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Incus == nil {
		writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "Incus client not configured")
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}

	var req instanceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Image == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and image are required")
		return
	}
	if req.Type == "" {
		req.Type = "container"
	}

	post := incus.InstanceCreate{
		Name:  req.Name,
		Type:  req.Type,
		Image: req.Image,
		Start: req.Start,
	}
	op, err := s.cfg.Incus.CreateInstance(r.Context(), post)
	if err != nil {
		s.cfg.Logger.Error("incus create instance", slog.String("name", req.Name), slog.Any("err", err))
		if isAlreadyExists(err) {
			writeError(w, http.StatusConflict, "conflict", "instance already exists")
			return
		}
		writeError(w, http.StatusBadGateway, "incus_error", "could not create instance")
		return
	}

	dbOp, err := s.cfg.Store.CreateOperation(r.Context(), user.ID, "instance.create", req.Name, op.ID())
	if err != nil {
		s.cfg.Logger.Error("create operation row", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusAccepted, toOperationResponse(&dbOp))
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Incus == nil {
		writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "Incus client not configured")
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	name := chi.URLParam(r, "name")

	op, err := s.cfg.Incus.DeleteInstance(r.Context(), name)
	if err != nil {
		s.cfg.Logger.Error("incus delete instance", slog.String("name", name), slog.Any("err", err))
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "instance not found")
			return
		}
		writeError(w, http.StatusBadGateway, "incus_error", "could not delete instance")
		return
	}

	dbOp, err := s.cfg.Store.CreateOperation(r.Context(), user.ID, "instance.delete", name, op.ID())
	if err != nil {
		s.cfg.Logger.Error("create operation row", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusAccepted, toOperationResponse(&dbOp))
}

func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request) {
	s.handleInstanceStateChange(w, r, "start", false, 0, "instance.start")
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	var req instanceStopRequest
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 30
	}
	s.handleInstanceStateChange(w, r, "stop", req.Force, req.TimeoutSeconds, "instance.stop")
}

func (s *Server) handleInstanceStateChange(w http.ResponseWriter, r *http.Request, action string, force bool, timeoutSec int, kind string) {
	if s.cfg.Incus == nil {
		writeError(w, http.StatusServiceUnavailable, "incus_unavailable", "Incus client not configured")
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	name := chi.URLParam(r, "name")

	op, err := s.cfg.Incus.UpdateInstanceState(r.Context(), name, action, force, timeoutSec)
	if err != nil {
		s.cfg.Logger.Error("incus update state",
			slog.String("name", name),
			slog.String("action", action),
			slog.Any("err", err),
		)
		switch {
		case isNotFound(err):
			writeError(w, http.StatusNotFound, "not_found", "instance not found")
		case isConflict(err):
			writeError(w, http.StatusConflict, "conflict", "instance state conflict")
		default:
			writeError(w, http.StatusBadGateway, "incus_error", "could not change instance state")
		}
		return
	}

	dbOp, err := s.cfg.Store.CreateOperation(r.Context(), user.ID, kind, name, op.ID())
	if err != nil {
		s.cfg.Logger.Error("create operation row", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusAccepted, toOperationResponse(&dbOp))
}

// toOperationResponse converts a store.Operation into the JSON shape from
// components.schemas.Operation in api/openapi.yaml.
func toOperationResponse(op *store.Operation) map[string]any {
	out := map[string]any{
		"id":         op.ID,
		"kind":       op.Kind,
		"target":     op.Target,
		"status":     string(op.Status),
		"created_at": op.CreatedAt,
		"updated_at": op.UpdatedAt,
	}
	if op.Error != "" {
		out["error"] = op.Error
	}
	return out
}

// Incus error helpers — string match against the upstream client's wrapped errors.
// The Incus client does not export typed sentinels for these conditions.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists")
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "conflict") ||
		strings.Contains(msg, "already running") ||
		strings.Contains(msg, "already stopped") ||
		errors.Is(err, store.ErrNotFound)
}
