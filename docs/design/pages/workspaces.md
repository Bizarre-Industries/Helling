# Workspaces

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/workspaces`

> **Data source (ADR-014):** Workspaces are ephemeral Incus instances launched from a Helling-managed template. List of active sessions is tracked in Helling; launching an instance hits the Incus proxy. See docs/spec/webui-spec.md ("workspaces" section) for baseline.

---

## Layout

Sidebar: "Workspaces" selected (power-user section, always visible). Main panel: two stacked regions — template gallery on top (`ProList grid`), active sessions below (`ProTable`). Empty state when neither region has content directs user to create a template.

## API Endpoints

Workspace endpoints are not yet in `/api/v1/*`. Expected v0.5+ surface:

- `GET /api/v1/workspaces/templates` -- list workspace templates (Helling-managed, distinct from app templates)
- `POST /api/v1/workspaces/templates` -- create/update workspace template
- `DELETE /api/v1/workspaces/templates/{id}` -- delete template
- `GET /api/v1/workspaces/sessions` -- list active sessions
- `POST /api/v1/workspaces/sessions` -- launch workspace from template (returns instance name + console URL)
- `DELETE /api/v1/workspaces/sessions/{id}` -- destroy session (stops + deletes underlying instance)
- `GET /api/v1/workspaces/sessions/{id}/idle` -- SSE of idle-timer state

Launch flow under the hood: POST to Incus proxy to create instance, wait until running, open console tab. Session row tracks the association.

## Components

### Template Gallery

- `ProList` grid — column=3 or 4 depending on viewport
- `Card` per template: name, icon, short description, base image, resource profile (e.g. "2 vCPU / 4 GB"), idle timeout, "Launch" primary button
- Click template body → detail `Drawer` with full config (cloud-init, profiles, hookscripts)
- Toolbar: "New Template" button (`StepsForm`: Name → Base Image → Resources → Cloud-Init → Idle Policy → Review)

### Active Sessions

- `ProTable` columns: name (monospace), template, user, uptime, idle countdown, state `Badge`
- Inline row actions: Open Console (icon), Destroy (icon, `Popconfirm` danger)
- Bulk action on selection: Destroy Selected
- `Statistic.Countdown` in the idle column; auto-refreshes via SSE `/idle`

### Launch Action

1. Click "Launch" on a template card
2. Optional `ModalForm` if template has parameterized env vars or SSH key injection
3. Progress `Drawer` (bottom) shows provisioning steps streamed via SSE
4. On instance `Running`, auto-switch to console tab in a new browser tab

## Data Model

- Template: `id`, `name`, `description`, `icon`, `base_image`, `cpu`, `memory_mb`, `disk_gb`, `cloud_init_yaml`, `profiles[]`, `hookscripts{}`, `idle_timeout_minutes`
- Session: `id`, `template_id`, `instance_name`, `user_id`, `created_at`, `last_activity_at`, `state`, `console_url`

## States

### Empty State

Two-row empty state. Top: "No workspace templates yet. [Create Template]. Workspaces are disposable dev environments launched from templates." Bottom: "No active sessions."

### Loading State

Template gallery: skeleton cards. Session table: default `ProTable` skeleton. Launch button disabled if template list is still loading.

### Error State

Launch failure: `Alert type="error"` at top of sessions table with the instance name that failed and a "Retry" button. Don't block unrelated rows.

### Idle Warning

Session `idle_timeout_minutes - last_activity_at` < 5 min: row highlights warning color, countdown badge pulses (but no animation — just color change per philosophy rule 6).

## User Actions

- Browse workspace templates
- Create/edit/delete workspace templates (admin only)
- Launch a session from a template
- Monitor idle countdown
- Destroy a session (reclaims resources immediately)
- Bulk destroy selected sessions

## Keyboard

- `L` on a focused template card: launch
- `D` on a focused session row: destroy (with confirm)
- `Enter` on a focused session row: open console
- See docs/design/keyboard.md for global bindings

## Cross-References

- Spec: docs/spec/webui-spec.md (Workspaces section)
- Related: docs/design/pages/templates.md (app templates, a different concept)
- Pattern: docs/design/patterns/data-tables.md
- Pattern: docs/design/patterns/empty-states.md
- ADR: 014 (proxy architecture)
