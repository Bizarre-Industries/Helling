# Configuration Specification (helling.yaml)

<!-- markdownlint-disable MD060 -->

Normative configuration contract for Helling v0.1 runtime.

## Scope

- Applies to `hellingd` runtime configuration.
- Covers file-based config and environment overrides.
- OpenAPI remains API-contract source of truth; this file is runtime-config source of truth.

## File Location and Ownership

- Primary path: `/etc/helling/helling.yaml`
- Owner: `root:root`
- Mode: `0600`
- Runtime service user reads via controlled startup path.

## Environment Override Pattern

Any key may be overridden by env var:

- Pattern: `HELLING_<UPPER_SNAKE_PATH>`
- Dot path to env conversion example:
  - `auth.jwt.access_ttl` -> `HELLING_AUTH_JWT_ACCESS_TTL`
  - `listen.socket` -> `HELLING_LISTEN_SOCKET`

Precedence:

1. Explicit env var
2. `helling.yaml`
3. Built-in default

## Required Keys (v0.1)

| Key                              | Type     | Required | Default                              | Notes                                            |
| -------------------------------- | -------- | -------- | ------------------------------------ | ------------------------------------------------ |
| `listen.socket`                  | string   | yes      | `/run/helling/hellingd.sock`         | Unix socket for edge proxy -> daemon traffic     |
| `listen.socket_mode`             | string   | yes      | `0660`                               | Socket permissions                               |
| `incus.https_address`            | string   | yes      | `127.0.0.1:8443`                     | Loopback-only Incus HTTPS endpoint               |
| `incus.admin_cert_path`          | string   | yes      | `/etc/helling/certs/incus-admin.crt` | Admin client cert for Incus trust management     |
| `incus.admin_key_path`           | string   | yes      | `/etc/helling/certs/incus-admin.key` | Admin client key                                 |
| `podman.socket`                  | string   | yes      | `/run/podman/podman.sock`            | Local Podman Unix socket                         |
| `secrets.identity_path`          | string   | yes      | `/etc/helling/age/identity.txt`      | age identity file for secret encryption          |
| `auth.jwt.signing_key_path`      | string   | yes      | `/etc/helling/jwt/ed25519.key`       | Ed25519 signing key                              |
| `auth.jwt.access_ttl`            | duration | yes      | `15m`                                | Access token TTL                                 |
| `auth.jwt.refresh_ttl`           | duration | yes      | `168h`                               | Refresh token TTL (7d)                           |
| `auth.pam_service`               | string   | yes      | `helling`                            | PAM service name                                 |
| `auth.rate_limit.login_attempts` | int      | yes      | `5`                                  | Failed attempts before lockout                   |
| `auth.rate_limit.login_window`   | duration | yes      | `15m`                                | Lock window for failed auth                      |
| `warnings.interval`              | duration | yes      | `5m`                                 | Warning engine interval                          |
| `warnings.pool_full_pct`         | int      | yes      | `85`                                 | Storage pool warning threshold                   |
| `warnings.cert_expiry_days`      | int      | yes      | `30`                                 | Certificate expiry warning threshold             |
| `warnings.backup_age_hours`      | int      | yes      | `48`                                 | Backup staleness warning threshold               |
| `warnings.stopped_instance_days` | int      | yes      | `90`                                 | Long-stopped instance warning threshold          |
| `auto_snapshot.enabled`          | bool     | yes      | `true`                               | Auto-snapshot before destructive operations      |
| `auto_snapshot.retention`        | duration | yes      | `24h`                                | Auto-snapshot retention                          |
| `auto_snapshot.strict_mode`      | bool     | yes      | `true`                               | Block destructive change if snapshot fails       |
| `logging.level`                  | string   | yes      | `info`                               | Allowed values: `debug`, `info`, `warn`, `error` |
| `logging.format`                 | string   | yes      | `json`                               | Allowed values: `json`, `text`                   |

## Validation Rules

- Unknown top-level keys MUST fail startup validation.
- Duration fields MUST use Go duration format (`15m`, `24h`, `168h`).
- `incus.https_address` MUST be loopback-scoped in v0.1.
- `listen.socket` and `podman.socket` MUST be absolute paths.
- `auth.rate_limit.login_attempts` MUST be > 0.
- `warnings.pool_full_pct` MUST be in range `[1,100]`.

## Change Management

- Hot reload is optional per key; if unsupported, restart is required.
- On invalid config, daemon startup MUST fail with explicit key-level error.
- Config mutations from API/UI MUST preserve this schema and write atomically.
