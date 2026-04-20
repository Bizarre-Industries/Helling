# ADR-017: systemd Timers Over In-Process Cron

> Status: Accepted
>
> Amended 2026-04-20 by ADR-050 (hellingd non-root). Root-based assumptions removed; unit management now via DBus + polkit + auditable SUID helper for symlink operations.

## Context

The previous architecture used `go-co-op/gocron/v2` for scheduling backups, snapshots, and other periodic operations. This ran a cron engine inside the hellingd process, requiring:

- Custom Go code for schedule CRUD, persistence, execution, error handling
- SQLite tables for schedule state
- Process restart = lost in-flight schedules
- No visibility into schedule execution outside of Helling's own UI/API
- Another dependency in go.mod

Helling is an OS. systemd is always present. systemd timers are the standard Linux mechanism for scheduled tasks.

## Decision

Backup and snapshot schedules write systemd timer + service units to `/etc/systemd/system/helling-managed/` (a group-writable staging directory owned `root:helling 0750`), then link them into `/etc/systemd/system/` via a small auditable SUID helper and call systemd over DBus to reload/enable.

Example: a daily backup schedule for instance `vm-web-1` creates:

```ini
# /etc/systemd/system/helling-managed/helling-backup-vm-web-1.timer
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
# /etc/systemd/system/helling-managed/helling-backup-vm-web-1.service
[Unit]
Description=Helling backup for vm-web-1

[Service]
Type=oneshot
User=helling
Group=helling
ExecStart=/usr/local/bin/helling schedule run backup vm-web-1
```

The `helling schedule run` command calls the hellingd API, which triggers the Incus backup via the proxy.

### Unit-management flow (non-root hellingd, per ADR-050)

hellingd runs as the `helling` system user. Writing units goes through two staged steps, not a direct write to `/etc/systemd/system/`:

1. **Write unit body** to `/etc/systemd/system/helling-managed/` (owned `root:helling` mode `0750`; hellingd, as a member of `helling`, can write here).
2. **Link into active unit path** via the shipped SUID helper `/usr/lib/helling/helling-unit-link` (mode `4750`, owner `root:helling`). The helper validates the filename matches `^helling-[a-z0-9-]+\.(timer|service)$`, refuses anything else, and calls `systemctl link` + `systemctl daemon-reload` + `systemctl enable --now` over its root-owned DBus connection.
3. **hellingd observes result** over the system DBus (`org.freedesktop.systemd1`) using `godbus/dbus/v5`. Status queries, enabled-state checks, and `systemctl list-timers` equivalents all go through DBus. No shell-out to `systemctl` from hellingd directly.

Polkit ensures hellingd cannot call `ManagerInterface.StartUnit` / `StopUnit` on anything outside the `helling-*` pattern — see ADR-050 for the policy.

Schedule CRUD in hellingd:

- `POST /api/v1/schedules` → writes timer+service unit files to staging, invokes SUID helper to link+enable, tracks result
- `GET /api/v1/schedules` → DBus `ListUnits` filtered by `helling-*.timer` prefix
- `DELETE /api/v1/schedules/{id}` → DBus `StopUnit` + `DisableUnitFiles` + remove-and-unlink via SUID helper
- Status: DBus `GetUnit` + `Unit.Properties`

## Consequences

**Easier:**

- Schedules survive hellingd restarts (systemd manages them independently)
- DBus `ListUnits` returns all `helling-*` timers with metadata in one call (faster than parsing `systemctl list-timers` output)
- `journalctl -u helling-backup-vm-web-1` shows execution history (works under non-root via `systemd-journal` group)
- No gocron dependency, no SQLite schedule tables
- `Persistent=true` catches up on missed runs after reboot
- `RandomizedDelaySec` prevents thundering herd on cluster nodes
- hellingd does not need root; web-facing compromise stays within `helling`-scoped capability (ADR-050)

**Harder:**

- Small SUID helper is new code; must stay ~40 LOC and be fuzzed against directory-traversal and unit-name injection.
- Staging-dir design means two writes per schedule CRUD (body write, then link).
- Testing requires a real systemd + polkit configuration; use Lima VM in CI, not a bare container.
- Uninstall (`apt purge helling`) must clean up both the staging dir and any live symlinks in `/etc/systemd/system/`.

## References

- ADR-050 (hellingd non-root model that this ADR now relies on)
- ADR-018 (shell-out policy — `systemctl` shell-out explicitly NOT used by hellingd per this ADR; the SUID helper is the single exception)
- `docs/spec/systemd-units.md` (normative unit-file templates)
