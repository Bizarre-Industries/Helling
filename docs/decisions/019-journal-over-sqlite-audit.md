# ADR-019: systemd Journal Over SQLite for Audit Logging

> Status: Accepted
>
> Amended 2026-04-20: Structured field schema spelled out below; emission via `go-systemd/v22/journal` (no-cgo) per the exception in ADR-018. Read path remains `journalctl --output=json` shell-out.

## Context

The previous architecture maintained an append-only audit log in SQLite and a parallel set of JSON log files in `/var/log/helling/audit/`. This required:

- Custom Go code for audit writes, rotation, retention
- SQLite tables for audit entries
- A query API endpoint with pagination and filtering
- Duplicate data (SQLite + JSON files)
- Custom CSV export logic

Helling is an OS. systemd journal is the standard structured logging facility. It already provides: append-only storage, structured fields, retention policies, filtering, JSON export, and integration with every Linux monitoring tool.

## Decision

Audit logging uses the systemd journal directly as the backend. Every API mutation, auth event, and policy decision is emitted as a structured journal entry with Helling-specific indexed fields.

Emission uses `github.com/coreos/go-systemd/v22/journal` (cgo-free, socket protocol) — see ADR-018 for the narrow-exception scope.

### Structured field schema (normative)

Every Helling audit record emits a `MESSAGE=` plus the following indexed fields. All `HELLING_*` fields are journald-indexed (via `journalctl HELLING_FIELD=value` fast-path):

| Field                   | Type   | Required | Description                                                                                                                                                |
| ----------------------- | ------ | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MESSAGE`               | string | yes      | Human-readable action description (e.g. `"user.login succeeded"`, `"schedule.run_now"`)                                                                    |
| `PRIORITY`              | int    | yes      | syslog 0-7 (typically 5 info / 3 err / 4 warn)                                                                                                             |
| `SYSLOG_IDENTIFIER`     | string | yes      | `hellingd` (always; enables `journalctl -t hellingd`)                                                                                                      |
| `HELLING_ACTION`        | string | yes      | Action type — dot-separated (e.g. `auth.login`, `user.create`, `instance.delete`, `schedule.run_now`, `policy.deny`)                                       |
| `HELLING_ACTOR`         | string | yes      | Username (or `"system"` for machine-initiated events, or `"anonymous"` for pre-auth failures)                                                              |
| `HELLING_ACTOR_ID`      | string | yes      | User ULID (or `"system"` / `"anonymous"`)                                                                                                                  |
| `HELLING_ROLE`          | string | yes      | Actor role **at event time** — `admin`, `user`, or `none` for pre-auth events                                                                              |
| `HELLING_OUTCOME`       | string | yes      | One of `success`, `failure`, `denied`                                                                                                                      |
| `HELLING_REQUEST_ID`    | string | yes      | `X-Request-ID` UUID correlating the HTTP request                                                                                                           |
| `HELLING_SOURCE_IP`     | string | yes      | Client IP (as seen by Caddy, propagated via `X-Forwarded-For` stripping per ADR-037)                                                                       |
| `HELLING_USER_AGENT`    | string | yes      | `User-Agent` header (truncated to 512 bytes)                                                                                                               |
| `HELLING_JWT_ID`        | string | no       | `jti` claim from the JWT, when one was presented (omitted for auth events before JWT issuance)                                                             |
| `HELLING_METHOD`        | string | no       | HTTP method (`GET`/`POST`/`PUT`/`DELETE`) — present for API events                                                                                         |
| `HELLING_PATH`          | string | no       | Request path (e.g. `/api/v1/users/01HABC...`) — present for API events                                                                                     |
| `HELLING_STATUS`        | int    | no       | HTTP response status code — present for API events                                                                                                         |
| `HELLING_DURATION_MS`   | int    | no       | Request latency in milliseconds — present for API events                                                                                                   |
| `HELLING_TARGET_TYPE`   | string | no       | Resource type being acted on (`user`, `instance`, `schedule`, etc.) — present for mutation events                                                          |
| `HELLING_TARGET_ID`     | string | no       | Resource identifier (ULID or name) — present for mutation events                                                                                           |
| `HELLING_BEFORE`        | string | no       | JSON-encoded snapshot of mutated fields **before** the change — present for update mutations; truncated to 4 KB with `"truncated":true` marker if exceeded |
| `HELLING_AFTER`         | string | no       | JSON-encoded snapshot **after** the change — same truncation rule as `HELLING_BEFORE`                                                                      |
| `HELLING_POLICY_REASON` | string | no       | Structured reason for denial (e.g. `rbac.insufficient_role`, `rate_limit.login`) — present when `HELLING_OUTCOME=denied`                                   |

### Emission pattern

```go
import "github.com/coreos/go-systemd/v22/journal"

fields := map[string]string{
    "HELLING_ACTION":      "user.create",
    "HELLING_ACTOR":       actor.Username,
    "HELLING_ACTOR_ID":    actor.ID,
    "HELLING_ROLE":        actor.Role, // role at event time, not current
    "HELLING_OUTCOME":     "success",
    "HELLING_REQUEST_ID":  reqID,
    "HELLING_SOURCE_IP":   clientIP,
    "HELLING_USER_AGENT":  userAgent, // truncated upstream
    "HELLING_METHOD":      "POST",
    "HELLING_PATH":        "/api/v1/users",
    "HELLING_STATUS":      "201",
    "HELLING_DURATION_MS": strconv.Itoa(int(elapsed.Milliseconds())),
    "HELLING_TARGET_TYPE": "user",
    "HELLING_TARGET_ID":   newUser.ID,
    "HELLING_AFTER":       truncateJSON(mustJSON(newUser.PublicFields()), 4096),
    "SYSLOG_IDENTIFIER":   "hellingd",
}
if err := journal.Send("user.create succeeded", journal.PriInfo, fields); err != nil {
    // Fall back to stderr; systemd captures stderr via StandardError=journal.
    log.Printf("journal emit failed: %v", err)
}
```

### Query patterns

Read path is shell-out to `journalctl` (ADR-018):

```bash
# All actions by a specific user in a time window
journalctl -t hellingd \
    HELLING_ACTOR=alice \
    --since "2026-04-01" --until "2026-04-15" \
    --output=json

# All denials
journalctl -t hellingd \
    HELLING_OUTCOME=denied \
    --output=json

# All mutations on a specific resource
journalctl -t hellingd \
    HELLING_TARGET_ID=01HABC... \
    --output=json

# Correlate across the full request
journalctl -t hellingd \
    HELLING_REQUEST_ID=<uuid> \
    --output=json
```

Dashboard API:

- `GET /api/v1/audit?actor=alice&since=2026-04-01&until=2026-04-15&outcome=denied&action=auth.login`
  → hellingd constructs `journalctl` args from filter params, streams JSON output, transforms to Helling envelope.
- `GET /api/v1/audit/export?...` — same filters, streams CSV or JSONL.

### Retention

Retention is configured via `journald.conf` (standard systemd mechanism), not Helling's config. Recommended installer defaults:

```ini
# /etc/systemd/journald.conf.d/99-helling.conf
[Journal]
Storage=persistent
SystemMaxUse=4G
SystemKeepFree=1G
MaxRetentionSec=90d
```

Administrators who need different retention edit the drop-in.

## Consequences

**Easier:**

- No SQLite audit table, no custom rotation, no retention management
- `journalctl` provides filtering, full-text search, time ranges, JSON output for free
- Indexed `HELLING_*` fields give fast point queries without scanning full message bodies
- Audit entries survive hellingd crashes (journal is managed by systemd)
- Standard Linux tooling works: `journalctl`, `systemd-journal-remote`, log forwarding
- Cluster-wide audit: `systemd-journal-remote` can aggregate from multiple nodes
- Syslog-compatible forwarding to external SIEM systems
- Role at event time is captured in `HELLING_ROLE`, so changing a user's role later doesn't rewrite historical audit

**Harder:**

- Journal storage is per-node (no built-in cross-node query without remote journal)
- Query performance for very large time ranges depends on journal indexing; keep admin dashboards scoped to recent windows (default "last 24h") and encourage JSONL export for long-range analysis
- Dashboard audit page requires parsing journal JSON (different format than a simple SQL query)
- Journal retention is configured via `journald.conf` drop-in, not Helling's own config; document this in `docs/spec/config.md` so operators know where to look
- `HELLING_BEFORE`/`HELLING_AFTER` field size is capped at 4 KB per field; mutations touching larger payloads emit a truncation marker and require the caller to log a separate detail event if full context is needed

## Follow-up documents to update when this amendment lands

- `docs/spec/audit.md` — reference this schema as the normative field list; remove any references to a `sqlite.audit_events` table.
- `docs/spec/observability.md` — spell out the `journalctl` query API wrapper in hellingd.
- `docs/design/pages/audit.md` — confirm the `before`/`after` diff column maps to `HELLING_BEFORE`/`HELLING_AFTER` with truncation handling.
- `docs/design/magic.md` Feature 8 (Infrastructure Changelog) — rewrite implementation note to point at the journal with `HELLING_BEFORE`/`HELLING_AFTER`, NOT a SQLite audit table.
- `docs/design/tools-and-frameworks.md` — add `go-systemd/v22/journal` to the hellingd dependency table.

## References

- ADR-018 (shell-out policy — defines the two exceptions, including the journal emission exception used here)
- ADR-050 (hellingd non-root — emission works unchanged under non-root via the journal socket)
