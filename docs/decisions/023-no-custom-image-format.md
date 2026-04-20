# ADR-023: No custom image format or wrapper

> Status: Accepted (2026-04-15)

## Context

Each workload runtime has its own image model:

- **Incus VMs/LXC:** Incus image format with remote servers (`images:`, `ubuntu:`, etc.)
- **Podman OCI:** OCI images from registries (Docker Hub, GHCR, etc.)

MicroVM runtimes (Cloud Hypervisor, Firecracker) are deferred per ADR-006 and ADR-022; image handling for those is out of v0.1 scope.

Considered wrapping these in a unified Helling image format or inventing an image catalogue API.

## Decision

Helling invents no custom image format, image wrapper, or image catalogue. Each runtime consumes its native image format via its existing tooling:

- Incus images: managed via Incus API (proxy through `/api/incus/*`)
- Podman images: managed via Podman libpod API (proxy through `/api/podman/*`)

App templates = Podman Compose YAML files stored in SQLite. Instance templates = Incus profiles. No Helling-specific template format.

## Consequences

- Zero Helling-specific image management code
- Users interact with image tools they already know (Incus remotes, `podman pull`)
- Incus image caching, layer dedup, and remote servers work out of the box
- OCI registry auth is Podman's concern, not Helling's
- When microVM support lands post-v0.1, its image handling will be decided in the ADR that supersedes ADR-006; no preemptive format work required
