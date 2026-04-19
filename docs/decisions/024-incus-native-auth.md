# ADR-024: Incus native auth for per-user resource isolation

> Status: Accepted (2026-04-15)

## Context

Helling needs per-user Incus resource isolation (user A cannot see user B's VMs). Two approaches were considered:

1. **v0.1–v0.4 (current):** JWT → `?project=` query param injection. hellingd reads the JWT `project` claim and appends `?project=<user-project>` to every proxied Incus request. Incus enforces project isolation.

2. **v0.5+:** Incus fine-grained authorization (OpenFGA backend, auth groups, per-user TLS client certificates). Each user gets a dedicated TLS certificate; the proxy presents the user's cert on their behalf.

## Decision

Implement JWT + `?project=` injection for v0.1–v0.4. Plan migration to Incus fine-grained auth in v0.5.

In v0.5, when the user authenticates to Helling, the proxy retrieves (or generates on-demand) a user-specific TLS certificate issued by Helling's internal CA, and presents it to Incus for each proxied request. Incus OpenFGA maps cert identity → auth group → allowed resources. This provides defense-in-depth: even if a JWT is compromised, the per-user cert scope limits blast radius.

## Consequences

- v0.1–v0.4: simple, works today, no Incus config changes required
- v0.5: Incus OpenFGA must be configured; per-user cert issuance adds complexity
- Per-user cert approach is more auditable (Incus logs show per-cert identity)
- Migration path: `?project=` and cert auth can coexist during transition
- No custom authorization engine in Helling — reuse Incus OpenFGA
