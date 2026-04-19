# Compute Specification

All compute operations go through the proxy to Incus and Podman (ADR-014). This document covers only what Helling adds on top.

## Four Workload Types

| Type                    | Engine           | Managed by                  | Dashboard page |
| ----------------------- | ---------------- | --------------------------- | -------------- |
| VMs (QEMU/KVM)          | Incus            | `incus` CLI, Incus proxy    | /instances     |
| System Containers (LXC) | Incus            | `incus` CLI, Incus proxy    | /instances     |
| App Containers (OCI)    | Podman           | `podman` CLI, Podman proxy  | /containers    |
| microVMs                | Cloud Hypervisor | hellingd CH process manager | /microvms      |

For the full Incus instance API, see [Incus REST API](https://linuxcontainers.org/incus/docs/main/rest-api-spec/).
For the full Podman container API, see [Podman API](https://docs.podman.io/en/latest/_static/api.html).

## Helling Additions

### Auto-Snapshot Before Destructive Operations

When the proxy detects a destructive request (DELETE instance, stop with force, rebuild), it creates an automatic snapshot before forwarding:

```
Client → DELETE /api/incus/1.0/instances/vm-web-1
  → hellingd proxy middleware
    → Create snapshot: vm-web-1/auto-pre-delete-20260415T120000
    → Forward DELETE to Incus socket
    → Return Incus response
```

Configurable in helling.yaml:

```yaml
proxy:
  auto_snapshot:
    enabled: true
    retention: 24h # auto-snapshots older than this are cleaned up
    operations: # which operations trigger auto-snapshot
      - delete
      - rebuild
      - stop_force
```

### VM Screenshots / Thumbnails

For VM instances, hellingd periodically captures VGA framebuffer screenshots via the Incus console API. These are served as thumbnails in the dashboard instance list (hover to preview) and instance detail page.

```
GET /api/v1/instances/{name}/thumbnail → PNG screenshot
```

This is a Helling-specific endpoint (not proxied) because it requires capturing and caching screenshots.

### Console

- **VMs:** SPICE protocol via WebSocket proxy to Incus VGA console (ADR-010)
- **System Containers:** Serial console via WebSocket proxy to Incus console
- **App Containers:** Exec terminal via WebSocket proxy to Podman exec

All console connections go through the proxy with WebSocket upgrade support.

### Compose Stacks

Podman compose stacks are managed through the Podman proxy. The dashboard provides a compose file editor and stack lifecycle controls. Template deployment (one-click app install) creates a compose stack.

### App Templates

A library of compose files for common applications. Stored in `/var/lib/helling/templates/`. The dashboard shows a template gallery with one-click deploy:

1. User selects template
2. Dashboard shows env var form (parsed from compose file)
3. User fills in values
4. hellingd writes compose file + .env, calls Podman compose up via proxy

Templates are a Helling feature (served from the Helling API), not proxied.

## Cloud Hypervisor microVMs

microVMs are the fourth workload type. Cloud Hypervisor (CH) runs as a per-VM process managed by hellingd. Each microVM has its own Unix socket at `/run/ch-{name}/api.sock`.

### Process Lifecycle

1. `POST /api/v1/microvms` (Helling-owned endpoint) — hellingd spawns a CH process, records `{name, pid, socket}` in SQLite `microvm_instances`
2. Requests to `/api/ch/{name}/*` are forwarded to the per-VM socket via `httputil.ReverseProxy`
3. `DELETE /api/v1/microvms/{name}` — hellingd sends SIGTERM to CH process, removes socket, deletes SQLite row

On hellingd startup: load `microvm_instances` from SQLite and verify each process is still running (PID check). Dead entries are cleaned up.

### Image Requirements

Cloud Hypervisor requires:

- **Kernel:** `vmlinux` (uncompressed) ELF binary
- **Disk image:** raw (`.raw`) or QCOW2 (`.qcow2`) disk image

Images are referenced by absolute path on the host filesystem at microVM creation time. No registry, no Helling image catalogue.

### Use Cases

- Ephemeral CI runners (spin up in <100ms, destroy after job)
- Security sandboxes (stronger isolation than LXC, faster than full QEMU VMs)
- Short-lived compute tasks with known kernel requirements

### K8s on microVMs

K8s on Cloud Hypervisor microVMs is explicitly deferred (see ADR-006, ADR-022). K8s clusters provisioned by Helling use Incus VMs as nodes (CAPN).
