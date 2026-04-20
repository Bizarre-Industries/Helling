# Schedules

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/schedules`

> **Data source (ADR-014):** Helling API (`/api/v1/schedules`). Envelope `{data, meta}`. Execution runs under systemd timers (ADR-017).

---

## Layout

Sidebar: "Schedules" selected. Main panel: single `ProTable` of all scheduled actions across all targets. Detail shown inline via `expandedRowRender` (next-run preview, last-run summary, last-failure summary).

## API Endpoints

- `GET /api/v1/schedules` -- `scheduleList`
- `POST /api/v1/schedules` -- `scheduleCreate`
- `GET /api/v1/schedules/{id}` -- `scheduleGet`
- `PUT /api/v1/schedules/{id}` -- `scheduleUpdate`
- `DELETE /api/v1/schedules/{id}` -- `scheduleDelete`
- `POST /api/v1/schedules/{id}/run` -- `scheduleRunNow`

Schedules are evaluated by systemd timers on the host (ADR-017), not by a Go-level cron loop.

## Components

### Main Table

`ProTable` columns:

- `Switch` enable/disable (inline; optimistic update)
- Name (bold)
- Action `Tag` (backup | snapshot | restart | update-check | custom-hookscript)
- Target: instance name, "all instances", or tag expression (monospace)
- Cron spec (monospace, with humanized hint on hover: "Every Monday at 03:00")
- Next run (relative time, e.g. "in 2h 14m")
- Last status `Badge` (success | failed | skipped | never)
- Last duration (e.g. "1m 42s")
- Actions: Run Now (icon), Edit (icon), Delete (icon with `Popconfirm` danger)

`rowSelection` bulk: Enable, Disable, Delete.

`search={{ filterType: 'light' }}` for inline filters on action, target, status.

### Create/Edit Form

`ModalForm` with fields:

- Name (required)
- Action (Select: backup/snapshot/restart/update-check/custom)
- If custom: hookscript dropdown (populated from `/api/v1/system/hookscripts` — v0.5+)
- Target mode (Segmented): Single instance | All instances | By tag
- Target value (conditional: instance Select, or tag expression input)
- Cron (Input + live validation against a robust cron parser; show next 5 runs below)
- Retention (if action is backup/snapshot): keep last N / keep for days
- Enabled (Switch, default on)

### Expanded Row Detail

`Descriptions`:

- Next 5 scheduled runs (with timezone note)
- Last 5 executions: timestamp, duration, status, log excerpt, "View in Logs" link (deep-links to `/logs?query=...`)
- systemd unit name (monospace, read-only; explains where to look on the host)

## Data Model

- Schedule: `id`, `name`, `action`, `target` (string, expression or instance name), `cron`, `timezone`, `retention_days`, `retention_count`, `enabled`, `last_run`, `last_status`, `last_duration_ms`, `next_run`

## States

### Empty State

"No schedules configured. Schedules let you automate backups, snapshots, restarts, and update checks. [Create Schedule]. Helling uses systemd timers (ADR-017), so schedules survive daemon restarts."

### Loading State

`ProTable` default skeleton. Inline Switch toggles disabled while individual update inflight.

### Error State

- Individual row 409 conflict (name in use): inline red border on name field, preserve other form state
- Network failure on enable/disable toggle: roll back optimistic change, show toast

### Warning State

Schedule with `last_status = failed` for N consecutive runs: row highlight, show ! icon before name, link to last execution in `/logs`.

## User Actions

- Create / edit / delete / enable / disable schedules
- Run a schedule immediately
- View upcoming and past executions
- Bulk enable/disable/delete
- Deep-link from execution history into `/logs`

## Keyboard

- `C` — open create modal
- `R` on focused row — run now (with confirm)
- `E` on focused row — edit
- `D` on focused row — delete (with confirm)
- `Space` on focused row — toggle enable
- See docs/design/keyboard.md

## Cross-References

- Spec: docs/spec/webui-spec.md (Schedules section)
- API: `/api/v1/schedules` in api/openapi.yaml (tag: Schedules)
- ADR: 017 (systemd timers over cron)
- ADR: 014 (proxy architecture, not used here — Helling-owned domain)
- Pattern: docs/design/patterns/data-tables.md
- Pattern: docs/design/patterns/forms-wizards.md
- Pattern: docs/design/patterns/empty-states.md
- Related: docs/design/pages/backups.md (backup schedules surface both here and there)
