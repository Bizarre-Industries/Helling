# ADR-039: age over bespoke AES-256-GCM secret envelope implementation

> Status: Accepted (2026-04-20)

## Context

Helling needs encrypted secret-at-rest storage for control-plane values. The prior direction relied on custom AES-256-GCM envelope code and local key lifecycle logic.

Custom cryptographic plumbing increases operational and implementation risk (nonce handling, envelope format, rotation behavior, compatibility tooling).

## Decision

Use `age` (`filippo.io/age`) as the encryption format/library for Helling-managed secrets.

- Encrypt/decrypt payloads using age recipients/identities
- Store ciphertext in Helling persistence layer
- Keep key material external to application data storage

Age replaces bespoke application-level AES envelope implementation for v0.1+ secret workflows.

## Consequences

**Easier:**

- Reduces custom crypto code surface area
- Uses a focused, well-audited format and Go implementation
- Improves operator interoperability with standard `age` tooling

**Harder:**

- Requires defined key-generation/storage/rotation runbooks
- Requires migration path if any pre-existing custom-encrypted secrets exist
