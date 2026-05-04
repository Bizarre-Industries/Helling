# ADR-029: Dedicated hellingprox system user

> Status: Superseded by ADR-037. v0.1 uses packaged Caddy as the edge service and keeps `helling-proxy` only as the Unix-socket group.

## Context

Running both daemons as root expands impact of web-facing compromise.

## Decision

Historical decision: run the custom Go edge service as dedicated low-privilege system user `hellingprox`.

- No shell login
- Minimal filesystem permissions
- Access only to required cert/static/socket paths

## Consequences

- Reduced privilege exposure on edge-facing component
- Requires explicit install-time user/group provisioning
