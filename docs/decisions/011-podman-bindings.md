# ADR-011: Proxy to Podman socket, no Go bindings

> Status: Accepted (2026-04-15) — supersedes original ADR-011 (Podman native bindings over Docker client), replaced approach by ADR-014 (proxy over handlers)

## Context

ADR-011 originally chose `containers/podman/v5/pkg/bindings` over `docker/docker` to avoid API version mismatches. This was correct but still required a Go service layer (~500 lines) wrapping every Podman operation.

ADR-014 (proxy-first architecture) generalises the approach: `httputil.ReverseProxy` routes the full upstream API to its Unix socket with no Go wrapping. Incus is already proxied this way. The same pattern applies to Podman.

## Decision

Remove `containers/podman/v5` from go.mod entirely. Route all Podman operations through `httputil.ReverseProxy` to `/run/podman/podman.sock`. The proxy adds JWT validation, RBAC scoping, and audit logging — then forwards the request byte-for-byte to Podman and streams the response back.

WebSocket exec connections (101 Switching Protocols) are handled natively by `httputil.ReverseProxy` with WebSocket support enabled.

## Consequences

- Zero Podman-specific Go code in hellingd
- `containers/podman/v5` removed from go.mod — six Go dependencies total
- Full Podman libpod API available at `/api/podman/*` with no reimplementation
- WebSocket exec (101 upgrades) handled natively — no manual upgrade code
- New Podman API features available immediately without Helling changes
- Frontend consumes native Podman API response format directly (ADR-015)
