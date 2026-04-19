# Kubernetes Specification

Helling provisions Kubernetes clusters on Incus VMs using k3s bootstrap via cloud-init (ADR-005).

For workload management (pods, deployments, services), use kubectl with the kubeconfig from `helling k8s kubeconfig <name>`.

## Helling K8s API

| Method | Endpoint                             | Description                             |
| ------ | ------------------------------------ | --------------------------------------- |
| GET    | /api/v1/kubernetes                   | List clusters                           |
| POST   | /api/v1/kubernetes                   | Create cluster                          |
| GET    | /api/v1/kubernetes/{name}            | Cluster detail (nodes, version, status) |
| DELETE | /api/v1/kubernetes/{name}            | Delete cluster (destroys VMs)           |
| POST   | /api/v1/kubernetes/{name}/scale      | Scale worker pool                       |
| POST   | /api/v1/kubernetes/{name}/upgrade    | Rolling upgrade                         |
| GET    | /api/v1/kubernetes/{name}/kubeconfig | Download kubeconfig                     |

## Cluster Create Wizard

Dashboard 6-step flow:

1. Basics: cluster name, Kubernetes version, network range
2. Control plane: count (1 or 3), CPU, RAM, disk, storage pool
3. Worker pools: count, CPU, RAM, disk
4. Networking: pod CIDR, service CIDR, ingress toggle
5. Add-ons: metrics-server, ingress controller, cert-manager
6. Review: summary, estimated resources, create

## Provisioning Flow (v0.1)

1. Helling creates Incus VMs for control plane and workers.
2. cloud-init installs and configures k3s server on control-plane node(s).
3. cloud-init joins worker nodes with generated bootstrap token.
4. Helling records cluster metadata in SQLite.
5. User downloads kubeconfig from API/CLI/WebUI.

## Bootstrap Sequence (Normative)

Helling uses an orchestrated sequence to avoid race conditions in HA bootstrap.

1. Generate cluster token and cloud-init payloads.
2. Create first control-plane VM (CP1) with `--cluster-init`.
3. Wait for CP1 readiness with bounded polling:
   - k3s API reachable on `:6443`
   - readiness endpoint reports ready
4. Create remaining control-plane nodes using the shared token and CP1 endpoint.
5. After control-plane readiness threshold is met, create worker nodes in parallel.
6. Persist cluster metadata and return kubeconfig.

Timeout and failure behavior:

- Global provisioning timeout: 10 minutes (default)
- On timeout or bootstrap failure, Helling tears down created cluster VMs and returns a terminal error.
- Partial clusters are not retained by default in v0.1.

## What Helling Does Not Do

- Manage Kubernetes workloads (use kubectl)
- Replace Kubernetes RBAC model
- Provide full in-cluster observability stack by default

Helling manages infrastructure lifecycle for the cluster VMs. Kubernetes manages application workloads.
