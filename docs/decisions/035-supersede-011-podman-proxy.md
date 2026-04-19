# ADR-035: Supersede ADR-011 with proxy-only Podman access

> Status: Accepted (2026-04-19)

## Context

ADR-011 originally focused on Podman bindings selection. Current architecture uses proxy-first patterns for both Incus and Podman.

## Decision

Supersede ADR-011 with explicit proxy-only Podman access for v0.1.

- No Podman Go bindings in baseline implementation
- Podman operations are forwarded via Unix socket proxy

## Consequences

- Smaller dependency surface and code footprint
- Native upstream API behavior preserved
