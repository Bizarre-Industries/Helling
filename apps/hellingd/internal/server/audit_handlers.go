package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)

type auditEventResponse struct {
	ID         string         `json:"id,omitempty"`
	Time       time.Time      `json:"time"`
	Actor      string         `json:"actor,omitempty"`
	Action     string         `json:"action"`
	Outcome    string         `json:"outcome"`
	TargetType string         `json:"target_type,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	Message    string         `json:"message"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

func (s *Server) handleAuditQuery(w http.ResponseWriter, r *http.Request) {
	limit := auditLimit(r, 100, 500)
	rows, err := readAuditJournal(r.Context(), r, limit)
	if err != nil {
		s.cfg.Logger.Warn("audit journal read failed", slog.Any("err", err))
		writeError(w, http.StatusBadGateway, "audit_unavailable", "audit journal unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	limit := auditLimit(r, 500, 5000)
	if err := streamAuditJournal(r.Context(), r, w, limit); err != nil {
		s.cfg.Logger.Warn("audit export stream failed", slog.Any("err", err))
	}
}

func streamAuditJournal(ctx context.Context, r *http.Request, w http.ResponseWriter, limit int) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// #nosec G204 -- journalctl is a fixed binary and args are bounded filters from this endpoint.
	cmd := exec.CommandContext(ctx, "journalctl", auditJournalArgs(r, limit)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeError(w, http.StatusBadGateway, "audit_unavailable", "audit journal unavailable")
		return fmt.Errorf("opening journalctl stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusBadGateway, "audit_unavailable", "audit journal unavailable")
		return fmt.Errorf("starting journalctl: %w", err)
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-export.jsonl")
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	enc := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)
	for scanner.Scan() {
		var raw map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}
		event, ok := auditFromJournal(raw)
		if !ok {
			continue
		}
		if err := enc.Encode(event); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if scanErr != nil {
		return scanErr
	}
	if waitErr != nil {
		return fmt.Errorf("journalctl export: %w", waitErr)
	}
	return nil
}

func auditLimit(r *http.Request, fallback, ceiling int) int {
	limit := fallback
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= ceiling {
			limit = n
		}
	}
	return limit
}

func readAuditJournal(ctx context.Context, r *http.Request, limit int) ([]auditEventResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	args := auditJournalArgs(r, limit)
	// #nosec G204 -- journalctl is a fixed binary and args are bounded filters from this endpoint.
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running journalctl: %w", err)
	}
	return parseAuditJSONLines(out)
}

func auditJournalArgs(r *http.Request, limit int) []string {
	args := []string{"--output=json", "-n", strconv.Itoa(limit), "SYSLOG_IDENTIFIER=hellingd", "HELLING_AUDIT=1"}
	if actor := r.URL.Query().Get("actor"); actor != "" {
		args = append(args, "HELLING_ACTOR="+actor)
	}
	if action := r.URL.Query().Get("action"); action != "" {
		args = append(args, "HELLING_ACTION="+action)
	}
	if outcome := r.URL.Query().Get("outcome"); outcome != "" {
		args = append(args, "HELLING_OUTCOME="+outcome)
	}
	if since := r.URL.Query().Get("since"); since != "" {
		args = append(args, "--since", since)
	}
	if until := r.URL.Query().Get("until"); until != "" {
		args = append(args, "--until", until)
	}
	return args
}

func parseAuditJSONLines(body []byte) ([]auditEventResponse, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var rows []auditEventResponse
	for scanner.Scan() {
		var raw map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}
		event, ok := auditFromJournal(raw)
		if !ok {
			continue
		}
		rows = append(rows, event)
	}
	return rows, scanner.Err()
}

func auditFromJournal(raw map[string]any) (auditEventResponse, bool) {
	action := stringField(raw, "HELLING_ACTION")
	outcome := stringField(raw, "HELLING_OUTCOME")
	if action == "" || !validAuditOutcome(outcome) {
		return auditEventResponse{}, false
	}
	ts := time.Now().UTC()
	if micros, ok := raw["__REALTIME_TIMESTAMP"].(string); ok {
		if n, err := strconv.ParseInt(micros, 10, 64); err == nil {
			ts = time.UnixMicro(n).UTC()
		}
	}
	return auditEventResponse{
		ID:         stringField(raw, "__CURSOR"),
		Time:       ts,
		Actor:      stringField(raw, "HELLING_ACTOR"),
		Action:     action,
		Outcome:    outcome,
		TargetType: stringField(raw, "HELLING_TARGET_TYPE"),
		TargetID:   stringField(raw, "HELLING_TARGET_ID"),
		Message:    stringField(raw, "MESSAGE"),
		Metadata:   raw,
	}, true
}

func stringField(raw map[string]any, key string) string {
	if v, ok := raw[key].(string); ok {
		return v
	}
	return ""
}

func validAuditOutcome(outcome string) bool {
	return outcome == outcomeSuccess || outcome == outcomeFailure || outcome == outcomeDenied
}
