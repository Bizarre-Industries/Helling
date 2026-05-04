package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

type createBMCRequest struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
}

type bmcPowerRequest struct {
	Action string `json:"action"`
}

func (s *Server) handleListBMC(w http.ResponseWriter, r *http.Request) {
	endpoints, err := s.cfg.Store.ListBMCEndpoints(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list bmc", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (s *Server) handleCreateBMC(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	var req createBMCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Address == "" || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name, address, username, and password are required")
		return
	}
	if req.Port == 0 {
		req.Port = 623
	}
	if req.Protocol == "" {
		req.Protocol = "ipmi"
	}

	b, err := s.cfg.Store.CreateBMCEndpoint(r.Context(), u.ID, req.Name, req.Address, req.Port, req.Username, req.Password, req.Protocol)
	if err != nil {
		s.cfg.Logger.Error("create bmc", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleGetBMC(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := s.cfg.Store.GetBMCEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "BMC endpoint not found")
			return
		}
		s.cfg.Logger.Error("get bmc", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleDeleteBMC(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.cfg.Store.DeleteBMCEndpoint(r.Context(), id); err != nil {
		s.cfg.Logger.Error("delete bmc", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBMCPower(w http.ResponseWriter, r *http.Request) {
	var req bmcPowerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Action == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "action is required (on, off, cycle, reset)")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC power control is deferred")
}

func (s *Server) handleBMCSensors(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.cfg.Store.GetBMCEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "BMC endpoint not found")
			return
		}
		s.cfg.Logger.Error("bmc sensors: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	// v0.1: sensor reading is a no-op.
	writeJSON(w, http.StatusOK, []map[string]any{})
}

func (s *Server) handleBMCSEL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.cfg.Store.GetBMCEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "BMC endpoint not found")
			return
		}
		s.cfg.Logger.Error("bmc sel: get", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	// v0.1: SEL reading is a no-op.
	writeJSON(w, http.StatusOK, []map[string]any{})
}
