# ADR-012: Incus Network ACLs for VM/CT Firewalling

> Status: Accepted (2026-04-14)

## Context

The `google/nftables` Go library requires netlink and root access. Incus already manages nftables rules for its network bridges. Having Helling also modify nftables creates conflicts.

## Decision

- For VM/CT firewalling: Use Incus Network ACLs (security groups that Incus manages)
- For host/Podman firewalling: Shell out to `nft --json` CLI (simpler than Go library)
- Remove `google/nftables` Go dependency

## Consequences

- VM/CT firewall rules are managed by Incus, which handles nftables correctly
- Host firewall uses `nft --json` for read/write (no netlink, works in containers)
- Simpler codebase: ~100 lines of CLI wrapper vs ~300 lines of nftables library usage
- Dashboard shows both layers unified — user doesn't see the implementation difference
