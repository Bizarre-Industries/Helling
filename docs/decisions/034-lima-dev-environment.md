# ADR-034: Lima-based development environment for macOS/Windows contributors

> Status: Accepted (2026-04-19)

## Context

Helling targets Debian-first host behavior. Non-Linux contributor environments need a reproducible Linux VM workflow.

## Decision

Use Lima virtual machines as the recommended dev environment for macOS/Windows.

- Debian VM baseline
- Go/Bun/Incus tooling installed in VM
- VS Code remote workflow supported

## Consequences

- More consistent local behavior across contributor platforms
- Additional local setup cost versus native development
