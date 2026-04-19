# ADR-043: Huma via humago for Helling-owned HTTP layer

> Status: Proposed (2026-04-20)

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

1. Implement Helling error envelope transformer:
   - Keep ErrorEnvelope format from docs/spec/api.md.
   - Do not adopt default RFC7807 output externally.
2. Implement generic success envelope typing:
   - Keep data + meta shape (request_id, page metadata).
3. Ensure generated OpenAPI still passes api/.vacuum.yaml custom ruleset.

## Spike plan (must pass before acceptance)

Run a focused prototype on two endpoints:

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

If any criterion fails, revert to hand-authored OpenAPI workflow for v0.1.

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
