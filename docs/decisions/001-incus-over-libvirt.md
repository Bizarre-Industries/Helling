# ADR-001: Incus over go-libvirt

> Status: Accepted

## Context

Original architecture used go-libvirt + netlink + Corosync/Pacemaker for VM management. This required ~10 separate packages reimplementing what a single tool already provides: VM lifecycle, container lifecycle, OCI support, storage pools, networks, clustering, images, snapshots, backup, and migration.

## Decision

Use Incus (`lxc/incus/v6/client`) as the sole compute engine. Incus wraps QEMU/KVM for VMs and LXC for system containers, providing a unified Go client that maps 1:1 to its REST API. Never call libvirt, QEMU, or LXC directly.

## Consequences

- One client replaces a fragmented stack of low-level VM/container packages.
- VMs and system containers are managed through a unified Incus API.
- QEMU features are accessed only through Incus config keys (`raw.qemu`, `raw.qemu.conf`). No direct QEMU process control and no libvirt API usage.
- Incus socket path varies by distro and must remain configurable.
