# Kubernetes Specification

Helling provisions K8s clusters using CAPN (Cluster API Provider for Nested virtualization). CAPN creates Incus VMs, installs K8s on them, and manages the cluster lifecycle.

For managing K8s workloads (pods, deployments, services), use `kubectl` with the kubeconfig from `helling k8s kubeconfig <name>`.

## Helling K8s API

| Method | Endpoint | Description |
|---|---|---|
| GET | /api/v1/kubernetes | List clusters |
| POST | /api/v1/kubernetes | Create cluster |
| GET | /api/v1/kubernetes/{name} | Cluster detail (nodes, version, status) |
| DELETE | /api/v1/kubernetes/{name} | Delete cluster (destroys VMs) |
| POST | /api/v1/kubernetes/{name}/scale | Scale worker pool |
| POST | /api/v1/kubernetes/{name}/upgrade | Rolling upgrade |
| GET | /api/v1/kubernetes/{name}/kubeconfig | Download kubeconfig |

## Cluster Create Wizard

Dashboard 6-step StepsForm:

1. **Basics:** cluster name, K8s version, network range
2. **Control plane:** count (1 or 3), CPU, RAM, disk, storage pool
3. **Worker pools:** count, CPU, RAM, disk, GPU passthrough
4. **Networking:** CNI plugin (Flannel, Calico, Cilium), pod CIDR, service CIDR
5. **Add-ons:** metrics-server, ingress-nginx, cert-manager, dashboard
6. **Review:** summary, estimated resources, create

## How It Works

1. Helling calls CAPN to provision Incus VMs for control plane + workers
2. CAPN installs K8s (via kubeadm or k3s) on the VMs
3. Helling stores cluster metadata in SQLite (name, status, kubeconfig path, node count)
4. The Incus VMs are visible in the dashboard instance list (tagged with `user.tag.k8s-cluster=<name>`)
5. User downloads kubeconfig via `helling k8s kubeconfig <name>` or the dashboard

## What Helling Does NOT Do

- Manage K8s workloads (that's kubectl)
- Run a K8s dashboard (deploy it as an add-on if wanted)
- Provide K8s RBAC (that's K8s native)
- Monitor K8s pods/services (that's Prometheus + Grafana inside the cluster)

Helling manages the infrastructure (VMs) that K8s runs on. K8s manages the workloads.
