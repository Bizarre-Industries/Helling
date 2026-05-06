# Firewall

> Status: Draft

Route: `/firewall`

> **Data source (ADR-014):** Helling API (`/api/v1/firewall/host`). Responses in Helling envelope format `{data, meta}`. v0.2 manages host firewall rules only with structured nftables DTOs (see ADR-012).

---

## Layout

Sidebar: "Firewall" selected. Main panel: one host-rule table. Security groups,
IP sets, macros, VM/CT Network ACL views, and drag reordering are post-v0.2.

## API Endpoints

- `GET /api/v1/firewall/host` -- list Helling-managed host rules
- `POST /api/v1/firewall/host` -- create a host rule
- `DELETE /api/v1/firewall/host/{id}` -- delete a host rule

## Components

**Host rules table:** `ProTable` with direction Tag, action Tag, protocol, port,
source CIDR, destination CIDR, enabled state, nft handle when known, and
`helling:<uuid>` comment. `ModalForm` for Add Rule uses structured fields only;
the daemon renders nft as argv with `exec.CommandContext`.

## Data Model

- Rule: `id`, `direction` (input/output/forward), `action` (accept/drop/reject), `protocol`, `source_cidr`, `destination_cidr`, `destination_port`, `enabled`, `comment`, `nft_handle`, `created_at`, `updated_at`

## States

### Empty State

"No firewall rules. All traffic is allowed to all instances." [Create Rule] [Apply Default Policy]. "Recommended: start with deny-all and add specific allow rules."

### Loading State

Cached rules shown immediately. Reorder updates optimistically.

### Error State

nftables unavailable: banner with link to system logs. Rules shown as read-only "(stale)".

## User Actions

- Add/delete host firewall rules
- Inspect nft metadata and last apply status
- Filter by action, protocol, source, destination, and enabled state

## Cross-References

- Spec: docs/spec/webui-spec.md (Firewall section)
