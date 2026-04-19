# Threat Model (v0.1)

<!-- markdownlint-disable MD032 -->

Security threat model baseline for Helling management plane.

## Scope

- In scope: Helling-owned API, edge proxy, auth/session/token handling, config/secrets, control-plane data.
- Out of scope: tenant workload application-layer vulnerabilities inside managed VMs/containers.

## Assets

- Admin/user credentials
- JWT/API tokens
- age identity and encrypted secrets
- Incus delegated-user trust certificates
- Control-plane SQLite state
- Audit/event records

## Trust Boundaries

1. Browser/UI -> Caddy edge service
2. Caddy -> hellingd Unix socket
3. hellingd -> Incus HTTPS loopback (mTLS)
4. hellingd -> Podman Unix socket
5. hellingd -> SQLite/host filesystem

## STRIDE Summary

### Spoofing

Risks:

- credential theft/token replay
- forged upstream identity

Mitigations:

- PAM + JWT + TOTP
- short-lived access tokens + revocable refresh/session state
- per-user mTLS cert identity for Incus proxy path

### Tampering

Risks:

- config/db manipulation
- request/response path alteration

Mitigations:

- file permissions and service hardening
- signed package/update path
- audit trail for mutation operations

### Repudiation

Risks:

- inability to attribute privileged changes

Mitigations:

- request-scoped audit events with user/method/path/time
- immutable journal-oriented audit strategy

### Information Disclosure

Risks:

- leaking secrets/tokens/internal errors

Mitigations:

- secret-at-rest encryption with age
- redaction policy in logs/errors
- role and scope enforcement

### Denial of Service

Risks:

- auth brute force, API flooding, resource exhaustion

Mitigations:

- auth rate limiting and lock windows
- service supervision and health checks
- operational alerts on saturation/error rates

### Elevation of Privilege

Risks:

- bypassing role/scope checks
- privilege escalation via broad service capabilities

Mitigations:

- fixed role matrix (v0.1)
- authorization middleware for Helling-owned paths
- least-privilege systemd capability set

## Assumptions

- Host OS is maintained and receives security updates.
- Incus/Podman upstreams are trusted local components.
- TLS and key material are configured according to standards.

## Residual Risks (v0.1)

- No enterprise IdP/SSO in v0.1.
- Operational misconfiguration risk remains if PAM/Caddy/systemd are altered outside documented contracts.

## Review Cadence

- Revisit threat model for each major release and whenever authn/authz boundaries change.
