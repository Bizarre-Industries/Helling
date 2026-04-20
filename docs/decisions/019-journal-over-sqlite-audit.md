# ADR-019: systemd Journal Over SQLite for Audit Logging

> Status: Accepted

## Context

The previous architecture maintained an append-only audit log in SQLite and a parallel set of JSON log files in `/var/log/helling/audit/`. This required:

- Custom Go code for audit writes, rotation, retention
- SQLite tables for audit entries
- A query API endpoint with pagination and filtering
- Duplicate data (SQLite + JSON files)
- Custom CSV export logic

Helling is an OS. systemd journal is the standard structured logging facility. It already provides: append-only storage, structured fields, retention policies, filtering, JSON export, and integration with every Linux monitoring tool.

## Decision

Audit logging uses `slog` with the systemd journal as the backend. Every API mutation is logged as a structured journal entry:

```go
slog.Info("api.mutation",
    "request_id", requestID,
    "user", username,
    "source_ip", clientIP,
    "method", r.Method,
    "path", r.URL.Path,
    "status", statusCode,
    "duration_ms", elapsed.Milliseconds(),
)
```

Query audit logs:

```bash
journalctl -t hellingd --output json-pretty \
    SYSLOG_IDENTIFIER=hellingd \
    --since "2026-04-01" --until "2026-04-15" \
    | jq 'select(.MESSAGE | contains("api.mutation"))'
```

The dashboard audit page queries the journal via hellingd (which shells out to `journalctl` with filters). CSV export pipes journal JSON through a formatter.

Dashboard API:

- `GET /api/v1/audit?user=admin&since=2026-04-01&until=2026-04-15` → hellingd runs `journalctl` with matching filters, returns parsed entries

## Consequences

**Easier:**

- No SQLite audit table, no custom rotation, no retention management
- `journalctl` provides filtering, full-text search, time ranges, JSON output for free
- Audit entries survive hellingd crashes (journal is managed by systemd)
- Standard Linux tooling works: `journalctl`, `systemd-journal-remote`, log forwarding
- Cluster-wide audit: `systemd-journal-remote` can aggregate from multiple nodes
- Syslog-compatible forwarding to external SIEM systems

**Harder:**

- Journal storage is per-node (no built-in cross-node query without remote journal)
- Query performance for large time ranges depends on journal indexing
- Dashboard audit page requires parsing journal JSON (different format than a simple SQL query)
- Journal retention is configured via journald.conf, not Helling's own config

## Audit Field Schema

All audit entries are structured with these standard fields:

| Field               | Type              | Example               | Purpose                                      |
| ------------------- | ----------------- | --------------------- | -------------------------------------------- |
| `MESSAGE`           | string            | `"api.mutation"`      | Log event identifier (e.g., category.action) |
| `PRIORITY`          | int               | `6` (info)            | syslog severity (0-7)                        |
| `SYSLOG_IDENTIFIER` | string            | `"hellingd"`          | Daemon name (systemd-managed)                |
| `REQUEST_ID`        | string            | uuid                  | Correlation ID across logs                   |
| `USER`              | string            | `"alice"`             | Authenticated user or service account        |
| `SOURCE_IP`         | string            | `"10.0.1.50"`         | Client IP (or internal service identifier)   |
| `METHOD`            | string            | `"POST"`              | HTTP verb or RPC method                      |
| `PATH`              | string            | `"/api/v1/instances"` | API resource path                            |
| `STATUS`            | int               | `201`                 | HTTP status or operation result code         |
| `DURATION_MS`       | int               | `45`                  | Operation time in milliseconds               |
| `ERROR`             | string (optional) | `"permission denied"` | Error message if operation failed            |
| `RESOURCE_ID`       | string (optional) | instance name         | Affected resource identifier                 |

Query examples:

```bash
# All mutations by user alice
journalctl -t hellingd --output json-pretty MESSAGE="api.mutation" USER="alice"

# Failed operations
journalctl -t hellingd --output json-pretty ERROR="*"

# Specific resource changes
journalctl -t hellingd --output json-pretty RESOURCE_ID="prod-web-01"
```

Dashboard audit filtering uses these fields with AND/OR logic to support user queries like "show all mutations by user X on resource Y since date Z".
