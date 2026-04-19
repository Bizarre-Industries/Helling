# ADR-016: Helling CLI for Helling Features Only

> Status: Accepted

## Context

The previous architecture defined ~392 CLI commands that wrapped every Incus and Podman operation: `helling instance list`, `helling container start`, `helling storage pool create`, etc. Each command built an HTTP request to hellingd, which then called the Incus/Podman Go client. This created a chain of three tools (`helling` → `hellingd` → `incus`/`podman`) where one would do.

Users already have `incus` and `podman` CLIs installed — they ship in the Helling ISO. These CLIs are maintained by their respective upstream projects, have complete feature coverage, shell completions, and extensive documentation.

## Decision

The `helling` CLI covers only Helling-specific features that `incus`, `podman`, and `kubectl` don't provide:

```
helling auth login/logout/token        # Helling JWT auth
helling user list/create/delete        # Helling user management (PAM)
helling schedule list/create/delete    # Backup scheduling (systemd timers)
helling webhook list/create/delete     # Webhook management
helling bmc scan/power/sensors         # BMC management (bmclib)
helling k8s create/list/delete/kubeconfig  # K8s cluster provisioning (k3s via cloud-init)
helling system info/upgrade/config     # System management
helling firewall list/create/delete    # Host nftables rules
helling version                        # Version info
helling completion bash/zsh/fish       # Shell completions
```

For everything else:

- `incus` for instances, storage, networks, profiles, projects, cluster, images
- `podman` for containers, pods, images, volumes, networks, secrets, compose
- `kubectl` for Kubernetes workloads (after `helling k8s kubeconfig`)

## Consequences

**Easier:**

- ~15 commands instead of ~392 — trivial to maintain
- No CLI commands go stale when upstream adds features
- Man page for `helling(1)` is concise and useful
- Users get the full power of upstream CLIs with their shell completions
- Generated Go client is small (~3k lines instead of ~29k)

**Harder:**

- Users need to know three CLIs (but they're standard tools they likely already know)
- `helling auth token` must produce a token that works with `incus` and `podman` (or users configure those separately)
- No single unified CLI for all operations
