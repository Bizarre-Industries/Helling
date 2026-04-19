# ADR-014: Authenticated Reverse Proxy Over Handler-Per-Endpoint

> Status: Accepted

## Context

Helling wraps Incus and Podman. The previous architecture implemented individual Go handler functions for each upstream API endpoint (~150 handlers, ~200 planned). Each handler decoded the HTTP request, called the Incus or Podman Go client, re-encoded the response, and returned it. This pattern:

- Duplicated the entire upstream API surface in Go code (~30k+ lines of generated code)
- Required updating Helling every time Incus or Podman added features
- Created a massive OpenAPI spec (~6,000 lines, 307 endpoints) that was mostly restating what Incus/Podman already document
- Added no value beyond auth and audit for proxied resources
- Made the codebase 5x larger than it needed to be

## Decision

Replace per-endpoint handlers with a generic authenticated reverse proxy middleware. hellingd forwards requests to Incus and Podman Unix sockets after:

1. JWT validation
2. RBAC project scoping (maps Helling user to Incus project via `?project=` param)
3. Audit logging (async, to systemd journal)
4. Auto-snapshot hook (before destructive operations)

The proxy middleware is ~200-300 lines of Go. It uses `net/http/httputil.ReverseProxy` to forward requests to Unix sockets.

Helling-specific features that Incus/Podman don't provide keep dedicated handlers (~25 endpoints):

- Auth (login, JWT, TOTP, API tokens)
- Users (PAM CRUD)
- Schedules (systemd timer management)
- Webhooks (HMAC event delivery)
- BMC (bmclib integration)
- K8s (CAPN cluster provisioning)
- System (config, upgrade, diagnostics)
- Host firewall (nftables via nft CLI)

Everything else — instances, containers, storage, networks, images, profiles, projects, cluster, operations, events, metrics, warnings, certificates — is proxied to the upstream socket.

Proxy routes:
- `/api/incus/*` → `/var/lib/incus/unix.socket`
- `/api/podman/*` → `/run/podman/podman.sock`
- `/api/v1/*` → Helling-specific handlers

## Consequences

**Easier:**
- New Incus/Podman features work automatically (zero Helling changes)
- OpenAPI spec shrinks from ~300 endpoints to ~40
- CLI shrinks from ~392 commands to ~15 (users use `incus`/`podman` directly)
- Codebase shrinks by ~80%
- hellingd has 6 Go dependencies instead of 20+
- No type synchronization between Helling Go types and Incus/Podman types

**Harder:**
- Frontend must handle two response formats (Helling envelope for Helling endpoints, native Incus/Podman format for proxied endpoints)
- Can't transform or enrich proxied responses without breaking the transparent proxy (use Helling-specific endpoints for enrichment)
- WebSocket upgrade (console, exec) needs careful proxy handling
- Audit logging must intercept proxy requests without blocking them
