# Security Standards

Last reviewed: 2026-05-02

This doc describes the security posture Helling commits to. Anything not listed here is either out of scope or pending decision in [docs/spec/architecture.md §10](../spec/architecture.md).

## Threat model (v0.1)

**Assumed adversaries:**

- Unauthenticated attacker on the public network reaching `helling-proxy:443`
- Authenticated low-privilege user trying to escalate
- Local non-root user on the host (with shell access but not in the `helling` or `incus` group)

**Out of scope for v0.1:**

- Adversary with root on the host (game over by definition)
- Adversary with access to `/var/lib/incus/unix.socket` (they already have full Incus control; Helling can't help)
- Adversary with physical access to the host
- Adversary in `incus-admin` group (full Incus power)

## Privilege model

```text
Network (public)
  ↓ TLS
helling-proxy   ← runs as user `helling-proxy`, no group memberships
  ↓ Unix socket /run/helling/api.sock (mode 0660, group helling-proxy)
hellingd        ← runs as user `helling`, group `incus` (NOT incus-admin)
  ↓ Unix socket /var/lib/incus/unix.socket
incusd          ← root daemon
```

Two non-root users, two sockets, no shared filesystem state outside `/var/lib/helling` (owned by `helling:helling`, mode 0750).

The `incus` group grants restricted Incus access — no `incus admin` operations. v0.1 doesn't need them. If a future feature needs `incus-admin`, it moves into a privileged sub-process gated behind the operations API, not by promoting `hellingd` itself.

## Authentication

- **Argon2id for password hashes.** Parameters: `time=3, memory=64 MiB, parallelism=4, saltLen=16, keyLen=32`. Re-tune annually.
- **Session IDs**: 256-bit, generated with `crypto/rand`. Stored in DB as **SHA-256 hash of the session ID** so DB compromise alone doesn't grant active sessions.
- **Cookies**: `HttpOnly; Secure; SameSite=Lax; Path=/`. 7-day max-age, sliding via `last_seen_at`.
- **Login rate limit**: 5 failures per username per 15 minutes, 20 failures per source IP per 15 minutes. Implemented as in-memory token bucket with persistence across restart deferred to v0.2.
- **No password reset in v0.1.** Admin resets via CLI (`helling admin reset-password <user>`).

## Authorization

- Two roles: `user` and `admin`. Stored as boolean `is_admin` on `users`.
- Authorization decisions are explicit at the handler entry point. No global RBAC matrix yet.
- v0.1 does not isolate users from each other on the Incus side — every user sees the same default Incus project. This is documented as a known limitation in the roadmap.

## Input validation

- Every request is validated against the OpenAPI spec by `oapi-codegen/nethttp-middleware` before reaching handlers.
- Server side never trusts client-supplied IDs. Resource ownership is checked on every state-changing request.
- Instance names match the regex in the OpenAPI parameter; rejection is at the middleware layer.

## Output encoding

- All API responses are `application/json; charset=utf-8`. No HTML. No JSONP.
- Error responses never include stack traces, file paths, or internal type names. Map to stable `code` + sanitized `message`.
- Logs may include detailed errors; logs are not user-visible.

## Secrets

- No secrets in environment variables that get printed in `/proc/<pid>/environ`. Use config file with `0640` permissions.
- The session signing key (if we add one) lives in `/var/lib/helling/secrets/session.key`, mode `0600`, owned by `helling`. Rotated by writing a new file; old keys remain valid until expiry.
- TLS keys: `/etc/helling/tls/`, mode `0640`, group `helling-proxy`.
- Never commit secrets. `gitleaks detect` runs in CI and locally via the pre-commit hook.

## Network exposure

- `hellingd` listens **only** on a Unix socket. No TCP listener exists.
- `helling-proxy` is the only network-facing process. It binds `:443` (TLS) and optionally `:80` (HTTP→HTTPS redirect only).
- No outbound network calls from `hellingd` in v0.1 except the Incus socket.

## TLS

- Minimum TLS version: 1.2.
- Cipher suites: Go's defaults. We don't override.
- Go 1.26 enables hybrid post-quantum key exchange by default (`X25519MLKEM768`). We don't disable it.
- HSTS: `Strict-Transport-Security: max-age=31536000; includeSubDomains` once we ship a stable release.

## Dependency hygiene

- Renovate or Dependabot keeps deps current. PRs reviewed before merge.
- `govulncheck` runs in CI on every PR. Fail on known-exploitable findings against our import graph.
- Pinned Go toolchain via `toolchain` directive in `go.mod`.
- No replace directives in production modules.
- Frontend deps audited with `bun audit` on every dependency PR.

## Build and supply chain

- Reproducible builds via `-trimpath` and pinned `LDFLAGS` (date, commit).
- Released binaries are signed (cosign keyless) and attested via SLSA Level 3 once we ship v0.1. Documented in `docs/standards/release.md` (TBD).
- `step-security/harden-runner` is enabled on all GitHub Actions workflows.
- Container images use a pinned digest base, not a tag.

## Logging and audit

- Every state-changing API call writes a row to `operations` (or `audit`, TBD).
- Audit fields: `user_id`, `kind`, `target`, `outcome`, `request_id`, `source_ip`, `timestamp`.
- Logs are written via `log/slog` JSON handler. Operational logs and audit logs share the same destination in v0.1; split in v0.2.

## Reporting vulnerabilities

See [SECURITY.md](../../SECURITY.md) for disclosure process and PGP key.

## Things we explicitly accept (residual risk)

- **Single-host failure domain.** No HA in v0.1. If the host goes down, the platform is down.
- **No tenant isolation.** All authenticated users can manipulate all instances. Documented limitation; gated by future Incus-projects-per-user mapping.
- **No 2FA.** v0.1 is password + session. WebAuthn deferred.
- **No password expiry.** Forcing rotation on a fixed schedule increases password reuse and decreases hygiene. We will add detection for compromised passwords (HIBP) before we add expiry.
- **No per-instance audit trail beyond Helling's view.** Incus's own logs are authoritative for what happened on the Incus side.
