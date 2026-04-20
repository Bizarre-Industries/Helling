# Audit

<!-- markdownlint-disable MD022 MD032 -->

> Status: Draft

Route: `/audit`

> **Data source (ADR-014, ADR-019):** Helling API (`/api/v1/audit`). Envelope `{data, meta}`. Underlying store is the host journal (ADR-019), NOT a SQLite audit table. Read-only from the WebUI.

---

## Layout

Sidebar: "Audit" selected (admin and auditor role only, per ADR-032). Main panel: filter toolbar + `ProTable` of records. Detail in expanded row or side `Drawer` showing the full structured event.

## API Endpoints

- `GET /api/v1/audit` -- `auditQuery` (paginated; filterable by user, action, target, date range, status, source IP)
- `GET /api/v1/audit/export` -- `auditExport` (streaming CSV/JSONL of filtered results, respects query)

The audit log is append-only from the WebUI's perspective. Records are written by `hellingd` and stored in the systemd journal with Helling-specific identifiers (ADR-019). Tamper detection is out of WebUI scope — verification happens server-side via journal attributes.

## Components

### Toolbar

- `Input.Search` — text search across actor, action, target, user agent
- `Select` — actor (usernames; multi-select; searchable)
- `Select` — action category (auth | user | schedule | webhook | firewall | kubernetes | system; multi)
- `Select` — outcome (success | failure | denied)
- `DatePicker.RangePicker` — time window; default "last 24h"
- `Input` — source IP / CIDR
- `Button` — "Export CSV" and "Export JSONL" (both hit `/export` with current filters)

Filters are URL query params so pages can be bookmarked and shared.

### Main Table

`ProTable` columns (compact, default 50/page):

- Timestamp (monospace, clicking sorts)
- Actor (username; Tag showing role at the time)
- Action (monospace, e.g. `user.create`, `auth.login`, `schedule.run_now`)
- Target (resource id or name; clickable → deep link to that resource page where possible)
- Outcome `Badge` (green success / red failure / orange denied)
- Source IP (monospace, `Typography.Text copyable`)
- Request ID (short form, monospace; click to filter-by)

`expandable.expandedRowRender`: full structured event as `Descriptions` (user agent, request ID, JWT ID, before/after diff for mutating actions, policy decisions for denials).

### Deep-link Affordances

Click a target like `vm-web-1` → open `/instances/vm-web-1`. Click request ID → filter to that request ID (shows all correlated events, including cross-service). Click action name like `auth.login` → filter to that action type.

### Export

`Button` triggers `/export` with current query. Response streams; show `Progress` in a non-blocking `notification.info` with a cancel button. Max 50k rows per export — if exceeded, toast recommends narrower window.

### Journal field mapping

The audit table renders from the `/api/v1/audit/query` response, which is itself parsed from `journalctl --output=json`. Column-to-field mapping:

| Column         | Journal field                                                 |
| -------------- | ------------------------------------------------------------- |
| Timestamp      | `__REALTIME_TIMESTAMP` (μs epoch)                             |
| Actor          | `HELLING_ACTOR`                                               |
| Role           | `HELLING_ROLE`                                                |
| Action         | `HELLING_ACTION`                                              |
| Outcome        | `HELLING_OUTCOME` (`success` / `failure` / `denied`)          |
| Target         | `HELLING_TARGET_TYPE` + `HELLING_TARGET_ID`                   |
| Source IP      | `HELLING_SOURCE_IP`                                           |
| Request ID     | `HELLING_REQUEST_ID`                                          |
| Before / After | `HELLING_BEFORE` / `HELLING_AFTER` (JSON strings, ≤4 KB each) |
| Policy reason  | `HELLING_POLICY_REASON` (present only when outcome=denied)    |

The diff viewer displays `HELLING_BEFORE` on the left and `HELLING_AFTER` on the right. If either field ends with `,"truncated":true`, the viewer shows a `Truncated (>4 KB)` badge and a "View full context in related events" link filtering to the same `HELLING_REQUEST_ID`. See `docs/spec/audit.md` §1 for the truncation marker contract.

## Data Model

- AuditEvent: `timestamp`, `actor_user_id`, `actor_username`, `actor_role`, `action`, `target_type`, `target_id`, `outcome`, `source_ip`, `user_agent`, `request_id`, `jwt_id`, `fields{}` (action-specific payload, may include `before` / `after` maps for mutations)

## States

### Empty State

"No audit events match this filter." Two secondary actions: "Clear filters" and "Last 24h" reset.

### Loading State

`ProTable` default skeleton. Export button shows inline spinner when export is mid-stream.

### Error State

403: redirect to `/403` with explanation (non-admin / non-auditor user). 429 on export: banner "Export rate limit reached. Try again in X minutes or narrow the filter."

### Large Result Warning

If the filter returns >10,000 matching events, banner: "10,000+ events match. Export to JSONL for offline analysis, or narrow the filter."

## User Actions

- Filter by actor, action, target, outcome, IP, time
- Expand row to see structured payload
- Deep-link to the affected resource
- Export filtered results to CSV or JSONL

## Keyboard

- `/` — focus search
- `E` — export (opens export modal with format choice)
- `C` — clear filters
- `Enter` on focused row — expand / collapse detail
- See docs/design/keyboard.md

## Cross-References

- Spec: docs/spec/webui-spec.md (Audit section)
- API: `/api/v1/audit` in api/openapi.yaml (tag: Audit)
- ADR: 019 (journal over sqlite audit)
- ADR: 032 (three fixed roles — admin and auditor see audit; user does not)
- Pattern: docs/design/patterns/data-tables.md
- Pattern: docs/design/patterns/detail-views.md
- Pattern: docs/design/patterns/empty-states.md
- Related: docs/design/pages/logs.md (audit is user-action trail; logs is system/service logs)
