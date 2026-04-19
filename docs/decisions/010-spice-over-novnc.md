# ADR-010: SPICE as in-browser VM console protocol

> Status: Accepted (re-affirmed 2026-04-20)

## Context

Helling needs a browser-native VM console that remains aligned with Incus VM console semantics and avoids per-VM launch overrides.

Incus VM VGA console sessions are SPICE-based. The upstream API and CLI flows for `type=vga` are built around SPICE sockets and operation websocket bridging.

Using noVNC as a primary path requires a `raw.qemu` VNC override per VM, which introduces portability and lifecycle risks and conflicts with the project goal of minimizing runtime behavior that diverges from upstream Incus defaults.

## Decision

Use SPICE as the default in-browser VM console protocol in v0.1.

Implementation:

1. `hellingd` proxies Incus VGA operation websockets without protocol translation.
2. WebUI uses a SPICE-capable browser client (`spice-html5` class) for VM VGA sessions.
3. Helling does not inject `raw.qemu` VNC overrides into VM definitions.
4. External SPICE client workflows remain compatible for troubleshooting.

## Consequences

- Browser VM console path stays aligned with Incus `type=vga` behavior
- Helling avoids per-VM `raw.qemu` drift and migration fragility
- WebUI must maintain SPICE browser client compatibility and websocket handling
- No fallback VNC control plane is introduced in v0.1
