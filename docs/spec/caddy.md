# Caddy Edge Service Specification

<!-- markdownlint-disable MD029 MD032 -->

Normative behavior for Caddy as Helling edge service (ADR-037).

## Responsibilities

- Terminate TLS for dashboard and API traffic.
- Serve static WebUI assets.
- Proxy API requests to `hellingd` over local Unix socket.
- Inject operational headers that do not alter API payload contracts.

## Listener and Paths

- Default public endpoint: `https://<host>:8006`
- WebUI assets: `/`
- Helling API: `/api/v1/*`
- Incus proxy path: `/api/incus/*`
- Podman proxy path: `/api/podman/*`
- Health passthrough: `/api/v1/health`

## Upstream Transport

- Caddy -> `hellingd` transport uses Unix socket.
- Default upstream socket: `/run/helling/hellingd.sock`
- Socket path and mode must match `docs/spec/config.md`.

## TLS Modes

1. Bootstrap/self-signed mode:

- Used on first boot until ACME is configured.
- Browser warning is expected.

2. Managed ACME mode:

- Caddy obtains/renews certificates automatically.
- Renewal failures must emit warnings and surface in health checks.

## Security Headers

Caddy should enforce baseline response headers:

- `Strict-Transport-Security`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy`

CSP policy is defined by application/security standards and must not break SSE/WebSocket/API paths.

## Reload and Runtime Operations

- Config updates use Caddy reload semantics (no full host reboot).
- Reload must be zero-downtime for established management sessions where possible.
- Failed reload must preserve last-known-good configuration.

## Service Supervision

- Caddy runs as systemd-managed service.
- Required state for healthy system: `caddy` active + `hellingd` active.

## Validation Checks

- `systemctl status caddy` is active.
- `https://<host>:8006/` serves WebUI.
- `/api/v1/health` returns success through Caddy path.
- `/api/incus/*` and `/api/podman/*` proxy paths remain reachable when authorized.
