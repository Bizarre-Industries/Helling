# ADR-036: Incus HTTPS loopback transport for delegated-user proxy calls

> Status: Accepted for delegated-user proxy calls; v0.1 admin-only proxy mode defers this transport.

## Context

Helling delegated-user authorization for Incus uses per-user TLS certificate
identities (ADR-024). v0.1 rejects non-admin raw Incus proxy requests until that
transport is wired end to end.

Per-user certificate enforcement requires TLS client authentication against the Incus HTTPS API. The Incus Unix socket does not provide that delegated-user certificate boundary.

## Decision

For the delegated-user Incus proxy path:

1. `hellingd` uses the Incus HTTPS API on loopback (`127.0.0.1:8443`) for non-admin `/api/incus/*` proxy calls.
2. Each delegated user request is forwarded with that caller's per-user client certificate.
3. Incus Unix socket access remains reserved for host administrator CLI operations.
4. Query-parameter project scoping is not an authorization mechanism.

## Consequences

- A single auditable authorization boundary exists for delegated-user Incus calls: certificate identity plus Incus trust restrictions.
- Deployment must ensure `core.https_address=127.0.0.1:8443` is configured.
- Transport behavior is consistent across API, architecture, auth, networking, and storage specifications.
