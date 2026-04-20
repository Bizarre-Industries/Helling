# Logs

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/logs`

> **Data source (ADR-014, ADR-019):** System logs come from the host journal (journald) via `hellingd`. Per-instance logs route through the Incus proxy. Per-container logs route through the Podman proxy. No SQLite audit store — journal is the source of truth (ADR-019).

---

## Layout

Sidebar: "Logs" selected. Main panel: toolbar row + streaming log viewport (`xterm.js` or virtualized list, follow-mode by default). Filter controls collapse into a compact toolbar; no separate filter panel.

## API Endpoints

System logs are read from journald; WebUI hits `hellingd` which tails the journal. Expected surface (not yet in `api/v1/*`):

- `GET /api/v1/logs` -- paginated historical query with filters (source, severity, time range, search, instance, container)
- `GET /api/v1/logs/stream` -- SSE live tail with same filter params
- `GET /api/v1/logs/export` -- bulk download (journalctl-format or JSON) with the same filters applied
- Per-instance logs route through the Incus proxy at `/api/incus/1.0/instances/{name}/logs`
- Per-container logs route through the Podman proxy at `/api/podman/libpod/containers/{id}/logs`

## Components

### Toolbar

- `Segmented` — source: System | hellingd | hellingprox | helling-agent | Incus | Podman | All
- `Select` — severity: Emerg..Debug (multi-select)
- `Input.Search` — message substring (server-side grep)
- `DatePicker.RangePicker` — time window; default "last 15 min"
- `Select` — instance filter (searchable, nullable)
- `Switch` — "Follow" (auto-scroll to tail)
- `Switch` — "Timestamps" (toggle display)
- `Button` — "Download" (triggers `/export` with current filters)

### Viewport

Virtualized list using `@ant-design/pro-components` (custom render) OR `xterm.js` in readonly mode for the ANSI-colored journald format. Each entry:

- Timestamp (monospace, dim)
- Severity badge (color-coded, one char: E/W/I/D)
- Source tag (short, monospace)
- Message (monospace, wraps)

Click an entry → `Drawer` with structured fields (unit, PID, boot ID, MESSAGE_ID, full JSON). `Typography.Text copyable` on every value.

### Follow Mode

When `Follow=true`, viewport pins to bottom. On new SSE entry, append and scroll. If user scrolls up, auto-disable follow and show "Paused at offset X — [Resume]" banner.

### Severity Threshold Hint

If current filter excludes severity levels with active entries (e.g. user filtered to ERR+ but there are recent WARN entries), show a small `Alert` at the top: "42 warnings in this time range not shown. [Show warnings]."

## Data Model

- LogEntry: `timestamp` (RFC3339), `severity` (0-7 syslog scale), `source`, `unit`, `pid`, `message`, `fields{}` (free-form structured)
- Filter state: `source[]`, `severity[]`, `search`, `from`, `to`, `instance`, `follow`

## States

### Empty State

"No log entries match this filter." Secondary action: "Clear filters" button.

### Loading State

Skeleton lines (6 rows) while initial page loads. SSE reconnection shows a slim banner, not a spinner.

### Error State

SSE stream drop: `Alert type="warning"` with exponential backoff reconnect indicator. "Retry now" button for manual reconnect.

### Rate-Limited State

Server returning 429: banner "Log stream rate-limited — showing 1-in-10 entries. Narrow the time range for full fidelity."

## User Actions

- Filter by source, severity, time, search text, instance
- Toggle follow / timestamps
- Click entry to see structured fields
- Download filtered results as journald JSON or plain text
- Copy individual field values

## Keyboard

- `/` — focus search input
- `F` — toggle follow
- `T` — toggle timestamps
- `Cmd/Ctrl+End` — jump to tail (re-enables follow)
- `Cmd/Ctrl+Home` — jump to head
- See docs/design/keyboard.md

## Cross-References

- Spec: docs/spec/webui-spec.md (Logs section)
- Spec: docs/spec/observability.md
- ADR: 019 (journal over sqlite audit)
- Pattern: docs/design/patterns/loading-error.md
- Pattern: docs/design/patterns/empty-states.md
- Note: this page is _system_ and _service_ logs. For the immutable user-action audit trail, see docs/design/pages/audit.md.
