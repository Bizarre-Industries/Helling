# OpenAPI Pipeline

**Date:** 2026-04-20
**Supersedes:** 2026-04-20 Huma code-first migration plan for v0.1

---

## The Pipeline

Helling v0.1 uses a spec-first contract flow. `api/openapi.yaml` is the hand-authored source of truth; generated Go and TypeScript clients must be regenerated from that file.

```text
api/openapi.yaml
        |
        v
Validate contract with vacuum
        |
   +----+-------------------------+
   |                              |
   v                              v
oapi-codegen (Go server + CLI) hey-api/openapi-ts (WebUI SDK + Query)
   |                              |
apps/hellingd/api              web/src/api/generated
apps/helling-cli/internal
```

Proxy pass-through paths stay plain handlers:

- /api/incus/\*
- /api/podman/\*

These routes bypass generated Helling-owned handlers to preserve ADR-014 and ADR-015 behavior.

---

## Scope and Invariants

- `api/openapi.yaml` manages Helling-owned endpoints under /api/v1/\*.
- chi remains the top-level router for the current v0.1 implementation.
- URI major versioning remains /api/v1 (ADR-041 preserved).
- OpenAPI remains committed in-repo and is intentionally reviewed by humans.

---

## Generation Model

### Source of contract truth

For Helling-owned routes, contract truth lives in `api/openapi.yaml`:

- request and response schemas
- validation constraints
- operation metadata (summary, description, operationId, tags, responses)

### Generated artifact

- Go server/model code: `apps/hellingd/api/*.gen.go`
- Go CLI client code: `apps/helling-cli/internal/client/*.gen.go`
- Web client code: `web/src/api/generated/**`
- Any PR changing API behavior must update `api/openapi.yaml` first, then regenerate downstream artifacts.

### Downstream codegen

- CLI: oapi-codegen client continues unchanged.
- WebUI: hey-api/openapi-ts generates fetch client, SDK, schemas, and TanStack Query options.

---

## Why this flow

- Keeps the API contract reviewable before implementation changes.
- Lets backend, CLI, and WebUI generation share one source of truth.
- Avoids code-first migration churn until v0.1 is stable.

---

## Verification Gates

1. Validate the committed OpenAPI contract.
2. Run vacuum against api/.vacuum.yaml.
3. Regenerate CLI and WebUI clients from committed api/openapi.yaml.
4. Ensure no stale generated diff remains in CI.

Reference command:

```bash
vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml
```

---

## Artifact Ownership

### Generated artifacts (never hand-edit)

- apps/hellingd/api/\*.gen.go
- apps/helling-cli/internal/client/\*.gen.go
- web/src/api/generated/\*\*

### Hand-authored artifacts

- api/openapi.yaml
- Handler implementation in apps/hellingd/
- Proxy pass-through handlers for /api/incus/\* and /api/podman/\*
- Contract policy docs in docs/spec/
