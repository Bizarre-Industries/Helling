// Package audit emits Helling audit events to journald.
package audit

import (
	"log/slog"
	"strconv"
	"unicode/utf8"

	"github.com/coreos/go-systemd/v22/journal"
)

// Record describes one auditable control-plane action.
type Record struct {
	Actor        string
	ActorID      string
	ActorRole    string
	Action       string
	Outcome      string
	TargetType   string
	TargetID     string
	PolicyReason string
	RequestID    string
	Method       string
	RequestPath  string
	SourceIP     string
	StatusCode   int
	DurationMS   int64
	JWTID        string
	UserAgent    string
	Message      string
}

// Emit writes an audit record to journald and logs failures without blocking the caller.
func Emit(logger *slog.Logger, rec *Record) {
	if rec == nil {
		return
	}
	fields := map[string]string{
		"SYSLOG_IDENTIFIER":   "hellingd",
		"HELLING_AUDIT":       "1",
		"HELLING_ACTOR":       rec.Actor,
		"HELLING_ACTOR_ID":    rec.ActorID,
		"HELLING_ACTOR_ROLE":  rec.ActorRole,
		"HELLING_ROLE":        rec.ActorRole,
		"HELLING_ACTION":      rec.Action,
		"HELLING_OUTCOME":     rec.Outcome,
		"HELLING_TARGET_TYPE": rec.TargetType,
		"HELLING_TARGET_ID":   rec.TargetID,
		"HELLING_REQUEST_ID":  rec.RequestID,
		"HELLING_METHOD":      rec.Method,
		"HELLING_PATH":        rec.RequestPath,
		"HELLING_SOURCE_IP":   rec.SourceIP,
		"HELLING_USER_AGENT":  truncateJournalField(rec.UserAgent, 512),
	}
	if rec.Message == "" {
		rec.Message = rec.Action
	}
	if rec.PolicyReason != "" {
		fields["HELLING_POLICY_REASON"] = rec.PolicyReason
	}
	if rec.StatusCode > 0 {
		fields["HELLING_STATUS"] = strconv.Itoa(rec.StatusCode)
		fields["HELLING_STATUS_CODE"] = strconv.Itoa(rec.StatusCode)
	}
	if rec.DurationMS >= 0 {
		fields["HELLING_DURATION_MS"] = strconv.FormatInt(rec.DurationMS, 10)
	}
	if rec.JWTID != "" {
		fields["HELLING_JWT_ID"] = rec.JWTID
	}
	if err := journal.Send(rec.Message, journal.PriInfo, fields); err != nil && logger != nil {
		logger.Warn("emit audit journal record", slog.Any("err", err), slog.String("priority", strconv.Itoa(int(journal.PriInfo))))
	}
}

func truncateJournalField(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.ValidString(value[:maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}
