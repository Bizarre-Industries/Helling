# ADR-041: URI versioning policy for Helling API contracts

> Status: Accepted (2026-04-20)

## Context

Helling serves multiple API surfaces:

- Helling-owned control plane (`/api/v1/*`)
- Incus proxy (`/api/incus/*`)
- Podman proxy (`/api/podman/*`)

Without a strict versioning policy, clients and docs can drift and break compatibility guarantees.

## Decision

Use explicit URI major versioning for Helling-owned endpoints and treat proxied upstream paths as upstream-owned.

- Helling-owned APIs MUST be versioned as `/api/v{major}/...`.
- v0.1 baseline is `/api/v1/*`.
- Breaking changes require a new major path (`/api/v2/*`).
- Non-breaking changes within v1 are additive only.
- Deprecated v1 fields/endpoints MUST have a documented removal window before v2 removal.
- `/api/incus/*` and `/api/podman/*` are passthrough surfaces and follow upstream API compatibility/versioning.

## Consequences

**Easier:**

- Clear compatibility contract for CLI/WebUI and automation clients
- Cleaner OpenAPI lifecycle and changelog discipline
- Explicit separation of Helling-owned vs upstream version ownership

**Harder:**

- Requires deprecation tracking and versioned migration docs
- Adds maintenance overhead when introducing v2+ surfaces
