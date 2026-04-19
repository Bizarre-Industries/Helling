# ADR-027: Two-daemon split (edge service + hellingd)

> Status: Accepted (2026-04-19)
> Implementation updated by ADR-037

## Context

Helling serves static UI/TLS edge concerns and privileged backend operations. Combining both in one process increases blast radius.

## Decision

Keep a two-daemon model:

- Edge service (Caddy): TLS termination and static web serving
- `hellingd`: authenticated API and Unix-socket proxy to Incus/Podman

## Consequences

- Better privilege separation
- Cleaner operational boundaries
- Additional service coordination required during startup and upgrades
