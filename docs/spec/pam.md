# PAM Integration Specification (Deferred)

Helling v0.1 does not use PAM. It uses Helling-managed local users with
argon2id password hashes, a one-time installer setup token for the first admin,
Ed25519 JWT access tokens, server-side session cookies, TOTP, and API tokens.

This file is retained only as a placeholder for possible future enterprise IAM
work. It is not normative for v0.1.
