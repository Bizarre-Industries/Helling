# systemd Units Specification

<!-- markdownlint-disable MD032 -->

Normative systemd unit behavior for Helling v0.1.

## Core Services

### `hellingd.service`

Purpose:

- Run Helling backend daemon and API surface.

Key requirements:

- Starts after network and local filesystem readiness.
- Uses Unix socket `/run/helling/api.sock` and configured runtime paths from `docs/spec/config.md`.
- Auto-restart on failure.

User and group:

- `User=helling`
- `Group=helling`
- `SupplementaryGroups=helling-proxy incus`
- Created at install time with reserved UID (e.g. 999 on Debian) and subuid/subgid ranges for rootless Podman

Hardening baseline:

- `CapabilityBoundingSet=` (empty — no capabilities needed; file access via DAC)
- `AmbientCapabilities=` (empty)
- `ProtectSystem=strict`
- `ProtectHome=true`
- `PrivateTmp=true`
- `ReadWritePaths=/var/lib/helling /var/log/helling /etc/helling /run/helling`
- `ReadOnlyPaths=/usr/bin`
- `RuntimeDirectory=helling`
- `RuntimeDirectoryMode=0755` so Caddy can traverse to the `0660` socket controlled by group `helling-proxy`

Root-level unit access:

- v0.1 install does not grant `hellingd` polkit rights for root-level systemd unit management. Schedule unit management stays deferred until the privileged-helper design from ADR-050 is implemented and reviewed.

### `caddy.service`

Purpose:

- Serve WebUI and proxy API to `hellingd`.

Key requirements:

- Starts after first boot installs `/etc/caddy/Caddyfile`.
- Proxies Helling API paths to `/run/helling/api.sock`.
- Reload-safe configuration updates.
- TLS mode behavior as defined in `docs/spec/caddy.md`.

User and group:

- Debian's packaged `caddy` user is added to group `helling-proxy` by the ISO first-boot service.
- Caddy must not be added to `helling`, `incus`, `incus-admin`, `podman`, or `systemd-journal`.

### `helling-first-boot.service`

Purpose:

- Finish ISO-installed host setup on the installed target system.

Key requirements:

- Creates `helling`, `helling-proxy`, and required supplementary group memberships.
- Creates `/etc/helling`, `/var/lib/helling`, `/var/log/helling`, and `/run/helling` with install-time permissions.
- Creates `/etc/helling/setup-token` for first-admin setup.
- Writes `/etc/helling/helling.yaml` only if missing.
- Initializes Incus with loopback HTTPS.
- Enables and starts `hellingd.service` and `caddy.service`.
- Verifies `/healthz` through the Unix socket and through Caddy.
- Marks completion at `/var/lib/helling/.first-boot-complete` and is idempotent.

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

For future schedule CRUD (deferred until the privileged helper is implemented):

- Create -> helper validates and installs timer/service units -> reload/enable
- Update -> helper disables old timer -> installs new timer/service units -> reload
- Delete -> helper disables/removes managed units -> reload

Note: v0.1 does not install the helper or grant broad unit-management policy; schedule mutation returns explicit deferred behavior until this is implemented.

## Health Expectations

Healthy baseline:

- `hellingd.service` active
- `caddy.service` active
- No failed generated timer/service units for active schedules
