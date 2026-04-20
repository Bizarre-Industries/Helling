# Kubernetes

> Status: Draft

Route: `/kubernetes` (cluster list) + `/kubernetes/:id` (cluster detail)

> **Data source (ADR-014):** Helling API (`/api/v1/*`). Responses in Helling envelope format `{data, meta}`. K8s clusters are managed by Helling; underlying VMs are Incus instances.

---

## Layout

Sidebar: K8s section in resource tree. List view: ProTable of clusters. Detail view: breadcrumb + 8 Tabs with lifecycle toolbar. Task log drawer at bottom.

## API Endpoints

- `GET /api/v1/kubernetes/clusters` -- cluster list
- `GET /api/v1/kubernetes/clusters/:id` -- cluster detail
- `POST /api/v1/kubernetes/clusters` -- create cluster
- `DELETE /api/v1/kubernetes/clusters/:id` -- delete cluster
- `POST /api/v1/kubernetes/clusters/:id/upgrade` -- upgrade K8s version
- `POST /api/v1/kubernetes/clusters/:id/scale` -- scale workers
- `GET /api/v1/kubernetes/clusters/:id/kubeconfig` -- download kubeconfig
- `POST /api/v1/kubernetes/clusters/:id/etcd/snapshot` -- etcd snapshot
- `SSE /api/v1/kubernetes/clusters/:id/events` -- K8s events

## Components

### List (`/kubernetes`)

- `ProTable` -- columns: name, flavor Tag, nodes, K8s version, status Badge, resource usage Progress
- `StepsForm` -- create wizard (6 steps: Flavor cards, Control Plane sliders, Worker Pools repeatable section with labels/taints, Networking CNI Select + CIDR inputs, Add-ons checkboxes, Review Descriptions)

### Detail (`/kubernetes/:id`) -- 8 Tabs

**Overview:** `Descriptions` (version, endpoint copyable, status, etcd health, node count, pod count). Resource `Progress` bars. Kubeconfig download `Button`.

**Nodes:** `ProTable` (name, role Tag, status Badge, version, CPU/RAM Progress, pods, conditions). Actions: Cordon, Drain, Uncordon, Delete, Maintenance Mode.

**Workloads:** Sub-tabs via `Tabs`: Deployments | StatefulSets | DaemonSets | Jobs | Pods. Each a `ProTable`. Click row opens `Drawer` with YAML, events, logs.

**Services:** `ProTable` for Services, Ingresses, NetworkPolicies.

**Storage:** `ProTable` for PVCs, StorageClasses, PVs.

**Config:** `ProTable` for ConfigMaps, Secrets (values hidden by default).

**Events:** Real-time K8s events `Table` via SSE. Filter by namespace `Select`, type `Select`.

**kubectl:** Embedded xterm.js terminal with pre-loaded kubeconfig and autocomplete.

**Lifecycle toolbar:** Upgrade `ModalForm` (version picker, rolling progress), Scale `ModalForm`, etcd Snapshot/Restore.

## Data Model

- Cluster: `id`, `name`, `flavor`, `k8s_version`, `status`, `endpoint`, `nodes[]`, `created_at`
- Node: `name`, `role`, `status`, `version`, `cpu`, `memory`, `pods`, `conditions[]`
- Workload: `name`, `namespace`, `kind`, `replicas`, `ready`, `status`

## States

### Empty State

"No Kubernetes clusters yet." [Create Cluster]. "Helling creates K8s clusters from Incus VMs with full lifecycle management."

### Loading State

Cached cluster list shown. SSE pushes status changes during provisioning.

### Error State

Cluster unhealthy: status Badge shows degraded. Events tab highlights errors. API unreachable: "(stale)" indicator.

## User Actions

- Create cluster via 6-step StepsForm wizard
- Scale worker nodes, upgrade K8s version, etcd snapshot/restore
- Download kubeconfig, exec kubectl in embedded terminal
- Cordon/drain/uncordon individual nodes
- Browse workloads, services, storage, config, events

## Cross-References

- Spec: docs/spec/webui-spec.md (Kubernetes sections)
- Patterns: docs/design/patterns/console.md
