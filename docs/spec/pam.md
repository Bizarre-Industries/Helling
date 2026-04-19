# PAM Integration Specification

Normative PAM contract for Helling v0.1 authentication.

## Scope

- PAM-backed username/password authentication for `POST /api/v1/auth/login`.
- Applies to local and SSSD-backed accounts supported by host PAM stack.

## PAM Service Name

- Service name: `helling`
- Config path: `/etc/pam.d/helling`
- Runtime key: `auth.pam_service` (see `docs/spec/config.md`)

## Required PAM Flow

For login attempts, `hellingd` MUST execute:

1. `pam_start`
2. `pam_authenticate`
3. `pam_acct_mgmt`
4. `pam_end`

Failure in any step yields auth failure response with non-leaky error envelope.

## Minimal PAM Policy Baseline

Example baseline (host-tailored):

```text
auth      required    pam_unix.so
account   required    pam_unix.so
```

Installations using SSSD/LDAP via PAM may extend this file, but Helling behavior still depends on the same PAM result semantics.

## Security and Error Handling

- Helling MUST not log plaintext credentials.
- PAM failures are rate-limited by auth policy:
  - 5 failed attempts / 15 minutes per IP + username.
- Account lock/expiry states are surfaced as auth failure with domain-specific error code.

## Operational Requirements

- Misconfigured or missing `/etc/pam.d/helling` is a startup/runtime health issue.
- Health diagnostics should flag PAM initialization failures explicitly.

## Out of Scope (v0.1)

- Native LDAP/OIDC auth providers in `hellingd`.
- WebAuthn/passkey auth paths.
- Custom PAM conversation UI beyond username/password + optional TOTP challenge flow.
