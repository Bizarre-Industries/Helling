package server

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

func (s *Server) handleListOperations(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}

	status := store.OperationStatus(r.URL.Query().Get("status"))
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "bad_request", "limit must be a positive integer")
			return
		}
		limit = n
	}

	ops, err := s.cfg.Store.ListOperations(r.Context(), user.ID, status, limit)
	if err != nil {
		s.cfg.Logger.Error("list operations", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	out := make([]map[string]any, 0, len(ops))
	for i := range ops {
		out = append(out, toOperationResponse(&ops[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	id := chi.URLParam(r, "id")
	op, err := s.cfg.Store.GetOperation(r.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "operation not found")
			return
		}
		s.cfg.Logger.Error("get operation", slog.String("id", id), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toOperationResponse(&op))
}
