# ADR-005: k3s via cloud-init for v0.1 Kubernetes

> Status: Accepted (canonical for v0.1, 2026-04-19)

## Context

Helling v0.1 needs pragmatic Kubernetes cluster creation on Incus VMs with low implementation risk.

Cluster API integration (including CAPN, ClusterClass management, and lifecycle reconciliation) introduces significant control-plane complexity for the first release.

## Decision

For v0.1, Helling provisions Kubernetes using k3s installed by cloud-init on Incus VMs.

Baseline flow:

1. Create control-plane and worker Incus VMs.
2. Inject cloud-init for deterministic k3s install and bootstrap.
3. Join workers using generated token workflow.
4. Return kubeconfig to the requesting user.

CAPN (Cluster API Provider for Incus) is deferred to v0.5+ as an optional advanced mode.

## Consequences

- Faster path to reliable Kubernetes support in v0.1
- Smaller operational surface for first-release debugging
- No dependency on CAPI controller stack in the baseline install
- CAPN can be introduced later without breaking v0.1 contract
