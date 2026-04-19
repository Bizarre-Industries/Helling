# OpenAPI Pipeline

**Date:** 2026-04-15
**Supersedes:** Previous version that described a 300-endpoint spec.

---

## The Pipeline

Helling now uses a code-first OpenAPI flow for Helling-owned routes and keeps proxy routes unchanged (ADR-014, ADR-043).

```text
Go request/response structs + validation tags + doc comments
        │
        ▼
       Huma operations on net/http ServeMux
      (Helling-owned /api/v1/* endpoints only)
        │
        ▼
  Build/generate step emits api/openapi.yaml
        │
        ▼
   Commit generated contract artifact for review
        │
     ┌──────────────┴──────────────┐
     ▼                             ▼
 oapi-codegen (CLI client)      orval (frontend hooks/models)
     │                             │
   helling CLI commands         web/src/api/generated/*

Proxy pass-through paths stay plain handlers:
/api/incus/* and /api/podman/* are not managed by Huma.
```

Operational scope:

- Huma manages approximately 34 Helling-owned endpoints under `/api/v1/*`.
- Incus/Podman proxy paths remain untouched and continue native upstream pass-through behavior.

Outcome:

- Manual OpenAPI upkeep drops from roughly 14 hours per iteration cycle to near-zero.
- Contract drift shifts from hand-authored YAML risk to typed Go compile-time and generation-time checks.

---

## Backend: Huma as contract source

Huma operation definitions and typed structs are now the canonical source for Helling-owned API contract shape.

- Validation constraints are declared in Go tags.
- Field and operation descriptions are declared in code comments/metadata.
- Generated OpenAPI reflects implementation by construction.
- Envelope behavior is preserved through compatibility wrappers/transformers.

---

## CLI: oapi-codegen client

Generates a typed Go HTTP client:

```go
// GENERATED
func (c *Client) ListUsers(ctx context.Context) (*http.Response, error)
func (c *Client) CreateSchedule(ctx context.Context, body CreateScheduleJSONRequestBody) (*http.Response, error)
```

Each Cobra command is a thin wrapper (~15 lines).

---

## Frontend: orval

Generates React Query hooks for Helling-specific endpoints:

```typescript
// GENERATED
export const useListUsers = () => useQuery({ queryKey: ["users"], queryFn: ... });
export const useCreateSchedule = () => useMutation({ mutationFn: ... });
```

Config:

```typescript
// web/orval.config.ts
export default defineConfig({
  helling: {
    input: { target: "../api/openapi.yaml" },
    output: {
      target: "src/api/generated/endpoints",
      schemas: "src/api/generated/models",
      client: "react-query",
      mode: "tags-split",
      baseUrl: "/api/v1",
      override: {
        mutator: { path: "./src/api/custom-fetcher.ts", name: "customFetch" },
      },
    },
  },
});
```

---

## What's Generated vs What's Written

### Generated (never edit, regenerate from spec)

| Artifact                             | Generator            | Output                                          |
| ------------------------------------ | -------------------- | ----------------------------------------------- |
| OpenAPI contract artifact            | Huma                 | `api/openapi.yaml`                              |
| Go API client (for CLI)              | oapi-codegen client  | `cli/internal/client/client.gen.go` (~3k lines) |
| TypeScript types + React Query hooks | orval                | `web/src/api/generated/`                        |
| Shell completions                    | Cobra built-in       | `helling completion bash/zsh/fish`              |
| Man pages                            | Cobra doc generation | `helling(1)`                                    |

### Written by hand

| Artifact                        | Lines      |
| ------------------------------- | ---------- |
| Proxy middleware (the core)     | ~300       |
| Helling handler implementations | ~500       |
| Auth service                    | ~1,000     |
| CLI commands                    | ~500       |
| Frontend pages                  | ~5,000     |
| **Total hand-written**          | **~7,300** |

---

## Makefile

```makefile
generate:
  cd apps/hellingd && go generate ./...
    cd apps/helling-cli && oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
    cd web && bunx orval

check-generated: generate
    @git diff --exit-code apps/hellingd/api/*.gen.go apps/helling-cli/internal/client/*.gen.go web/src/api/generated/ || \
        (echo "Generated code is stale. Run 'make generate'" && exit 1)
```
