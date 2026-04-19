# ADR-021: ISO-Only Installation

> Status: Accepted

## Context

The previous architecture maintained two runtime paths:

1. **Docker try-it mode:** A Dockerfile, a `devauth` build tag that bypassed PAM authentication, TCP listener support, mode detection logic, conditional feature flags, and graceful degradation for missing Incus/Podman. This created a crippled demo that showed Podman container management through a web UI — making Helling look like "Portainer with extra steps." Users never saw VMs, clustering, storage pools, SPICE console, firewall, or backups — the actual product.

2. **Bare metal mode:** The real product, running on Debian 13 with Incus and Podman.

Maintaining two paths required: Docker-specific entrypoint scripts, mode detection (`Docker` vs `BareMetal`), `devauth.go` (auth bypass for Docker), TCP listener option alongside Unix socket, conditional nil checks on 20+ handlers (`if h.Incus == nil`), and a Dockerfile + CI for Docker image publishing.

Proxmox doesn't have a Docker demo. You download the ISO and install it. Nobody evaluates a hypervisor in a container.

## Decision

One install method: boot the Helling ISO. The ISO is a Debian 13 derivative with Incus, Podman, and all Helling components pre-installed. Boot it, answer 3 questions (hostname, disk, admin password), and it's running.

- Incus is always present. hellingd assumes it.
- Podman is always present. hellingd assumes it.
- systemd is always present. hellingd assumes it.
- No mode detection. No conditional features. No graceful degradation.
- No TCP listener. hellingd listens on a Unix socket. Period.
- No `devauth` build tag. PAM authentication always.
- No Dockerfile. No Docker image publishing.

For development and CI testing, use a Lima VM or Vagrant box with the same ISO or a Debian 13 base with Incus + Podman pre-installed. This is a real test environment, not a fake one.

First impressions for prospective users: documentation, screenshots, and a 2-minute video showing ISO boot → install → dashboard → create VM → SPICE console.

## Consequences

**Easier:**
- One code path, not two
- No mode detection, no conditional features, no nil checks on services
- No Docker-specific files (Dockerfile, entrypoint.sh, devauth.go)
- Config simplified (no TCP listen, no Docker conditionals)
- CI tests against a real environment (Lima VM with Incus)
- hellingd startup is simpler: connect to sockets, start serving

**Harder:**
- Can't "try before you install" (but a video demo + docs serve this purpose)
- Development requires a VM with Incus (but Lima makes this trivial on macOS/Linux)
- CI requires VM-based testing (but this catches real bugs that Docker testing misses)
- Higher barrier to first experience (but the right barrier for a hypervisor platform)
