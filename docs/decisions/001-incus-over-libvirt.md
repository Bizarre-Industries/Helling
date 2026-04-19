# ADR-001: Incus over go-libvirt

> Status: Accepted

## Context

Original architecture used go-libvirt + netlink + Corosync/Pacemaker for VM management. This required ~10 separate packages reimplementing what a single tool already provides: VM lifecycle, container lifecycle, OCI support, storage pools, networks, clustering, images, snapshots, backup, and migration.

## Decision

Use Incus (`lxc/incus/v6/client`) as the sole compute engine. Incus wraps QEMU/KVM for VMs and LXC for system containers, providing a unified Go client that maps 1:1 to its REST API. Never call libvirt, QEMU, or LXC directly.

## Consequences

- One Go client replaces 10 packages
- VMs, system containers, and OCI containers through one API
- Storage, networking, clustering, images, snapshots, backup, migration included
- Locked to Incus release cadence and feature set
- No raw QEMU access (use `raw.qemu` config key for edge cases)
- Incus socket path varies by distro — must be configurable
