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

- Mutation operations MUST emit audit records.
- Audit logs are queryable via audit API and system journal tooling.
- Observability and audit streams must correlate via `request_id` where available.

## SLO Reference Baseline

Operational targets are defined in standards, but default tracking includes:

- API availability
- p95 API latency
- backup success rate
- edge and daemon service health
