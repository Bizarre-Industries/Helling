# Authentication and Authorization (v0.1)

> Status: Draft

This document defines the only supported auth model for Helling v0.1.

- Authentication: Helling-managed local users + JWT
- Incus authorization boundary: admin-only raw proxy in v0.1; per-user TLS client certificates are deferred
- Roles: fixed `admin` and `user` for v0.1 implementation
- Password verification: argon2id hashes stored in the Helling SQLite store (ADR-030)
- JWT signing: Ed25519 (ADR-031)

Future enterprise IAM features (LDAP, OIDC, custom roles, WebAuthn, policy engines) are tracked in [auth-v0.5.md](auth-v0.5.md) and are out of scope for v0.1.

---

## 1. Scope

### 1.1 In Scope (v0.1)

- Local account authentication via Helling-managed password hashes
- First-admin setup via one-time installer setup token
- JWT access tokens plus revocable server-side session cookies
- Admin/user roles and static permission matrix
- API token creation and revocation
- TOTP-based MFA with recovery codes

### 1.2 Out of Scope (v0.1)

- LDAP or Active Directory realms
- OIDC / SSO provider integration
- Custom roles or per-resource ACL editor
- WebAuthn / passkeys
- Incus OpenFGA integration

---

## 2. Authentication

### 2.1 Local Password Login

hellingd authenticates users against the Helling SQLite store.

- Passwords are stored as argon2id hashes.
- Failed logins are rate-limited to 5 attempts per 15 minutes per username and 20 attempts per 15 minutes per source IP.
- The first admin is created only through `POST /api/v1/auth/setup` while the user table is empty and the caller proves possession of `/etc/helling/setup-token`.

On success, hellingd loads the user's fixed Helling role and creates a session.

### 2.2 JWT And Session Model

JWT requirements are defined by ADR-031 and `docs/standards/security.md`.

- Algorithm: EdDSA (Ed25519)
- Access token TTL: 15 minutes
- Access token storage: memory only
- Session cookie TTL: 7 days (server-side revocable)
- Session cookie storage: `httpOnly`, `Secure`, `SameSite=Lax`
- v0.1 does not expose a separate `/auth/refresh` endpoint. A page reload loses the in-memory access token and returns to login unless a future refresh flow is added.

Required claims:

- `sub` (user id)
- `username`
- `role` (`admin` or `user`, derived from `is_admin`)
- `jti`
- `iat`
- `exp`

### 2.3 MFA (TOTP)

TOTP is supported in v0.1.

- 6 digits, 30 second period, SHA-1 (RFC 6238 compatibility profile)
- 10 single-use recovery codes, stored as argon2id hashes
- After 5 failed MFA attempts, only recovery code login is accepted for that challenge

---

## 3. Incus Authorization Boundary

### 3.1 No Query-Parameter Project Injection

Helling v0.1 does not rely on query parameter project scoping for authorization.

### 3.2 v0.1 Raw Proxy Boundary

v0.1 does not forward non-admin raw Incus proxy requests. Admin raw proxy requests use the restricted Incus user socket so the daemon never requires `incus-admin`.

### 3.3 Deferred Per-User Client Certificates

ADR-024 remains the non-admin delegation target:

- Incus HTTPS listener enabled on loopback (`core.https_address=127.0.0.1:8443`).
- Helling issues per-user client certificates through its internal CA.
- Incus trust restrictions (`restricted=true` and project limits) enforce visibility and action scope.

### 3.4 Certificate Lifecycle

- Issuance: automatic at user creation/provisioning
- Rotation: admin-triggered or automatic by expiry threshold
- Revocation: immediate on user disable/delete
- Storage: encrypted key material in database, never returned to clients

These lifecycle details are deferred with ADR-024.

### 3.5 Trust Certificate Lifecycle Details

See [docs/spec/internal-ca.md](internal-ca.md) for the complete CA lifecycle specification, including:

- CA key type (Ed25519), encryption (age), and rotation strategy
- User certificate validity periods (90 days, auto-renew at 60 days)
- Dual-sign periods during CA rotation
- SQLite storage schema with encryption
- Bootstrap and recovery procedures

Summary for auth scope:

- **Issuance:** automatic at user creation/provisioning
- **Renewal:** automatic renewal triggered at 60 days remaining validity
- **Dual-sign period:** 60 days of overlap between old and new certificates
- **Revocation:** immediate on user disable/delete
- **Storage:** encrypted key material in database (never returned to clients)

---

## 4. Authorization Model (v0.1)

Role mapping is fixed in ADR-032.

| Role    | Core Permissions                                                              |
| ------- | ----------------------------------------------------------------------------- |
| `admin` | Full system and resource management                                           |
| `user`  | Authenticated self-service surfaces; raw host/proxy management denied in v0.1 |

Custom roles and the auditor role are not part of v0.1.

Authorization check order:

1. Validate JWT or API token
2. Resolve user role
3. Enforce endpoint permission matrix
4. For raw Incus/Podman proxy calls, require admin role in v0.1
5. Emit audit log for allow/deny decision

---

## 5. API Tokens

API tokens are optional credentials for automation.

- Format prefix: `helling_`
- Stored as SHA-256 hash only
- Scopes: `read`, `write`, `admin`
- Default expiry: 90 days (max 365)
- Revocation: immediate

Token usage still resolves to the owning user identity for role checks and audit trails.

---

## 6. Security References

- Password hashing standard: ADR-030
- JWT signing standard: ADR-031
- Role model: ADR-032
- Incus auth model: ADR-024
- Baseline controls: `docs/standards/security.md`

---

## 7. v0.5+ Reference

Planned advanced IAM capabilities remain documented in [auth-v0.5.md](auth-v0.5.md). That file is roadmap material only and not normative for v0.1 implementation.
