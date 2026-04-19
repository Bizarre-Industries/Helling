# ADR-004: Podman required on host, proxied via Unix socket

> Status: Accepted (updated 2026-04-15)

## Context

Considered running Podman inside Incus containers for isolation. Nesting containers inside Incus breaks compose, pods, and systemd integration.

ISO-only deployment (ADR-021) means every Helling system always has Podman installed. There is no "optional" or "graceful degradation" path.

## Decision

Podman runs directly on the host alongside Incus and is a hard dependency. hellingd proxies the full Podman libpod API via `httputil.ReverseProxy` to the Podman Unix socket at `/run/podman/podman.sock`. No Go bindings (`containers/podman/v5`) — HTTP proxy only.

## Consequences

- Compose stacks work correctly
- Podman pods work correctly
- Systemd integration (generate unit files) works
- No isolation between Podman and host (acceptable for admin-managed infra)
- Podman socket is always present (ISO guarantees it — no graceful degradation needed)
- Zero Podman-specific Go code; `containers/podman/v5` not in go.mod
- WebSocket exec (101 upgrades) handled natively by `httputil.ReverseProxy`
- Two container runtimes on same host (Incus LXC + Podman OCI)
