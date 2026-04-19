# ADR-024: Incus per-user TLS auth from v0.1

> Status: Accepted (2026-04-19)

## Context

Helling must enforce per-user isolation for proxied Incus operations from the first release.

The previous stopgap model (JWT claim to project query-parameter injection) creates a split-brain authorization model where Helling middleware decides scope while Incus executes requests. This is brittle and hard to audit.

Incus already supports trust-scoped client certificates and project restrictions, which can be used as the primary enforcement boundary.

## Decision

From v0.1 onward, Helling uses per-user Incus client certificates.

1. Each Helling user has a dedicated Incus client certificate identity.
2. Certificates are issued by an internal Helling CA.
3. The user keypair and certificate are stored encrypted at rest in SQLite.
4. For every proxied Incus call, hellingd presents the calling user's certificate.
5. Incus trust restrictions and project limits enforce resource visibility and allowed operations.

Helling does not use query-parameter project injection as an authorization mechanism.

## Consequences

- v0.1 has a single authorization boundary for Incus calls: Incus certificate identity
- Auditability improves because Incus logs and trust state reflect per-user identities
- hellingd must implement certificate issuance, storage encryption, rotation, and revocation
- User disable/delete must also revoke corresponding Incus trust entries
- Future fine-grained policy systems can be layered later without changing the v0.1 boundary
