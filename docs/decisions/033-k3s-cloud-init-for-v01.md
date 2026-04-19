# ADR-033: k3s via cloud-init for v0.1 Kubernetes provisioning

> Status: Accepted (2026-04-19)

## Context

v0.1 needs practical Kubernetes provisioning without introducing a management cluster and CAPI controller complexity.

## Decision

Use cloud-init driven k3s bootstrap on Incus VMs for v0.1.

- Control-plane VM(s) provisioned first
- Worker VMs join using generated token
- kubeconfig returned through Helling API/CLI

CAPN remains a deferred v0.5+ option.

## Consequences

- Lower implementation risk and operational complexity in v0.1
- Predictable homelab and small-cluster bootstrap path
