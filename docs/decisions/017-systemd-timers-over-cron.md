# ADR-017: systemd Timers Over In-Process Cron

> Status: Accepted

## Context

The previous architecture used `go-co-op/gocron/v2` for scheduling backups, snapshots, and other periodic operations. This ran a cron engine inside the hellingd process, requiring:

- Custom Go code for schedule CRUD, persistence, execution, error handling
- SQLite tables for schedule state
- Process restart = lost in-flight schedules
- No visibility into schedule execution outside of Helling's own UI/API
- Another dependency in go.mod

Helling is an OS. systemd is always present. systemd timers are the standard Linux mechanism for scheduled tasks.

## Decision

Backup and snapshot schedules are managed as systemd timer units. Per ADR-050, hellingd runs as the non-root `helling` system user and manages schedules via the systemd DBus API (`org.freedesktop.systemd1.Manager`) instead of direct file writes. A polkit rule restricts the `helling` user to `helling-*` units only.

**Implementation:** hellingd emits DBus method calls:

- `StartTransientUnit` — for one-shot schedule execution
- `EnableUnitFiles` — to add schedule timers to the system
- `DisableUnitFiles` — to remove them
- `Reload` — equivalent to `systemctl daemon-reload`

Example: a daily backup schedule for instance `vm-web-1` creates:

```ini
# /etc/systemd/system/helling-backup-vm-web-1.timer
[Unit]
Description=Helling backup for vm-web-1

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/helling-backup-vm-web-1.service
[Unit]
Description=Helling backup for vm-web-1

[Service]
Type=oneshot
ExecStart=/usr/local/bin/helling schedule run backup vm-web-1
```

The `helling schedule run` command calls the hellingd API, which triggers the Incus backup via the proxy.

Schedule CRUD in hellingd:

- `POST /api/v1/schedules` → writes timer+service unit files, enables timer
- `GET /api/v1/schedules` → lists `helling-*.timer` units via `systemctl list-timers`
- `DELETE /api/v1/schedules/{id}` → stops and removes timer+service units
- Status: `systemctl status helling-backup-vm-web-1.timer`

## Consequences

**Easier:**

- Schedules survive hellingd restarts (systemd manages them independently)
- `systemctl list-timers` shows all schedules (standard Linux tooling)
- `journalctl -u helling-backup-vm-web-1` shows execution history
- No gocron dependency, no SQLite schedule tables
- Persistent=true catches up on missed runs after reboot
- RandomizedDelaySec prevents thundering herd on cluster nodes

**Harder:**

- DBus interaction with systemd API (tested; robust; requires `github.com/godbus/dbus/v5` or equivalent)
- Polkit rule must be installed and configured correctly
- More complex than a single gocron.NewScheduler() call
- Testing requires systemd (use Lima VM in CI, not a bare container)
- Transient units are temporary; long-term schedule state is persisted in SQLite and re-created at hellingd startup

## ADR-050 Integration

This ADR was revised when ADR-050 (hellingd non-root user) was accepted. Direct file writes to `/etc/systemd/system/` are replaced with DBus method calls that respect the polkit-enforced authorization model. The consequences section above reflects this implementation.
