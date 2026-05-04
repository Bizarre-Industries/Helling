package server

import (
	"encoding/json"
	"net/http"
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC management is deferred")
}

func (s *Server) handleCreateBMC(w http.ResponseWriter, r *http.Request) {
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC management is deferred")
}

func (s *Server) handleGetBMC(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC management is deferred")
}

func (s *Server) handleDeleteBMC(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC management is deferred")
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
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC management is deferred")
}

func (s *Server) handleBMCSEL(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "BMC management is deferred")
}
