# v0.2 Generated Artifact Triage

**Date:** 2026-05-06
**Scope:** Dirty-tree generated API artifacts for the v0.2 Platform Core work.

## Finding

The TypeScript API client diffs under `web/src/api/generated/` are intentional generated output from the expanded v0.2 OpenAPI contract. The generated index now exports v0.2 users, schedules, webhooks, firewall, events, and audit operations, matching the new paths and schemas in `api/openapi.yaml`.

## Verification

- `task check:openapi` passed against `api/openapi.yaml`.
- `cd web && bun run gen:api` completed successfully.
- SHA-256 manifests of `web/src/api/generated/**` before and after `bun run gen:api` were identical, so the checked-out generated TypeScript client is stable with the current OpenAPI contract.
- `cd web && bun run check` passed after low-risk complexity refactors.

## Notes

- Do not hand-edit `web/src/api/generated/**`; change `api/openapi.yaml` and regenerate instead.
- The broader tree is intentionally dirty with v0.2 implementation work, so this triage does not claim that every generated path in `Makefile GENERATED_PATHS` is ready to commit as a release unit.
