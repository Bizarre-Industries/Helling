# ADR-010: SPICE Client over noVNC for VM VGA Console

> Status: Accepted (2026-04-14)

## Context

Incus uses SPICE protocol for VM VGA console access. noVNC implements VNC protocol. These are incompatible — noVNC literally cannot connect to an Incus SPICE console.

## Decision

Use spice-js or spice-html5 for VGA console in the browser. Drop noVNC entirely.

## Consequences

- VGA console works correctly with Incus VMs
- SPICE provides better features (clipboard, USB redirect, audio) than VNC
- Need to add spice-js or spice-html5 as frontend dependency

## Addendum (2026-04-15)

Tested `@novnc/novnc` v1.7.0-beta. The top-level `await` bundling issue **is resolved** in Vite v8 (rolldown). The 190 KB noVNC bundle was included cleanly with zero build errors.

However, the **protocol incompatibility remains the blocking issue**: Incus VM consoles speak SPICE-over-WebSocket, not VNC/RFB. noVNC v1.7 would connect to the Incus WebSocket but fail immediately at handshake. This ADR stands.
