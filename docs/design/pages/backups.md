# Backups

> Status: Draft

Route: `/backups`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`.

---

## Layout

Sidebar: "Backups" selected. Main panel: 2 Tabs (Backups, Schedules).

## API Endpoints

- `GET /api/v1/backups` -- all backups across instances
- `GET /api/v1/backups/:id` -- backup detail
- `POST /api/v1/instances/:name/backups` -- backup now
- `POST /api/v1/backups/:id/restore` -- restore
- `POST /api/v1/backups/:id/verify` -- verify integrity
- `DELETE /api/v1/backups/:id` -- delete backup
- `GET /api/v1/schedules?type=backup` -- backup schedules
- `POST /api/v1/schedules` -- create schedule
- `PUT /api/v1/schedules/:id` -- update schedule
- `DELETE /api/v1/schedules/:id` -- delete schedule

## Components

**Backups tab:** `ProTable` (instance, name, date, size, compression Tag, status Badge, actions). Actions per row: Restore ModalForm (target pool Select), Verify Button, Delete (danger). Toolbar: "Backup Now" Button with instance picker Select.

**Schedules tab:** `ProTable` (instance, action, cron expression, retention policy, next run, last status Badge, enable Switch). Create/Edit via `ModalForm` (instance Select, cron builder, retention: daily/weekly/monthly counts, target pool, compression). Execution history expandable per row.

## Data Model

- Backup: `id`, `instance`, `name`, `created_at`, `size`, `compression`, `status`, `storage_pool`
- Schedule: `id`, `instance`, `action`, `cron`, `retention{}`, `target_pool`, `enabled`, `last_run`, `last_status`, `next_run`
- Retention: `daily`, `weekly`, `monthly`

## States

### Empty State

"No backups configured. Your data is not protected." [Configure Backup Schedule]. "Helling can automatically back up all your instances on a schedule."

### Loading State

Cached backup list shown. Schedule status updates via SSE.

### Error State

Backup failed: status Badge red with error message in expandable row. Toast: "Backup of vm-web-1 failed: storage pool 'default' is full (98%). [View Storage] [Retry Backup]"

## User Actions

- Backup now (select instance, compression, target)
- Restore from backup (select target pool)
- Verify backup integrity
- Create/edit/delete backup schedules
- Toggle schedule enabled/disabled
- View execution history per schedule
- Delete old backups

## Cross-References

- Spec: docs/spec/webui-spec.md (Backups section)
- Identity: docs/design/identity.md (Health Score -- backup deductions)
