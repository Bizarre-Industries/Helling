# Observability Specification

Normative logging, metrics, and event observability contract for Helling v0.1.

## Scope

- Structured logs (service and request lifecycle)
- Prometheus metrics surface
- Journal/audit interoperability

## Structured Logging

Required baseline fields:

- `timestamp` (RFC3339)
- `level` (`debug|info|warn|error`)
- `message`
- `request_id` (for request-scoped logs)
- `user` (when authenticated)
- `source_ip` (when request-scoped)

Reserved optional fields:

- `logger`
- `caller`
- `duration_ms`
- `task_id`
- `resource`
- `action`

Rules:

- Field names use `snake_case`.
- Sensitive values (passwords, tokens, private keys, raw secrets) MUST be redacted.
- Authz allow/deny decisions MUST be auditable.

## Metrics Surface

- Endpoint: `/metrics`
- Format: Prometheus exposition

Required metric families (v0.1 baseline):

- `helling_api_requests_total`
- `helling_api_errors_total`
- `helling_api_request_duration_seconds`
- `helling_goroutines`
- `helling_open_connections`
- `helling_db_size_bytes`

Node/resource metrics may include:

- `helling_node_cpu_usage_percent`
- `helling_node_memory_used_bytes`
- `helling_node_memory_total_bytes`
- `helling_storage_pool_used_bytes`
- `helling_storage_pool_total_bytes`

## Label and Cardinality Policy

- Allowed high-level labels: `method`, `path`, `status`.
- Do not use user IDs, token IDs, request IDs, or hostnames with unbounded cardinality as metric labels.

## Audit and Journal Interop

- Mutation operations MUST emit audit records — see `docs/spec/audit.md` for the normative contract.
- Audit records are journald entries with `HELLING_*` indexed fields per ADR-019, emitted via `coreos/go-systemd/v22/journal`.
- Audit logs are queryable via the `/api/v1/audit/*` surface and standard system journal tooling (`journalctl -t hellingd`, `journalctl HELLING_ACTION=auth.login`).
- Observability application logs (this spec) use `snake_case` lowercase field names; audit records (ADR-019) use `HELLING_*` UPPERCASE indexed journal fields. The two streams are distinct but correlate via `request_id` ↔ `HELLING_REQUEST_ID`.
- The hellingd audit query wrapper shells out to `journalctl --output=json` with filter flags translated from Helling DTOs. See `docs/spec/audit.md` §3.

## SLO Reference Baseline

Operational targets are defined in standards, but default tracking includes:

- API availability
- p95 API latency
- backup success rate
- edge and daemon service health

## Cross-references

- `docs/spec/audit.md` — normative audit contract (distinct stream; correlates via `request_id`)
- `docs/spec/proxies.md` — upstream proxy surface observability
- `docs/decisions/019-journal-over-sqlite-audit.md` — journal-field schema for audit emission
