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

Backup and snapshot schedules write systemd timer units. hellingd creates/updates/deletes `.timer` and `.service` unit files under `/etc/systemd/system/`, then calls `systemctl daemon-reload` and `systemctl enable --now`.

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

- hellingd needs write access to /etc/systemd/system/ (it runs as root, so this is fine)
- Schedule CRUD requires `systemctl daemon-reload` after writes
- More complex than a single gocron.NewScheduler() call
- Testing requires systemd (use Lima VM in CI, not a bare container)
