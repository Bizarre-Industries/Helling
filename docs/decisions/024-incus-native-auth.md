# ADR-024: Incus per-user TLS auth

> Status: Accepted for delegated proxy access; v0.1 defers non-admin proxying until the transport is implemented.

## Context

Helling must enforce per-user isolation for proxied Incus operations before
non-admin users can access the raw Incus proxy.

The previous stopgap model (JWT claim to project query-parameter injection) creates a split-brain authorization model where Helling middleware decides scope while Incus executes requests. This is brittle and hard to audit.

Incus already supports trust-scoped client certificates and project restrictions, which can be used as the primary enforcement boundary.

## Decision

For delegated non-admin Incus proxy access, Helling uses per-user Incus client
certificates. v0.1 keeps raw `/api/incus/*` proxy routes admin-only and uses the
restricted Incus user socket for daemon-owned container operations until this
certificate transport is implemented end to end.

Prerequisite transport requirement:

- Incus HTTPS must be enabled on loopback (`core.https_address=127.0.0.1:8443`) so hellingd can present per-user TLS identities to Incus.

1. Each Helling user has a dedicated Incus client certificate identity.
2. Certificates are issued by an internal Helling CA.
3. The user keypair and certificate are stored encrypted at rest in SQLite.
4. For every proxied Incus call, hellingd presents the calling user's certificate.
5. Incus trust restrictions and project limits enforce resource visibility and allowed operations.

Certificate identity split model:

- Admin certificate identity: used by hellingd for Incus trust administration and project-level management operations.
- Per-user certificate identity: used for delegated user resource operations proxied through `/api/incus/*`.

Issuance timing:

- Per-user certificates are issued during user creation/provisioning (not on first request).

Helling does not use query-parameter project injection as an authorization mechanism.

## Consequences

- Non-admin raw Incus proxy calls remain rejected until Incus certificate
  identity is available end to end.
- Auditability improves because Incus logs and trust state reflect per-user identities
- hellingd must implement certificate issuance, storage encryption, rotation, and revocation
- User disable/delete must also revoke corresponding Incus trust entries
- Future fine-grained policy systems can be layered later without changing the v0.1 boundary

## CA and Certificate Lifecycle

Full details on CA key management, user certificate lifecycle, rotation strategy, and encryption are in [docs/spec/internal-ca.md](../spec/internal-ca.md).

Key points:

- CA Key: Ed25519 (RFC 8037 per ADR-031), encrypted with age per ADR-039
- CA Cert: 5-year validity, self-signed
- User Certs: 90-day validity, auto-renewed at 60 days
- Storage: User keypair + certificate stored encrypted in SQLite
- Rotation: Manual CA rotation with 60-day dual-sign period
