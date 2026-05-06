package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/audit"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

const (
	unknownAuditField   = "unknown"
	anonymousAuditActor = "anonymous"
	anonymousAuditRole  = "none"
	auditRoleAdmin      = "admin"
	auditRoleUser       = "user"
)

func auditID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func defaultAuditEmit(logger *slog.Logger, rec *audit.Record) {
	audit.Emit(logger, rec)
}

func (s *Server) audit(r *http.Request, action, outcome, targetType, targetID, message string) {
	if user, ok := UserFromContext(r.Context()); ok {
		s.auditForUser(r, &user, action, outcome, targetType, targetID, message)
		return
	}
	s.auditForAnonymous(r, action, outcome, targetType, targetID, message)
}

func (s *Server) auditForUser(r *http.Request, user *store.User, action, outcome, targetType, targetID, message string) {
	s.auditForUserWithPolicyReason(r, user, action, outcome, targetType, targetID, "", message)
}

func (s *Server) auditForUserWithPolicyReason(
	r *http.Request,
	user *store.User,
	action,
	outcome,
	targetType,
	targetID,
	policyReason,
	message string,
) {
	actor := unknownAuditField
	actorID := unknownAuditField
	role := unknownAuditField
	if user != nil {
		actor = user.Username
		actorID = strconv.FormatInt(user.ID, 10)
		if user.IsAdmin {
			role = auditRoleAdmin
		} else {
			role = auditRoleUser
		}
	}
	s.auditForActorWithPolicyReason(r, actor, actorID, role, action, outcome, targetType, targetID, policyReason, message)
}

func (s *Server) auditForAnonymous(r *http.Request, action, outcome, targetType, targetID, message string) {
	s.auditForActor(r, anonymousAuditActor, anonymousAuditActor, anonymousAuditRole, action, outcome, targetType, targetID, message)
}

func (s *Server) auditForActor(r *http.Request, actor, actorID, role, action, outcome, targetType, targetID, message string) {
	s.auditForActorWithPolicyReason(r, actor, actorID, role, action, outcome, targetType, targetID, "", message)
}

func (s *Server) auditForActorWithPolicyReason(
	r *http.Request,
	actor,
	actorID,
	role,
	action,
	outcome,
	targetType,
	targetID,
	policyReason,
	message string,
) {
	if actor == "" {
		actor = unknownAuditField
	}
	if actor == anonymousAuditActor {
		actorID = anonymousAuditActor
		role = anonymousAuditRole
	}
	if actorID == "" {
		actorID = unknownAuditField
	}
	if role == "" {
		role = unknownAuditField
	}
	if outcome == outcomeFailed {
		outcome = outcomeFailure
	}
	emit := s.cfg.AuditEmit
	if emit == nil {
		emit = defaultAuditEmit
	}
	emit(s.cfg.Logger, &audit.Record{
		Actor:        actor,
		ActorID:      actorID,
		ActorRole:    role,
		Action:       action,
		Outcome:      outcome,
		TargetType:   targetType,
		TargetID:     targetID,
		PolicyReason: policyReason,
		RequestID:    RequestIDFromContext(r.Context()),
		Method:       r.Method,
		RequestPath:  r.URL.RequestURI(),
		SourceIP:     clientIP(r),
		StatusCode:   auditStatusForOutcome(outcome),
		DurationMS:   auditDurationMS(r),
		JWTID:        jwtIDFromContext(r.Context()),
		UserAgent:    r.UserAgent(),
		Message:      message,
	})
}

func (s *Server) auditIncusProxyMutation(r *http.Request, user *store.User, statusCode int) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	outcome := outcomeSuccess
	if statusCode >= http.StatusBadRequest {
		outcome = outcomeFailure
	}
	actor := unknownAuditField
	actorID := unknownAuditField
	role := unknownAuditField
	if user != nil {
		actor = user.Username
		actorID = strconv.FormatInt(user.ID, 10)
		if user.IsAdmin {
			role = auditRoleAdmin
		} else {
			role = auditRoleUser
		}
	}
	emit := s.cfg.AuditEmit
	if emit == nil {
		emit = defaultAuditEmit
	}
	emit(s.cfg.Logger, &audit.Record{
		Actor:       actor,
		ActorID:     actorID,
		ActorRole:   role,
		Action:      "incus.proxy",
		Outcome:     outcome,
		TargetType:  "incus",
		TargetID:    r.URL.RequestURI(),
		RequestID:   RequestIDFromContext(r.Context()),
		Method:      r.Method,
		RequestPath: r.URL.RequestURI(),
		SourceIP:    clientIP(r),
		StatusCode:  statusCode,
		DurationMS:  auditDurationMS(r),
		JWTID:       jwtIDFromContext(r.Context()),
		UserAgent:   r.UserAgent(),
		Message:     "Incus proxy mutation",
	})
}

func auditStatusForOutcome(outcome string) int {
	switch outcome {
	case outcomeDenied:
		return http.StatusForbidden
	case outcomeFailure, outcomeFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}

func auditDurationMS(r *http.Request) int64 {
	if r == nil {
		return 0
	}
	started, ok := requestStartedFromContext(r.Context())
	if !ok {
		return 0
	}
	return max(time.Since(started).Milliseconds(), 0)
}
