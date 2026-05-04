# ADR-032: Two fixed roles for v0.1

> Status: Accepted (2026-04-19)

## Context

Role definitions are currently spread across docs with optional custom-role language mixed into v0.1 scope. The first release needs a stable, testable RBAC baseline that matches the implemented user schema.

## Decision

Helling v0.1 supports exactly two built-in roles:

- `admin`: full management access
- `user`: standard operational access within assigned scope

The persisted model is `is_admin`; API and CLI surfaces may render that as the derived role label `admin` or `user`.

Custom roles, an `auditor` role, policy composition, and per-resource role authoring are out of scope for v0.1.

## Consequences

- Predictable permission matrix for API, CLI, and WebUI
- Reduced implementation risk in early releases
- Fewer authorization edge cases to test
- Future custom-role support can be added in v0.5+ without changing v0.1 guarantees
