# ADR-043: Huma via humago for Helling-owned HTTP layer

> Status: Accepted (2026-04-20)

## Context

Helling currently uses a docs-first contract process centered on hand-authored OpenAPI and generated clients.

Current repository status is pre-alpha and documentation-first, with implementation not yet started.

The most expensive near-term quality gate in this phase has been OpenAPI drift and hand-authored schema hygiene:

- request/response examples
- field-level descriptions and constraints
- consistency with generated clients and handler intent

Helling must preserve existing accepted architecture decisions:

- ADR-014 proxy-first routing
- ADR-015 native upstream response pass-through
- ADR-040 stdlib net/http ServeMux
- ADR-041 URI major versioning under /api/v1

## Decision

Adopt Huma for Helling-owned endpoints using the humago adapter on top of net/http ServeMux.

Scope:

- Huma manages only Helling-owned routes under /api/v1/\*.
- Incus and Podman pass-through routes remain plain http.Handler mounts.
- OpenAPI remains committed as a generated artifact, not hand-authored source.

Implementation constraints:

- Keep ServeMux as the top-level router.
- Preserve /api/v1 URI versioning policy.
- Preserve proxy pass-through behavior for /api/incus/\* and /api/podman/\*.

## Why

- Eliminates hand-maintained spec drift as a class of failure.
- Keeps ADR-040 intact through humago integration.
- Reduces duplicate work between schema authoring and handler implementation.
- Improves contract fidelity for generated clients by deriving OpenAPI from typed handlers.

## Required compatibility work

1. Implement Helling error envelope transformer in `apps/hellingd/api/envelope.go` (~50 LOC):
   - Map Huma/native errors to Helling's `ErrorEnvelope` from `docs/spec/api.md`.
   - Keep external wire format stable (`error.code`, `error.message`, optional `error.details`, `meta.request_id`).
   - Do not expose RFC7807 payloads externally.
   - Preserve status-code semantics (400/401/403/404/409/422/429/5xx) and deterministic error-code mapping.
2. Implement success envelope wrapper guidance:
   - Define a generic envelope type (or helper) for `data` + `meta` and keep `request_id` always present.
   - Support list responses with pagination metadata (`next_cursor`, `limit`, `has_more`) in `meta`.
   - Ensure handlers only return typed domain payloads while wrapper logic remains centralized.
3. Keep vacuum contract policy as a regression gate:
   - Continue running `vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml` in CI.
   - Treat generated OpenAPI as source-of-truth artifact for review and drift detection.
   - Retain custom intent rules (writeOnly, ULID patterns, envelope constraints, pagination shape) to catch accidental type/tag regressions.

## Implementation roadmap

### Phase 0: Two-endpoint spike (hard gate before broad rollout)

1. POST /api/v1/auth/login
   - Body validation constraints
   - 200/202/401/429 behavior
   - Error envelope mapping
2. GET /api/v1/users
   - Cursor pagination
   - List envelope shape
   - Stable operationId and tags

Exit criteria:

- Generated OpenAPI lints at 100/100 using api/.vacuum.yaml.
- Envelope format matches docs/spec/api.md for success and error paths.
- Existing proxy path model remains unchanged.
- No regression against ADR-040 routing model.

If any criterion fails, stop rollout and remediate compatibility gaps before broader migration.

### Phase 1: Full adoption for Helling-owned endpoints

1. Convert all Helling-owned handlers under `/api/v1/*` (~34 endpoints) to Huma operation definitions.
2. Preserve proxy mounts (`/api/incus/*`, `/api/podman/*`) as untouched plain `http.Handler` pass-through.
3. Generate and commit `api/openapi.yaml` from Huma during `make generate`.
4. Keep downstream codegen unchanged:
   - `oapi-codegen` client for CLI
   - `orval` hooks/models for frontend
5. Enforce CI gates:
   - `make check-generated`
   - vacuum ruleset pass
   - no behavior drift in envelope, auth, and pagination contracts

Completion criteria:

- All Helling-owned endpoints served through Huma with equivalent behavior.
- Generated artifact is reviewable and stable in PRs.
- CLI and frontend generation pipelines remain green without manual OpenAPI edits.

## Artifact evolution

| Artifact               | Before ADR-043                                 | After ADR-043 acceptance             | After Phase 1 completion                                       |
| ---------------------- | ---------------------------------------------- | ------------------------------------ | -------------------------------------------------------------- |
| `api/openapi.yaml`     | Hand-authored source contract                  | Declared generated contract artifact | Fully generated from Go types via Huma and committed           |
| Helling-owned handlers | Manual `net/http` + docs-first schema coupling | Migration target defined             | Huma operation handlers for all ~34 Helling endpoints          |
| Spec quality gate      | Manual hygiene burden; drift-prone             | Code-first intent adopted            | 100/100 by construction + custom intent-rule regression checks |
| CLI client generation  | `oapi-codegen` from hand-authored spec         | No tool change                       | `oapi-codegen` from generated `openapi.yaml`                   |
| Frontend hooks/models  | `orval` from hand-authored spec                | No tool change                       | `orval` from generated `openapi.yaml`                          |

## Consequences

Easier:

- Lower maintenance overhead for contract upkeep.
- Tighter coupling between implementation types and generated API docs.
- Faster iteration for adding fields/endpoints with constraints.

Harder:

- Introduces framework dependency in HTTP layer.
- Requires explicit custom error and envelope transformation.
- Some advanced OpenAPI customization may require post-generation patching.

## Follow-up documents to update when this ADR is accepted

- docs/design/openapi-pipeline.md
- docs/standards/standards-quality-assurance.md (OpenAPI gate semantics)
- docs/roadmap/implementation-guide.md
- docs/roadmap/plan.md
