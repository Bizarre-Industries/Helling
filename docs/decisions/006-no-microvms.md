# ADR-006: Cloud Hypervisor microVMs via direct proxy

> Status: Accepted (updated 2026-04-15)

## Context

Cloud Hypervisor (CH) is a Rust VMM that exposes a stable HTTP REST API on a Unix socket (`/run/ch-{name}/api.sock`). This is the same proxyable pattern as Incus and Podman. CH microVMs boot in <100ms and are purpose-built for ephemeral compute: CI runners, sandboxes, and latency-sensitive workloads.

Firecracker was also evaluated but uses a different socket-per-VM model without a management API suitable for proxying.

## Decision

Add Cloud Hypervisor as a fourth workload type, proxied via `httputil.ReverseProxy` to per-VM Unix sockets. hellingd manages the CH process lifecycle (~200 lines): spawn, monitor, and terminate one `cloud-hypervisor` process per microVM. SQLite records the name → PID → socket path mapping. The proxy routes `/api/ch/{name}/*` to the corresponding per-VM socket.

K8s on microVMs is explicitly deferred — use Incus VMs for K8s nodes (ADR-005). When Incus adds native Cloud Hypervisor backend support, the per-VM lifecycle management migrates to the Incus proxy automatically.

## Consequences

- Four workload types: Incus VMs, Incus LXC, Podman OCI, Cloud Hypervisor microVMs
- Sub-100ms boot for ephemeral workloads (CI runners, sandboxes)
- hellingd owns ~200 lines of CH process lifecycle management (spawn, health-check, cleanup)
- SQLite tracks name → PID → socket path (microvm_instances table)
- Image requirements: raw disk image or qcow2 + separate vmlinux kernel file
- Proxy handles CH REST API; no CH-specific Go SDK
- Incus native CH support (when available) = automatic migration path with zero Helling changes
