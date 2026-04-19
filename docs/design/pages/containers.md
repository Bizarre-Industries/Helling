# Containers

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/containers` (list) + `/containers/:id` (detail)

> **Data source (ADR-014):** Podman proxy (`/api/podman/v5.0/libpod/*`). Responses in native Podman format.

---

## Layout

Sidebar: Podman section in resource tree. List view: `Segmented` toggle for 3 views (Containers, Stacks, Pods). Detail view: breadcrumb + 6 Tabs. Task log drawer at bottom.

## API Endpoints

- `GET /api/podman/v5.0/libpod/containers/json` -- list all containers
- `GET /api/podman/v5.0/libpod/containers/:id/json` -- detail
- `GET /api/podman/v5.0/libpod/containers/:id/stats` -- live CPU/RAM/net/disk
- `GET /api/podman/v5.0/libpod/containers/:id/logs` -- WebSocket log stream
- `POST /api/podman/v5.0/libpod/containers/:id/exec` -- WebSocket exec
- `POST /api/podman/v5.0/libpod/containers/create` -- create
- `POST /api/podman/v5.0/libpod/containers/:id/start` / `stop` / `restart` -- lifecycle
- `DELETE /api/podman/v5.0/libpod/containers/:id` -- remove
- `GET /api/podman/v5.0/libpod/containers/json?pod=true` -- compose stacks
- `GET /api/podman/v5.0/libpod/pods/json` -- pod list

## Components

### List (`/containers`)

- `Segmented` -- Containers | Stacks | Pods view toggle
- **Containers view:** `ProTable` (status Badge, name, image, ports as links, CPU%, RAM%, health Badge). `rowSelection` for bulk. Image update `Badge dot` when newer digest available.
- **Stacks view:** `Collapse` panels per compose stack. Stack actions: Start/Stop All, View YAML, Edit + Redeploy, Combined Logs, Save as Template.
- **Pods view:** `Collapse` panels per pod. Same grouped pattern.
- `Button` (primary) -- "Create Container" toolbar. Also: "Import Compose File".

### Detail (`/containers/:id`) -- 6 Tabs

**Summary tab:** `Descriptions` (status, image, ports as links, env vars, volumes, limits, health, restart policy). Quick action `Button.Group`.

**Logs tab:** xterm.js with search, timestamps toggle, follow toggle. Severity filter `Select`.

**Exec tab:** xterm.js terminal. Shell selector `Select` (bash, sh, zsh).

**Stats tab:** `Area` charts from @ant-design/charts (CPU, RAM, net I/O, disk I/O). Timeframe `Segmented` (1h, 6h, 24h).

**Files tab:** `Tree` filesystem browser. Click file to view/edit (CodeMirror dynamic import). `Upload.Dragger` for upload. Download per file.

**Config tab:** `Descriptions` of full container config. `Typography.Text copyable` on all values.

## Data Model

- Container: `id`, `name`, `image`, `status`, `ports[]`, `volumes[]`, `env{}`, `labels{}`, `health`, `restart_policy`, `created_at`
- Stats: `cpu_percent`, `memory_usage`, `memory_limit`, `net_io`, `disk_io`
- Stack: `name`, `file`, `services[]`, `status`
- Pod: `name`, `containers[]`, `status`

## States

### Empty State

"No application containers yet." [Create Container] [Deploy from Template] [Import Compose File]

### Loading State

Cached data immediately. SSE pushes container state changes.

### Error State

Podman socket unavailable: "Podman service is not responding. Container management is offline." [View System Logs]. Socket-activated: handle connection refused gracefully with retry.

## User Actions

- Create container, import compose file, deploy from template
- Bulk stop, bulk remove stopped containers
- Inline start/stop/logs per row
- Stack-level actions: start/stop all, redeploy, save as template
- Detail: exec into shell, view/download files, edit config

## Cross-References

- Spec: docs/spec/webui-spec.md (Container List + Container Detail)
- Patterns: docs/design/patterns/console.md
