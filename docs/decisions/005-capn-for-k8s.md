# ADR-005: CAPN for Kubernetes

> Status: Accepted (updated 2026-04-15)

## Context

Needed Kubernetes cluster lifecycle management. Options: Flintlock + CAPMVM (microVM-based), CAPN (Cluster API Provider for Incus), or manual kubeadm scripting.

CAPMVM was explicitly rejected — see ADR-022.

## Decision

Use CAPN (Cluster API Provider for Incus), maintained under the lxc organization. Standard Cluster API interface.

Helling generates CAPI manifests (ClusterClass, Cluster, KubeadmControlPlane, MachineDeployment) and applies them to a management cluster via `kubectl`. Helling is a CAPI consumer, not a CAPI provider. K8s nodes are Incus VMs.

K8s on Cloud Hypervisor microVMs is explicitly deferred — use Incus VMs for K8s nodes.

## Consequences

- Standard Cluster API workflow (create, scale, upgrade, delete)
- Incus VMs as K8s nodes (full isolation)
- Maintained by lxc community
- Dependent on CAPN release cadence
- Cluster API is complex but well-documented
- No CAPMVM/Flintlock dependency — avoids bankrupt upstream (ADR-022)
- microVM-based K8s deferred until Cloud Hypervisor integration matures
