# systemd Units Specification

<!-- markdownlint-disable MD032 -->

Normative systemd unit behavior for Helling v0.2.

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

- `CapabilityBoundingSet=CAP_NET_ADMIN` (retained only so the reviewed firewall helper can acquire its file capability)
- `AmbientCapabilities=` (empty)
- `ProtectSystem=strict`
- `ProtectHome=true`
- `PrivateTmp=true`
- `ReadWritePaths=/var/lib/helling /var/log/helling /etc/helling /run/helling /etc/systemd/system`
- `ReadOnlyPaths=/usr/bin`
- `RuntimeDirectory=helling`
- `RuntimeDirectoryMode=0755` so Caddy can traverse to the `0660` socket controlled by group `helling-proxy`
- `NoNewPrivileges=false` only because v0.2 uses the reviewed setuid helper.
  `hellingd` itself still runs without Linux capabilities and without broad
  polkit rights.

Root-level unit access:

- v0.1 install does not grant `hellingd` polkit rights for root-level systemd unit management. Schedule unit management stays deferred until the privileged-helper design from ADR-050 is implemented and reviewed.
- v0.2 schedule support uses `/usr/lib/helling/helling-unit-link` for the narrow install/remove surface. `hellingd` still does not receive broad polkit unit-management rights.
- v0.2 firewall support uses `/usr/lib/helling/helling-firewall` for the
  narrow `nft` mutation surface. `hellingd` still receives no ambient Linux
  capabilities.

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

- Timer: `helling-schedule-<uuid>.timer`
- Service: `helling-schedule-<uuid>.service`

Timer requirements:

- `Persistent=true`
- Explicit `OnCalendar=` expression from public `on_calendar`
- Bound to corresponding `.service`

Service requirements:

- Executes `/usr/bin/helling schedule run --system <id>` for the target
  resource with `HELLING_API=http+unix:///run/helling/api.sock` and
  `HELLING_SCHEDULE_TOKEN_FILE=/etc/helling/schedule-runner.token`. The CLI
  calls the Helling API through the local Unix socket, which uses the Incus
  proxy; units do not shell directly into Incus.
- Emits structured logs and audit records.
- Non-zero exit marks run failure and records warning/event.

## Unit File Paths

- System units directory: `/etc/systemd/system/`
- Generated schedule units are first written under
  `/etc/systemd/system/helling-managed/`, owned `root:helling` mode `0770`.
- The helper links/removes validated `helling-schedule-<uuid>.service|timer`
  files into/from `/etc/systemd/system/`.

## Lifecycle Operations

For schedule CRUD:

- Create -> helper validates and installs timer/service units -> reload/enable
- Update -> helper disables old timer -> installs new timer/service units -> reload
- Delete -> helper disables/removes managed units -> reload

The helper accepts exactly `install <unit>` and `remove <unit>`, where `<unit>`
is a basename matching `helling-schedule-<uuid>.service|timer`.

## Health Expectations

Healthy baseline:

- `hellingd.service` active
- `caddy.service` active
- No failed generated timer/service units for active schedules
