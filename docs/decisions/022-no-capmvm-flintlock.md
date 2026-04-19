# ADR-022: No CAPMVM / Flintlock

> Status: Accepted (2026-04-15)

## Context

CAPMVM (Cluster API Provider for MicroVMs) uses Flintlock as the microVM orchestration layer. Flintlock is a gRPC service that manages Firecracker and Cloud Hypervisor VMs on behalf of CAPMVM.

Evaluated for K8s-on-microVMs (fast-start K8s nodes via Flintlock + Firecracker/CH).

## Decision

Reject CAPMVM and Flintlock. Use k3s via cloud-init on Incus VMs for v0.1 provisioning (ADR-005).

Reasons:

- **Weaveworks bankrupt (February 2024):** Flintlock was the primary Weaveworks project. The bankrupcy left Flintlock and CAPMVM without active maintainers.
- **Flintlock is unmaintained:** Last meaningful commit before this decision. Alpha quality, no stable release.
- **gRPC-based:** Flintlock exposes a gRPC API, not HTTP. `httputil.ReverseProxy` cannot proxy gRPC. Adding gRPC transport breaks the proxy-first architecture (ADR-014).
- **Requires containerd sidecar:** Flintlock requires a separate containerd process alongside it. Additional daemon, additional attack surface.
- **CAPMVM alpha / expired signing keys:** CAPMVM releases are alpha; signing keys expired with no rotation. Cannot verify release artifacts.

## Consequences

- K8s nodes are always Incus VMs in v0.1 — no microVM-based K8s
- K8s on Cloud Hypervisor microVMs deferred until a maintained, proxyable solution exists
- No gRPC dependency in hellingd
- No containerd sidecar required
