# Operations

> Status: Draft

Route: `/operations`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

Sidebar: "Operations" selected. Main panel: combined view of audit log, system logs, task history, scheduled operations. Tab or Segmented to switch between views.

## API Endpoints

- `GET /api/v1/audit` -- audit log entries
- `GET /api/v1/tasks` -- task history
- `GET /api/v1/logs` -- system logs (journald, Incus, Podman)
- `GET /api/v1/schedules` -- scheduled operations
- `POST /api/v1/schedules` -- create schedule
- `PUT /api/v1/schedules/:id` -- update schedule
- `DELETE /api/v1/schedules/:id` -- delete schedule
- `SSE /api/v1/events` -- real-time log stream

## Components

- `Tabs` or `Segmented` -- Audit Log | System Logs | Tasks | Schedules

**Audit Log:** `ProTable` (timestamp, user, action, target resource, status Badge, source IP). Filters: user Select, action Select, target input, DatePicker.RangePicker, status Select. Export CSV Button.

**System Logs:** Source `Segmented` (journald, Podman, Incus). Severity `Select` (debug, info, warn, error). `Input.Search` for full-text. `DatePicker.RangePicker`. Auto-scroll `Switch` (tail mode). Instance filter `Select`. Download `Button`.

**Tasks:** `ProTable` (timestamp, user, operation, target, status Badge with Progress for running, duration). Click to expand task detail/output. Cancel Button for running tasks. Filter by status, type, user, time range.

**Schedules:** `ProTable` (instance, action, cron, next run, last status Badge, enable Switch). Create `ModalForm` (instance Select, action Select, cron builder, retention). Execution history expandable per row.

## Data Model

- AuditEntry: `id`, `timestamp`, `user`, `action`, `target`, `status`, `source_ip`, `details`
- LogEntry: `timestamp`, `source`, `severity`, `message`, `instance`
- Task: `id`, `operation`, `target`, `user`, `status`, `started_at`, `duration`, `output`
- Schedule: `id`, `instance`, `action`, `cron`, `enabled`, `next_run`, `last_run`, `last_status`

## States

### Empty State
Audit: "No audit entries yet. All API mutations will be logged here." Schedules: "No scheduled operations." [Create Schedule]

### Loading State
Cached entries shown. New log entries pushed via SSE. Paginated for large datasets.

### Error State
Log source unavailable: inline Alert per source. Other sources still functional.

## User Actions

- Browse/filter/search audit log with export to CSV
- View system logs with severity/source/instance filters
- Monitor running tasks with progress, cancel if needed
- Create/edit/delete/toggle scheduled operations
- Expand task detail to see full output

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Audit, Logs, Tasks, Schedules sections)
