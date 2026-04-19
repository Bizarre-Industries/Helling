# Authentication and Authorization (v0.1)

> Status: Draft

This document defines the only supported auth model for Helling v0.1.

- Authentication: PAM + JWT
- Incus authorization boundary: per-user TLS client certificates issued by Helling internal CA
- Roles: fixed `admin`, `user`, `auditor` (ADR-032)
- Password hashing: argon2id (ADR-030)
- JWT signing: Ed25519 (ADR-031)

Future enterprise IAM features (LDAP, OIDC, custom roles, WebAuthn, policy engines) are tracked in [auth-v0.5.md](auth-v0.5.md) and are out of scope for v0.1.

---

## 1. Scope

### 1.1 In Scope (v0.1)

- Local account authentication via Linux PAM
- JWT access and refresh token lifecycle
- Per-user Incus TLS certificate issuance and use
- Three fixed roles and static permission matrix
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

### 2.1 PAM Login

hellingd authenticates users through `/etc/pam.d/helling`.

- `pam_authenticate()` verifies credentials.
- `pam_acct_mgmt()` enforces account policy (lock, expiry).
- Failed logins are rate-limited to 5 attempts per 15 minutes per IP and per username.

On success, hellingd loads the user's fixed Helling role and creates a session.

### 2.2 JWT Session Model

JWT requirements are defined by ADR-031 and `docs/standards/security.md`.

- Algorithm: EdDSA (Ed25519)
- Access token TTL: 15 minutes
- Refresh token TTL: 7 days (server-side revocable)
- Access token storage: memory only
- Refresh token storage: `httpOnly`, `Secure`, `SameSite=Strict` cookie

Required claims:

- `sub` (user id)
- `username`
- `role` (`admin`, `user`, `auditor`)
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

### 3.2 Per-User Client Certificates

As defined by ADR-024:

1. On first successful login, hellingd creates or loads a user-specific Incus client certificate.
2. The certificate is signed by Helling's internal CA.
3. The private key is encrypted at rest in SQLite.
4. For proxied Incus requests, hellingd presents the user's certificate.
5. Incus trust restrictions (`restricted=true` and project limits) enforce visibility and action scope.

This makes Incus the enforcement point and keeps auth decisions auditable through certificate identity.

### 3.3 Certificate Lifecycle

- Issuance: automatic on first authenticated use
- Rotation: admin-triggered or automatic by expiry threshold
- Revocation: immediate on user disable/delete
- Storage: encrypted key material in database, never returned to clients

---

## 4. Authorization Model (v0.1)

Role mapping is fixed in ADR-032.

| Role      | Core Permissions                               |
| --------- | ---------------------------------------------- |
| `admin`   | Full system and resource management            |
| `user`    | Manage resources inside assigned project scope |
| `auditor` | Read-only across allowed resources             |

Custom roles are not part of v0.1.

Authorization check order:

1. Validate JWT or API token
2. Resolve user role
3. Enforce endpoint permission matrix
4. For Incus proxy calls, forward using user certificate identity
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
