# ADR-034: Lima-based development environment for macOS/Windows contributors

> Status: Accepted (2026-04-19)
>
> Superseded-in-precedence-by: ADR-052 (2026-04-30) — Parallels Desktop is now the primary documented macOS dev/test VM. Lima is retained as a fallback path for contributors without Parallels.

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

## Update (2026-04-30)

ADR-052 has reordered the macOS contributor story. Parallels Desktop is now the primary documented path; Lima remains a fully supported fallback for contributors who cannot run Parallels (license cost, policy). The two paths share the same `vm:*:*` task namespace shape so commands are interchangeable. See ADR-052 for the rationale and the deploy-path tooling.
