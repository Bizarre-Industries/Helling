# Compute Specification

All compute operations go through the proxy to Incus and Podman (ADR-014). This document covers only what Helling adds on top.

## Three Workload Types

| Type                    | Engine | Managed by                 | Dashboard page |
| ----------------------- | ------ | -------------------------- | -------------- |
| VMs (QEMU/KVM)          | Incus  | `incus` CLI, Incus proxy   | /instances     |
| System Containers (LXC) | Incus  | `incus` CLI, Incus proxy   | /instances     |
| App Containers (OCI)    | Podman | `podman` CLI, Podman proxy | /containers    |

MicroVM support is deferred from v0.1 (ADR-006).

## Helling Additions

### Auto-Snapshot Before Destructive Operations

When the proxy detects a destructive request (delete/rebuild/forced stop), it creates an automatic snapshot before forwarding the request.

### VM Screenshots / Thumbnails

hellingd captures VM console thumbnails and serves them through Helling-specific endpoints.

### Console

- **VMs:** noVNC via WebSocket proxy to Incus VNC socket (ADR-010)
- **System Containers:** Serial console via WebSocket proxy to Incus console
- **App Containers:** Exec terminal via WebSocket proxy to Podman exec

### Compose Stacks and App Templates

Podman compose stacks are managed through the Podman proxy. Template files are stored under `/var/lib/helling/templates/`.

### Kubernetes Relation

Kubernetes clusters in v0.1 use k3s via cloud-init on Incus VMs (ADR-005).
