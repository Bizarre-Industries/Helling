package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

type createK8sRequest struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	ControlPlanes int    `json:"control_planes"`
	Workers       int    `json:"workers"`
}

type scaleK8sRequest struct {
	Workers int `json:"workers"`
}

type upgradeK8sRequest struct {
	Version string `json:"version"`
}

func (s *Server) handleListK8s(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.cfg.Store.ListK8sClusters(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list k8s", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, clusters)
}

func (s *Server) handleCreateK8s(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no_session", "no session")
		return
	}
	var req createK8sRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Version == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and version are required")
		return
	}
	if req.ControlPlanes < 1 {
		req.ControlPlanes = 1
	}
	if req.Workers < 1 {
		req.Workers = 2
	}

	c, err := s.cfg.Store.CreateK8sCluster(r.Context(), u.ID, req.Name, req.Version, req.ControlPlanes, req.Workers)
	if err != nil {
		s.cfg.Logger.Error("create k8s", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleGetK8s(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	c, err := s.cfg.Store.GetK8sCluster(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "cluster not found")
			return
		}
		s.cfg.Logger.Error("get k8s", slog.String("name", name), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteK8s(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.cfg.Store.DeleteK8sCluster(r.Context(), name); err != nil {
		s.cfg.Logger.Error("delete k8s", slog.String("name", name), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleScaleK8s(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	_, err := s.cfg.Store.GetK8sCluster(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "cluster not found")
			return
		}
		s.cfg.Logger.Error("scale k8s: get", slog.String("name", name), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	var req scaleK8sRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Workers < 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "workers must be >= 0")
		return
	}

	if err := s.cfg.Store.UpdateK8sClusterScale(r.Context(), name, req.Workers); err != nil {
		s.cfg.Logger.Error("scale k8s", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"cluster": name,
		"workers": req.Workers,
		"status":  "scaling",
	})
}

func (s *Server) handleUpgradeK8s(w http.ResponseWriter, r *http.Request) {
	var req upgradeK8sRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Version == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "version is required")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes upgrades are deferred")
}

func (s *Server) handleK8sKubeconfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	c, err := s.cfg.Store.GetK8sCluster(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "cluster not found")
			return
		}
		s.cfg.Logger.Error("k8s kubeconfig: get", slog.String("name", name), slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if c.Kubeconfig == nil {
		writeError(w, http.StatusNotFound, "not_found", "kubeconfig not yet available")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cluster":    name,
		"kubeconfig": *c.Kubeconfig,
	})
}
