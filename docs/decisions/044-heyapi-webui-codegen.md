# ADR-044: hey-api/openapi-ts for WebUI code generation

> Status: Accepted (2026-04-20)

## Context

Helling currently generates frontend API client code from api/openapi.yaml using orval.

ADR-043 is superseded for v0.1; api/openapi.yaml is the spec-first source of truth for Helling-owned endpoints.
The WebUI generator should align with this contract flow and simplify frontend client/runtime dependencies.

Current constraints:

- Keep api/openapi.yaml as committed generated artifact.
- Keep TanStack Query usage in WebUI.
- Reduce generator/runtime complexity and dependency surface.

## Decision

Adopt hey-api/openapi-ts for WebUI code generation.

Scope:

- Replace orval-based generation for Helling-owned API endpoints.
- Generate fetch-based client and typed SDK from api/openapi.yaml.
- Generate TanStack Query integration artifacts from the same contract.

Out of scope:

- Changes to backend routing architecture.
- Changes to CLI generation (oapi-codegen client remains).

## Why

- Better fit with a generated OpenAPI artifact workflow.
- Cleaner fetch-native runtime model for modern frontend stacks.
- Maintains typed SDK + query integration while reducing generator complexity.

## Compatibility requirements

1. Generated output must live under web/src/api/generated.
2. Existing WebUI API usage patterns must remain type-safe.
3. TanStack Query integration must remain available.
4. Build, lint, and typecheck must pass without orval/axios/refine runtime coupling.

## Migration plan

### Phase 0: Prototype

1. Add hey-api/openapi-ts configuration at web/hey-api.config.ts.
2. Generate code from api/openapi.yaml.
3. Validate representative read + write API calls in existing WebUI components.

Exit criteria:

- Generation succeeds deterministically.
- TypeScript passes with noEmit.
- No regression in request/response typing.

### Phase 1: Full replacement

1. Replace package scripts and codegen invocation from orval to openapi-ts.
2. Remove obsolete generator dependencies where no longer needed.
3. Update frontend QA checks to assert generated code freshness.

Completion criteria:

- No remaining orval generation path in repo workflow.
- Generated artifacts are committed and stable in PR diffs.

## Consequences

Easier:

- Cleaner generator pipeline tied to code-first contract artifact.
- Reduced API runtime coupling.

Harder:

- One-time migration of generated import paths and helper usage.
- Validation of generated query hooks/options against existing UI usage.

## Follow-up documents

- docs/design/tools-and-frameworks.md
- docs/design/openapi-pipeline.md
- docs/standards/coding.md
- docs/standards/quality-assurance.md
