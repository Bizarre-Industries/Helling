# Configuration Specification (helling.yaml)

<!-- markdownlint-disable MD060 -->

Normative configuration contract for Helling v0.1 runtime.

## Scope

- Applies to `hellingd` runtime configuration.
- Covers file-based config and environment overrides.
- OpenAPI remains API-contract source of truth; this file is runtime-config source of truth.

## File Location and Ownership

- Primary path: `/etc/helling/helling.yaml`
- Owner: `root:helling`
- Mode: `0640`
- Runtime service user reads directly as the `helling` group member.

## Environment Override Pattern

Any key may be overridden by env var:

- Pattern: `HELLING_<UPPER_SNAKE_PATH>`
- Dot path to env conversion example:
  - `auth.access_ttl_minutes` -> `HELLING_AUTH_ACCESS_TTL_MINUTES`
  - `server.socket_path` -> `HELLING_SOCKET_PATH`

Precedence:

1. Explicit env var
2. `helling.yaml`
3. Built-in default

## Required Keys (v0.1)

| Key                             | Type   | Required | Default                            | Notes                                                                |
| ------------------------------- | ------ | -------- | ---------------------------------- | -------------------------------------------------------------------- |
| `state_dir`                     | string | yes      | `/var/lib/helling`                 | SQLite state directory.                                              |
| `server.socket_path`            | string | yes      | `/run/helling/api.sock`            | Unix socket for edge proxy -> daemon traffic.                        |
| `server.socket_group`           | string | yes      | `helling-proxy`                    | Socket group Caddy is added to during ISO first boot.                |
| `server.socket_mode`            | int    | yes      | `432`                              | Decimal form of `0660`; YAML octal parsing is intentionally avoided. |
| `incus.socket_path`             | string | no       | `/var/lib/incus/unix.socket.user`  | Restricted Incus user socket for the `incus` group.                  |
| `incus.project`                 | string | yes      | `default`                          | Incus project used by v0.1 operations.                               |
| `auth.jwt_signing_key_path`     | string | yes      | `/var/lib/helling/jwt/ed25519.key` | Ed25519 signing key seed, created `0600` on first boot when absent.  |
| `auth.setup_token_path`         | string | yes      | `/etc/helling/setup-token`         | One-time first-admin setup token path, created by ISO first boot.    |
| `auth.access_ttl_minutes`       | int    | yes      | `15`                               | Access token TTL in minutes.                                         |
| `auth.session_ttl_hours`        | int    | yes      | `168`                              | Session cookie TTL in hours.                                         |
| `auth.login_rate_limit_per_15m` | int    | yes      | `5`                                | Failed username attempts before lockout.                             |
| `auth.argon2_time_cost`         | int    | yes      | `3`                                | Argon2id time cost.                                                  |
| `auth.argon2_memory_kib`        | int    | yes      | `65536`                            | Argon2id memory cost.                                                |
| `auth.argon2_parallelism`       | int    | yes      | `4`                                | Argon2id parallelism.                                                |
| `log.level`                     | string | yes      | `info`                             | Allowed values: `debug`, `info`, `warn`, `error`.                    |
| `log.format`                    | string | yes      | `json`                             | Allowed values: `json`, `text`.                                      |

## Validation Rules

- `server.socket_path` MUST be an absolute path.
- `server.socket_group` MUST resolve during daemon startup when non-empty.
- `server.socket_mode` MUST grant no broader access than `0660`.
- `auth.login_rate_limit_per_15m` MUST be > 0.
- `auth.access_ttl_minutes` MUST be > 0.
- `auth.setup_token_path` MUST be non-empty and absolute.
- `auth.argon2_time_cost` MUST be between 1 and 10.
- `auth.argon2_memory_kib` MUST be between 8192 and 262144.
- `auth.argon2_parallelism` MUST be between 1 and 8.

## Change Management

- Hot reload is optional per key; if unsupported, restart is required.
- On invalid config, daemon startup MUST fail with explicit key-level error.
- Config mutations from API/UI MUST preserve this schema and write atomically.
