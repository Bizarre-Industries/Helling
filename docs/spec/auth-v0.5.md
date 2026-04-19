# Authentication & Authorization (v0.5+ Additions)

> Status: Draft

This document is additive roadmap material for v0.5+ only.

For v0.1 baseline behavior, see [auth.md](auth.md).

## 1. Advanced Identity Realms

### 1.1 LDAP / Active Directory

Planned additions:

- External LDAP/AD realm configuration
- Group-to-role mapping
- Scheduled user/group synchronization
- Realm-aware login path and realm-scoped policy controls

### 1.2 OpenID Connect (OIDC)

Planned additions:

- Authorization Code + PKCE login support
- Claim mapping for username and groups
- Optional auto-provisioning of users on first login
- Provider-specific logout and token refresh behavior

## 2. WebAuthn / Passkeys

Planned additions:

- Passkey registration and login
- Multiple credentials per user
- Credential lifecycle management (rename, revoke)
- Recovery workflow integration with MFA controls

## 3. Custom Role Model

Planned additions:

- Custom roles beyond fixed v0.1 roles
- Granular permission bundles
- Project-scoped role assignment
- Role templates for common operational personas

## 4. Fine-Grained Authorization

Planned additions:

- Resource-level ACL entries
- Conditional policy checks for sensitive actions
- Expanded audit trails for policy decisions

## 5. Enterprise Session Controls

Planned additions:

- Realm-specific session policies
- Enhanced token policies and rotation controls
- Session revocation and risk-based session invalidation

## 6. Rollout Constraints

- All v0.5+ IAM features must preserve compatibility with v0.1 auth contracts where possible.
- Additive changes must not weaken v0.1 security defaults (PAM-backed verification, Ed25519 JWT, argon2id-managed secret storage).
