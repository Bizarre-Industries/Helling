# ADR-002: Debian 13 over Fedora

> Status: Accepted

## Context

Needed a stable Linux distribution for bare metal deployment. Fedora has newer packages but shorter lifecycle. Debian has longer support, stable packaging, and is the foundation of Proxmox (15+ years proven).

## Decision

Target Debian 13 "Trixie" as the base OS. Incus packages from Zabbly target Debian. IncusOS is Debian 13. AppArmor is the default MAC.

## Consequences

- Zabbly Incus packages work out of the box
- AppArmor (not SELinux) for mandatory access control
- Longer release cycle = more stable base
- Older kernel/package versions vs Fedora (acceptable trade-off)
- Package format is .deb, not .rpm
