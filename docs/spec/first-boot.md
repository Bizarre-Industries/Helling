# First Boot Engineering Sequence

<!-- markdownlint-disable MD029 MD032 -->

Normative first-boot sequence for ISO-installed Helling nodes.

## Objective

Bring a fresh host from installer completion to a functional management plane with:

- `hellingd` and Caddy active
- admin login enabled
- cryptographic material initialized
- Incus delegated-user trust baseline ready

## Ordered Steps

1. Host baseline setup

- Apply hostname and disk selection from installer answers.
- Ensure required packages/services are present.

2. Runtime directories and permissions

- Create `/etc/helling`, `/var/lib/helling`, `/run/helling` with hardened modes.

3. Secrets identity initialization

- Generate age identity if missing.
- Store at configured `secrets.identity_path` with mode `0400`.

4. JWT signing key initialization

- Generate Ed25519 keypair if missing.
- Persist private key at configured signing-key path.

5. Internal CA initialization

- Create Helling client-cert signing CA if missing.
- Persist CA key material encrypted at rest.

6. Incus admin trust bootstrap

- Ensure Incus HTTPS loopback is enabled (`127.0.0.1:8443`).
- Register/validate admin certificate used for trust-management operations.

7. Config materialization

- Write/validate `helling.yaml` from installer defaults and required keys.

8. Service enable/start

- Enable and start `hellingd`.
- Run schema migrations on startup.
- Enable and start Caddy edge service.

9. Health gate

- Verify `/api/v1/health` and edge path health.
- If failed: mark first boot incomplete and keep actionable diagnostics.

## Setup Completion Behavior

- System allows one-time bootstrap admin creation through `POST /api/v1/auth/setup`.
- If at least one admin user exists, setup endpoint is disabled.

## Failure Handling

- Any failed step records structured error logs and surfaces in diagnostics.
- Partial initialization must be resumable and idempotent on next boot/retry.
- Cryptographic material generation must never overwrite valid existing keys by default.

## Post-Boot Invariants

After successful first boot:

- `hellingd` active and serving `/api/v1/*`
- Caddy active and serving WebUI + proxy paths
- config and key paths exist with correct permissions
- setup flow available only when required
