# Cluster

> Status: Draft

Route: `/cluster`

> **Data source (ADR-014):** Incus proxy (`/api/incus/1.0/*`). Responses in native Incus format.

---

## Layout

Sidebar: node names visible in resource tree with status icons. Main panel: node cards + HA tab. Task log shows migration/evacuation progress.

## API Endpoints

- `GET /api/incus/1.0/cluster` -- cluster status, quorum, leader
- `GET /api/incus/1.0/cluster/members` -- node list with metrics
- `GET /api/incus/1.0/cluster/members/:name` -- node detail
- `POST /api/incus/1.0/cluster/members` -- join node
- `POST /api/incus/1.0/cluster/members/:name/state` -- evacuate
- `DELETE /api/incus/1.0/cluster/members/:name` -- remove node
- `GET /api/incus/1.0/instances?target=:name` -- instances on node

## Components

- `Card` (per node) -- hostname, status Badge, CPU/RAM `Progress` bars, instance count, role Tag (leader/voter/standby). Cards for <=8 nodes (explicit exception to tables rule).
- `ProTable` fallback -- if >8 nodes, switch to table view
- `Badge` -- quorum status (healthy/degraded/lost) at top
- `ModalForm` -- Join Node (generates join token, shows command to run on new node)
- `Button` per node: Evacuate (warning), Remove (danger)
- `Tabs` -- Nodes | HA (high availability settings)
- **HA tab:** Failover configuration, automatic migration settings, fencing config

## Data Model

- Cluster: `name`, `quorum`, `leader`, `member_count`
- Member: `name`, `address`, `status`, `roles[]`, `cpu_used`, `cpu_total`, `memory_used`, `memory_total`, `instances[]`, `architecture`

## States

### Empty State
"Not clustered. This is a standalone node." [Create Cluster]. "Clustering enables live migration, high availability, and distributed storage."

### Loading State
Node cards cached. SSE pushes status changes (node join, evacuate progress).

### Error State
Node offline: card shows red Badge. Quorum lost: top-level Alert with recovery instructions. Evacuation failed: task log shows error with retry.

## User Actions

- Create cluster (first node becomes leader)
- Join new node (generate token, show join command)
- Evacuate node (migrate all instances off before maintenance)
- Remove node from cluster
- View instance distribution per node
- Drag instance in resource tree from one node to another (live migration)
- Configure HA settings

## Cross-References

- Spec: docs/refs/spec/webui-spec.md (Cluster section)
