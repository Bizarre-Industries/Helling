package server

import (
	"encoding/json"
	"net/http"
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes management is deferred")
}

func (s *Server) handleCreateK8s(w http.ResponseWriter, r *http.Request) {
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes management is deferred")
}

func (s *Server) handleGetK8s(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes management is deferred")
}

func (s *Server) handleDeleteK8s(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes management is deferred")
}

func (s *Server) handleScaleK8s(w http.ResponseWriter, r *http.Request) {
	var req scaleK8sRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Workers < 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "workers must be >= 0")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes management is deferred")
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "Kubernetes management is deferred")
}
