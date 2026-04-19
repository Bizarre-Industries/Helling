# ADR-029: Dedicated hellingprox system user

> Status: Accepted (2026-04-19)

## Context

Running both daemons as root expands impact of web-facing compromise.

## Decision

Run the edge service as dedicated low-privilege system user `hellingprox`.

- No shell login
- Minimal filesystem permissions
- Access only to required cert/static/socket paths

## Consequences

- Reduced privilege exposure on edge-facing component
- Requires explicit install-time user/group provisioning
