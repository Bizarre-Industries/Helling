# ADR-013: Incus Project Limits for VM/CT Quotas

> Status: Accepted (2026-04-14)

## Context

Custom quota enforcement in hellingd would duplicate what Incus projects already provide. Incus projects support `limits.cpu`, `limits.memory`, `limits.disk`, `limits.instances`, `limits.containers`, `limits.virtual-machines`.

## Decision

Use Incus project limits for VM/CT resource quotas. Map Helling project quotas to Incus project config keys.

## Consequences

- Quotas enforced at the hypervisor level (stronger than application-level checks)
- No custom quota tracking code needed
- Helling projects map 1:1 to Incus projects
- Dashboard reads limits from Incus project config
