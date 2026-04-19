# Storage

> Status: Draft

Route: `/storage`

> **Data source (ADR-014):** Incus proxy (`/api/incus/1.0/*`). Responses in native Incus format.

---

## Layout

Sidebar: "Storage" selected. Main panel: pool cards at top, then volume ProTable for selected pool. Secondary tab for Disks. This is one of the few pages using cards (storage pool overview with <=6 pools is an explicit exception).

## API Endpoints

- `GET /api/incus/1.0/storage-pools` -- list pools with usage
- `GET /api/incus/1.0/storage-pools/:name` -- pool detail
- `POST /api/incus/1.0/storage-pools` -- create pool
- `DELETE /api/incus/1.0/storage-pools/:name` -- delete pool
- `GET /api/incus/1.0/storage-pools/:name/volumes` -- volumes in pool
- `POST /api/incus/1.0/storage-pools/:name/volumes` -- create volume
- `PATCH /api/incus/1.0/storage-pools/:name/volumes/:type/:vol` -- resize
- `GET /api/v1/system/disks` -- physical disks + SMART data (Helling API)

## Components

- `Card` -- per pool with `Progress` usage bar (segments colored per instance), type `Tag` (ZFS, LVM, Btrfs, dir, NFS, iSCSI), used/total/free stats
- `StepsForm` -- create pool wizard (type, disks, options, confirm)
- `ProTable` -- volume list per pool (name, format, size, used_by, actions). Actions: resize, clone, snapshot, delete.
- `Slider` -- volume resize within table row
- `Tabs` -- Pools | Disks
- **Disks tab:** `ProTable` of physical disks with SMART health `Badge`. Click row opens `Descriptions` with full SMART attributes. Wipe `Button` (danger).
- `Upload.Dragger` -- upload ISO/image to pool

## Data Model

- Pool: `name`, `driver` (zfs/lvm/btrfs/dir/nfs), `status`, `used`, `total`, `description`, `config{}`
- Volume: `name`, `type`, `size`, `used_by[]`, `content_type`, `created_at`
- Disk: `path`, `model`, `serial`, `size`, `smart_status`, `smart_attributes[]`, `pool` (if assigned)

## States

### Empty State
"No storage pools configured." [Create Pool]. "Storage pools provide disk space for your VMs, containers, and backups."

### Loading State
Cached pool cards shown. Volume list loads on pool selection.

### Error State
Pool degraded (ZFS): red Badge with status (degraded, resilvering). Show scrub results and repair actions. Pool full: warning banner with link to resize or clean up.

## User Actions

- Create pool via StepsForm wizard
- Click pool card to see its volumes
- Volume CRUD: create, resize (slider), clone, snapshot, delete
- Upload ISO/image to pool
- View SMART health per disk, wipe disk (danger action with confirmation)

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Storage section)
- Patterns: docs/design/patterns/empty-states.md
