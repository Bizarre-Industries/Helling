# systemd Units Specification

<!-- markdownlint-disable MD032 -->

Normative systemd unit behavior for Helling v0.1.

## Core Services

### `hellingd.service`

Purpose:

- Run Helling backend daemon and API surface.

Key requirements:

- Starts after network and local filesystem readiness.
- Uses Unix socket and configured runtime paths from `docs/spec/config.md`.
- Auto-restart on failure.

Hardening baseline:

- `CapabilityBoundingSet=CAP_DAC_OVERRIDE`
- `AmbientCapabilities=` (empty)
- `ProtectSystem=strict`
- `ProtectHome=true`
- `PrivateTmp=true`
- `ReadWritePaths=/var/lib/helling /var/log/helling /etc/helling /run/helling`
- `ReadOnlyPaths=/usr/bin`

### `caddy.service`

Purpose:

- Serve WebUI and proxy API to `hellingd`.

Key requirements:

- Starts after `hellingd` socket readiness.
- Reload-safe configuration updates.
- TLS mode behavior as defined in `docs/spec/caddy.md`.

## Schedule Units (ADR-017)

Scheduled operations are represented as paired unit files:

- Timer: `helling-<type>-<resource>.timer`
- Service: `helling-<type>-<resource>.service`

Timer requirements:

- `Persistent=true`
- Explicit `OnCalendar=` expression (5-field cron equivalent mapping)
- Bound to corresponding `.service`

Service requirements:

- Executes `helling` CLI schedule action for target resource.
- Emits structured logs and audit records.
- Non-zero exit marks run failure and records warning/event.

## Unit File Paths

- System units directory: `/etc/systemd/system/`
- Generated schedule units are written and managed under this path.

## Lifecycle Operations

For schedule CRUD:

- Create -> write unit files -> `daemon-reload` -> `enable --now timer`
- Update -> rewrite units -> `daemon-reload` -> restart timer
- Delete -> disable timer -> remove unit files -> `daemon-reload`

## Health Expectations

Healthy baseline:

- `hellingd.service` active
- `caddy.service` active
- No failed generated timer/service units for active schedules
