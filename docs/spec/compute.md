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

Before forwarding selected destructive Incus operations, hellingd attempts an automatic snapshot for recovery safety.

Trigger operations:

- Instance delete
- Instance rebuild
- Forced stop (`action=stop` with `force=true`)

Snapshot defaults:

- Name format: `auto-<operation>-<timestamp-utc>`
- Stateful mode: `false` by default for VM reliability and speed
- Retention: 24 hours by default
- Expiry enforcement: use Incus snapshot `expires_at` field (no custom cleanup worker required for auto-snapshots)

Failure policy:

- Fail-closed by default for safety (operation blocked when snapshot creation fails)
- Optional lenient mode can allow operation to proceed with warning and audit event

Scope controls:

- Per-project toggle: enable or disable auto-snapshot hook
- Per-request bypass: admin-only override flag

### VM Screenshots / Thumbnails

Deferred from v0.1.

Rationale: no stable low-cost capture path is currently specified for SPICE-backed consoles in the proxy-first model. Revisit in a later version when a reliable guest-agent or upstream-native capture mechanism is selected and documented.

### Console

- **VMs:** SPICE via WebSocket proxy to Incus VGA operation channels (ADR-010)
- **System Containers:** Serial console via WebSocket proxy to Incus console
- **App Containers:** Exec terminal via WebSocket proxy to Podman exec

### Compose Stacks and App Templates

Podman compose stacks are managed through the Podman proxy. Template files are stored under `/var/lib/helling/templates/`.

### Kubernetes Relation

Kubernetes clusters in v0.1 use k3s via cloud-init on Incus VMs (ADR-005).
