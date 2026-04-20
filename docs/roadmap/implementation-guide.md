# Helling Implementation Guide

This guide defines the current implementation sequence aligned with accepted ADRs and active migration decisions.

## Current Direction

- ADR-043: Huma plus humago for Helling-owned HTTP layer
- ADR-044: hey-api/openapi-ts for WebUI code generation
- ADR-014/015/040/041 preserved

## Execution Order

1. Freeze hand-authored OpenAPI changes.
2. Implement Huma spike endpoints (`POST /api/v1/auth/login`, `GET /api/v1/users`).
3. Generate and validate `api/openapi.yaml` with vacuum.
4. Expand Huma coverage to all Helling-owned routes.
5. Migrate WebUI API generation to hey-api/openapi-ts.
6. Land tooling and hook package.
7. Complete drift cleanup in docs/standards/spec/design.

## Mandatory Validation Loop

For each phase:

```bash
make generate
make check-generated
make fmt-check
make lint
make test
vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml
```

## Source of Truth

Use `docs/roadmap/migration-manifest-2026-04-20.md` as phase checklist and status tracker.
