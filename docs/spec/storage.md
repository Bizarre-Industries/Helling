# Storage Specification

All Incus storage operations go through the proxy (ADR-014). hellingd forwards requests from `/api/incus/*` to the Incus Unix socket.

For the full Incus storage API, see [Incus REST API](https://linuxcontainers.org/incus/docs/main/rest-api-spec/).

## What's Available via Proxy

Everything the Incus storage API provides:

- Storage pools: CRUD (ZFS, LVM, Btrfs, dir, Ceph, NFS, iSCSI)
- Storage volumes: CRUD, snapshots, backups
- Pool resources: usage, available, total

Podman volume operations go through the Podman proxy.

## Helling Additions

### Backup Scheduling

Helling manages backup schedules via systemd timers (ADR-017). When a timer fires, it triggers an Incus backup via the proxy.

```
POST /api/v1/schedules → creates systemd timer + service unit
```

The timer's service unit runs:
```
ExecStart=/usr/local/bin/helling schedule run backup <instance-name>
```

Which calls the Incus backup API via the proxy.

Schedule types:
- Instance backup: `incus export` via proxy
- Instance snapshot: `incus snapshot create` via proxy
- Volume snapshot: `incus storage volume snapshot create` via proxy

Retention enforcement: a cleanup timer runs daily, deleting backups/snapshots older than the configured retention period.

### Disk Health (SMART)

Helling reads disk SMART data by shelling out to `smartctl --json` (ADR-018). Displayed in the storage page and system diagnostics.

```
GET /api/v1/system/hardware → includes SMART data per disk
```

### Dashboard

- Pool cards with usage bars
- ZFS pool status (via `zpool status`, shell out)
- LVM details (via `lvs --reportformat json`, shell out)
- SMART health per physical disk
- Volume tables with snapshot counts
