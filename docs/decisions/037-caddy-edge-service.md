# ADR-037: Caddy as edge service (supersedes custom helling-proxy implementation)

> Status: Accepted (2026-04-20)
> Supersedes implementation details in ADR-027 while preserving ADR-027 intent

## Context

Helling needs a low-privilege edge process that terminates TLS, serves static assets, and forwards `/api/*` to `hellingd` over a Unix socket.

A custom Go edge binary duplicates mature edge concerns that Caddy already solves:

- TLS defaults and certificate lifecycle
- ACME automation and renewal
- static file serving
- reverse proxy behavior for HTTP and WebSocket upgrades

The platform requirement is separation of privileges and local socket transport, not a custom edge codebase.

## Decision

Use Caddy as the edge service instead of maintaining a custom `helling-proxy` Go binary.

- Keep two-daemon separation (edge service + `hellingd`)
- Keep Unix socket transport between edge and daemon (ADR-028)
- Keep low-privilege service user model from ADR-029
- Configure Caddy to:
  - terminate TLS
  - serve web assets
  - proxy `/api/*` to `hellingd` Unix socket

`hellingd` remains the policy boundary (authn/authz, proxy dispatch, audit).

## Consequences

**Easier:**

- Removes a custom edge codebase and release burden
- Gets robust TLS/ACME behavior from a mature edge server
- Reduces implementation risk for HTTPS and certificate renewal

**Harder:**

- Adds an external runtime dependency (Caddy package/service)
- Requires config management and reload integration in upgrade flow
- Requires explicit guardrails for local/self-signed deployments (for example `tls internal`)
