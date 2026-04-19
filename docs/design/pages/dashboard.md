# Dashboard

> Status: Draft

Route: `/`

> **Data source (ADR-014):** Mixed. System stats from Helling API (`/api/v1/*`, envelope format `{data, meta}`). Instance/storage/network data from Incus proxy (`/api/incus/1.0/*`, native Incus format). Container data from Podman proxy (`/api/podman/v5.0/libpod/*`, native Podman format).

---

## Layout

Sidebar: "Dashboard" selected in resource tree. Main panel: full-width grid of stats, tables, and widgets. No tabs. Task log drawer at bottom.

## API Endpoints

- `GET /api/v1/system/status` -- CPU, RAM, disk, uptime (Helling API)
- `GET /api/incus/1.0/instances` -- instance counts and top consumers (Incus proxy)
- `GET /api/podman/v5.0/libpod/containers/json` -- container counts (Podman proxy)
- `GET /api/incus/1.0/storage-pools` -- pool usage (Incus proxy)
- `GET /api/v1/tasks?limit=10` -- recent operations (Helling API)
- `GET /api/v1/warnings` -- active warnings (Helling API)
- `SSE /api/v1/events` -- real-time status updates (Helling API)

## Components

- `Row` / `Col` -- grid layout for stat cards
- `Statistic` -- VMs running/total, CTs, CPU%, RAM used/total
- `Progress` -- storage pool usage bars, health score gauge
- `ProTable` (compact) -- top resource consumers (CPU/RAM), recent tasks
- `Alert` -- active warnings list (disk health, backup failures, node offline)
- `Button` -- quick actions: Create Instance, Create Container, Deploy Template
- `Timeline` -- recent tasks with status icons

## Data Model

- System: `cpu_percent`, `ram_used`, `ram_total`, `disk_used`, `disk_total`, `uptime`
- Counts: `vms_running`, `vms_total`, `cts_running`, `cts_total`, `containers_running`
- Health: `score` (0-100), `deductions[]` (reason, points, resource)
- Warnings: `severity`, `message`, `resource`, `action`
- Tasks: `id`, `operation`, `target`, `status`, `duration`, `user`, `timestamp`

## States

### Empty State
"No virtual machines or containers yet." Two buttons: [Create Instance] [Deploy from Template]. Popular templates listed below (Jellyfin, Gitea, Uptime Kuma, Pi-hole).

### Loading State
Show cached data immediately via React Query. Background refresh via SSE. No full-page spinner.

### Error State
Banner: "Connection lost. Showing cached data." All data shows stale timestamp. Action buttons disabled with tooltip.

## User Actions

- Click quick action buttons to create resources
- Click warning to navigate to affected resource
- Click task row to expand detail/output
- Click storage pool bar to navigate to storage page

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Dashboard section)
- Patterns: docs/design/patterns/empty-states.md
- Identity: docs/design/identity.md (First 5 Minutes, Health Score)
