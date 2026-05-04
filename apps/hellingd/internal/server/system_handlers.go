package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"
)

type systemInfoResponse struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Kernel    string `json:"kernel"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	Uptime    int64  `json:"uptime_seconds"`
	GoVersion string `json:"go_version"`
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, systemInfoResponse{
		Hostname:  hostname,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Version:   s.cfg.Version.Version,
		Commit:    s.cfg.Version.Commit,
		BuildTime: s.cfg.Version.BuildTime,
		GoVersion: runtime.Version(),
	})
}

func (s *Server) handleSystemHardware(w http.ResponseWriter, r *http.Request) {
	// v0.1: return basic CPU/memory info from the runtime.
	// Full hardware inventory (SMART, NICs, GPUs) via shell-out lands in v0.2.
	writeJSON(w, http.StatusOK, map[string]any{
		"cpu": map[string]any{
			"arch":    runtime.GOARCH,
			"num_cpu": runtime.NumCPU(),
		},
		"memory": map[string]any{
			"go_memstats": "available via /debug/vars in v0.2",
		},
	})
}

func (s *Server) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	// v0.1: return placeholder config. Full config management lands in v0.2.
	writeJSON(w, http.StatusOK, map[string]any{
		"version": s.cfg.Version.Version,
		"auth": map[string]any{
			"session_ttl_hours": s.cfg.Auth.SessionTTL.Hours(),
		},
	})
}

func (s *Server) handleSystemConfigUpdate(w http.ResponseWriter, r *http.Request) {
	// v0.1: config updates not yet supported. Accept the body but return not-implemented.
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	writeError(w, http.StatusNotImplemented, "not_implemented", "config updates not available in v0.1")
}

func (s *Server) handleSystemUpgrade(w http.ResponseWriter, r *http.Request) {
	// v0.1: upgrade check not yet implemented.
	writeJSON(w, http.StatusOK, map[string]any{
		"current_version":  s.cfg.Version.Version,
		"latest_version":   nil,
		"update_available": false,
	})
}

func (s *Server) handleSystemDiagnostics(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// DB check.
	dbErr := s.cfg.Store.DB().PingContext(r.Context())

	// Incus check.
	incusReachable := false
	if s.cfg.IncusProber != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		incusReachable = s.cfg.IncusProber(ctx)
	}

	elapsed := time.Since(start).Milliseconds()

	status := "healthy"
	if dbErr != nil || !incusReachable {
		status = "degraded"
	}

	checks := map[string]any{
		"database": map[string]any{
			"status": boolToStatus(dbErr == nil),
			"error":  errToString(dbErr),
		},
		"incus": map[string]any{
			"status":    boolToStatus(incusReachable),
			"reachable": incusReachable,
		},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      status,
		"checks":      checks,
		"duration_ms": elapsed,
	})
}

func boolToStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "error"
}

func errToString(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
