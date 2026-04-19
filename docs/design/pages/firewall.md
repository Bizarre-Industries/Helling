# Firewall

> Status: Draft

Route: `/firewall`

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`. VM/CT firewalling uses Incus Network ACLs; host firewall uses `nft --json` (see ADR-012).

---

## Layout

Sidebar: "Firewall" selected. Main panel: 4 Tabs. Each tab contains a ProTable.

## API Endpoints

- `GET /api/v1/firewall/rules` -- list rules
- `POST /api/v1/firewall/rules` -- create rule
- `PUT /api/v1/firewall/rules/:id` -- update rule
- `PUT /api/v1/firewall/rules/order` -- reorder rules
- `DELETE /api/v1/firewall/rules/:id` -- delete rule
- `GET /api/v1/firewall/groups` -- security groups
- `POST /api/v1/firewall/groups` -- create group
- `GET /api/v1/firewall/ipsets` -- IP sets
- `POST /api/v1/firewall/ipsets` -- create IP set
- `GET /api/v1/firewall/macros` -- macro/alias list

## Components

- `Tabs` -- Rules | Security Groups | IP Sets | Macros

**Rules tab:** `ProTable` with drag-to-reorder rows (direction Tag, action Tag, protocol, port, source, enable Switch). `ModalForm` for Add Rule (direction, action, protocol, port range, source CIDR, comment). Scope selector: cluster-wide vs node-level.

**Security Groups tab:** `ProTable` (name, rules count, assigned instances count). Click to expand/edit rules. Assign group to instances via `Select`.

**IP Sets tab:** `ProTable` (name, entries count, type). Click to view/edit entries. `ModalForm` to create set and add IPs/subnets.

**Macros tab:** Read-only `ProTable` of built-in macros (HTTP, SSH, DNS, SMTP, etc.) with protocol/port details.

## Data Model

- Rule: `id`, `direction` (in/out), `action` (accept/drop/reject), `protocol`, `dport`, `source`, `enabled`, `comment`, `position`
- SecurityGroup: `id`, `name`, `rules[]`, `instances[]`
- IPSet: `id`, `name`, `type` (ip/net/port), `entries[]`
- Macro: `name`, `protocol`, `dport`, `description`

## States

### Empty State
"No firewall rules. All traffic is allowed to all instances." [Create Rule] [Apply Default Policy]. "Recommended: start with deny-all and add specific allow rules."

### Loading State
Cached rules shown immediately. Reorder updates optimistically.

### Error State
nftables unavailable: banner with link to system logs. Rules shown as read-only "(stale)".

## User Actions

- Add/edit/delete/reorder firewall rules (drag-to-reorder)
- Toggle individual rules on/off
- Create/manage security groups and assign to instances
- Create/manage IP sets
- View built-in macros
- Switch scope between cluster-wide and node-level

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Firewall section)
