# ADR-027: Two-daemon split (helling-proxy + hellingd)

> Status: Accepted (2026-04-19)

## Context

Helling serves static UI/TLS edge concerns and privileged backend operations. Combining both in one process increases blast radius.

## Decision

Keep a two-daemon model:

- `helling-proxy`: TLS termination and static web serving
- `hellingd`: authenticated API and Unix-socket proxy to Incus/Podman

## Consequences

- Better privilege separation
- Cleaner operational boundaries
- Additional service coordination required during startup and upgrades
