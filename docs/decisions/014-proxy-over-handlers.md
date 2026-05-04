# ADR-014: Authenticated Reverse Proxy Over Handler-Per-Endpoint

> Status: Accepted; amended for v0.1 admin-only raw proxy mode.

## Context

Helling wraps Incus and Podman. The previous architecture implemented individual Go handler functions for each upstream API endpoint (~150 handlers, ~200 planned). Each handler decoded the HTTP request, called the Incus or Podman Go client, re-encoded the response, and returned it. This pattern:

- Duplicated the entire upstream API surface in Go code (~30k+ lines of generated code)
- Required updating Helling every time Incus or Podman added features
- Created a massive OpenAPI spec (~6,000 lines, 307 endpoints) that was mostly restating what Incus/Podman already document
- Added no value beyond auth and audit for proxied resources
- Made the codebase 5x larger than it needed to be

## Decision

Replace per-endpoint handlers with a generic authenticated reverse proxy middleware. v0.1 keeps raw proxy routes admin-only; Incus forwards use the restricted user socket until ADR-024/036 delegated mTLS is implemented. hellingd forwards after:

1. JWT validation
2. Admin-only raw proxy authorization in v0.1
3. Audit logging (async, to systemd journal)
4. Auto-snapshot hook (before destructive operations)

The proxy middleware is ~200-300 lines of Go. It uses `net/http/httputil.ReverseProxy` to forward Incus requests to the restricted Incus user socket in v0.1 and Podman requests to the local Podman Unix socket.

Helling-specific features that Incus/Podman don't provide keep dedicated handlers (approximately 40 endpoints):

- Auth (login, JWT, TOTP, API tokens)
- Users (role/status and Incus trust identity lifecycle)
- Schedules (systemd timer management)
- Webhooks (HMAC event delivery)
- BMC (bmclib integration)
- K8s (k3s bootstrap via cloud-init)
- System (config, upgrade, diagnostics)
- Host firewall (nftables via nft CLI)

Everything else — instances, containers, storage, networks, images, profiles, projects, cluster, operations, metrics, warnings, certificates — is proxied to the upstream API surface.

**Events:** Aggregated via Helling-specific endpoint `/api/v1/events` (Server-Sent Events stream). Upstream Incus `/1.0/events` and Podman `/libpod/events` are sources for aggregation but are not directly proxied.

Proxy routes:

- `/api/incus/*` → restricted Incus user socket in v0.1; delegated mTLS is deferred
- `/api/podman/*` → `/run/podman/podman.sock`
- `/api/v1/*` → Helling-specific handlers

## Consequences

**Easier:**

- New Incus/Podman features work automatically (zero Helling changes)
- OpenAPI spec shrinks from ~300 endpoints to ~40
- CLI shrinks from ~392 commands to ~15 (users use `incus`/`podman` directly)
- Codebase shrinks by ~80%
- hellingd keeps a small, focused dependency set
- No type synchronization between Helling Go types and Incus/Podman types

**Harder:**

- Frontend must handle two response formats (Helling envelope for Helling endpoints, native Incus/Podman format for proxied endpoints)
- Can't transform or enrich proxied responses without breaking the transparent proxy (use Helling-specific endpoints for enrichment)
- WebSocket upgrade (console, exec) needs careful proxy handling
- Audit logging must intercept proxy requests without blocking them
