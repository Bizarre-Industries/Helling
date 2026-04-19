# Instances

> Status: Draft

Route: `/instances` (list) + `/instances/:name` (detail)

> **Data source (ADR-014):** Incus proxy (`/api/incus/1.0/*`). Responses in native Incus format.

---

## Layout

Sidebar: node group expanded, VMs/CTs visible in resource tree. List view: full-width ProTable. Detail view: breadcrumb + 8 Tabs. Task log drawer at bottom.

## API Endpoints

- `GET /api/incus/1.0/instances` -- list with status, metrics
- `GET /api/incus/1.0/instances/:name` -- full detail
- `GET /api/incus/1.0/instances/:name/state` -- live CPU/RAM/disk/net
- `POST /api/incus/1.0/instances` -- create
- `PUT /api/incus/1.0/instances/:name/state` -- start/stop/restart/freeze
- `DELETE /api/incus/1.0/instances/:name` -- delete
- `POST /api/incus/1.0/instances/:name/snapshots` -- take snapshot
- `POST /api/incus/1.0/instances/:name/backups` -- backup now
- `GET /api/incus/1.0/instances/:name/console` -- WebSocket (SPICE/serial)
- `GET /api/incus/1.0/instances/:name/exec` -- WebSocket (exec)
- `GET /api/incus/1.0/instances/:name/files` -- file browser
- `GET /api/incus/1.0/events` -- live status updates (SSE)

## Components

### List (`/instances`)
- `ProTable` -- columns: status Badge, name, type Tag (VM/CT), CPU%, RAM%, IPs (monospace), node, tags. `rowSelection` for bulk. `search` filterType light. `options` density toggle.
- `Button` (primary) -- "Create Instance" in toolbar
- `StepsForm` -- create wizard (Type, Image, Hardware, Network, Cloud-Init, Review)
- Inline row actions: Start/Stop/Console buttons per row

### Detail (`/instances/:name`) -- 8 Tabs

**Summary tab:** `Descriptions` (status, uptime, vCPUs, RAM, disks, IPs, MACs, tags, notes). `Progress` gauges for CPU/RAM/disk. `Button.Group` quick actions (Start, Stop, Restart, Console, Snapshot, Backup).

**Console tab:** SPICE VGA console (spice-js, dynamic import, ADR-010) for VMs with Ctrl+Alt+Del Button, clipboard, fullscreen. Serial console (xterm.js) for CTs. `Segmented` to switch SPICE/serial.

**Hardware tab:** `ProTable` of devices (CPU, RAM, disks, NICs, USB, PCI, GPU). Add/Edit/Detach per row. Disk resize `Slider`. GPU passthrough with IOMMU group display.

**Snapshots tab:** `ProTable` (name, date, includes RAM, size). `ModalForm` for Take Snapshot. Rollback/Delete per row. Optional `Timeline` visualization.

**Backup tab:** `ProTable` of backups for this instance. `ModalForm` for Backup Now (compression, target). Restore/Verify per row. Schedule link.

**Firewall tab:** `ProTable` of per-instance rules (direction, action, protocol, port, source). `ModalForm` for Add Rule. `Select` for Security Group. `Switch` enable/disable.

**Guest tab:** `Descriptions` (filesystems, disk usage from guest agent). Buttons: Reset Password, Inject SSH Key, Sysprep. Only rendered when guest agent available.

**Options tab:** Boot order (drag-and-drop List via dnd-kit). `Switch` for autostart, protection. Cloud-init editor (Monaco, dynamic import) with YAML toggle. `Select mode="multiple"` for profiles. Hookscript assignment.

## Data Model

- Instance: `name`, `type` (vm/ct), `status`, `architecture`, `profiles[]`, `config{}`, `devices{}`, `created_at`
- State: `cpu_usage`, `memory_usage`, `memory_total`, `disk_usage{}`, `network{}`, `pid`, `uptime`
- Snapshot: `name`, `created_at`, `stateful`, `size`, `description`
- Backup: `name`, `created_at`, `size`, `compression`, `status`

## States

### Empty State
"No virtual machines or system containers yet." [Create Instance] [Deploy from Template]. "New to Helling? Create your first VM in 60 seconds. [Quick Start]"

### Loading State
Cached data shown immediately. SSE pushes status changes. No skeleton for repeat visits.

### Error State
Incus unavailable: banner "Incus service is unavailable. VM and container management is offline." [View System Logs] [Restart Incus]. Show last cached list with "(stale)".

## User Actions

- Create instance via StepsForm wizard
- Bulk select: start all, stop all, migrate selected, delete selected
- Inline start/stop/console per row (no drill-down needed)
- Filter by status, type (VM/CT), tags, node
- Sort by name, status, CPU%, RAM%
- Detail: all tab-level actions (snapshot, backup, firewall rule, device add/edit/detach)

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Instance List + Instance Detail)
- Patterns: docs/design/patterns/detail-tabs.md, docs/design/patterns/console.md
