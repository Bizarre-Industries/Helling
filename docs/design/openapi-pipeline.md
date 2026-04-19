# OpenAPI Pipeline

**Date:** 2026-04-15
**Supersedes:** Previous version that described a 300-endpoint spec.

---

## The Pipeline

Helling has a small OpenAPI spec (~40 endpoints) covering only Helling-specific features. Everything else (instances, containers, storage, networks) goes through the proxy to Incus/Podman sockets (ADR-014).

```
                    api/openapi.yaml
                  (~40 Helling endpoints)
                          │
         ┌────────────────┼────────────────┐
         │                │                │
    oapi-codegen     oapi-codegen        orval
   (strict-server)    (client)       (react-query)
         │                │                │
   ~25 Go handlers    ~15 CLI          TS hooks
   (business logic)   commands       for Helling API
         │                │                │
    ┌────┴────┐      helling CLI    hellingClient
    │         │                     (Helling hooks)
  Helling   Proxy                        +
  handlers  middleware            incusClient
    │         │                  podmanClient
 hellingd  Unix sockets         (direct fetch)
           (Incus + Podman)
```

**One spec change → `make generate` → backend + CLI + frontend Helling hooks updated.**

Incus/Podman types are NOT generated from this spec. The frontend uses thin type definitions (`incusTypes.ts`) or generates types from the Incus/Podman swagger specs separately.

---

## Backend: oapi-codegen strict-server

Generates a typed Go interface from the spec:

```go
// GENERATED — never edit
type StrictServerInterface interface {
    Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error)
    ListUsers(ctx context.Context, request ListUsersRequestObject) (ListUsersResponseObject, error)
    CreateSchedule(ctx context.Context, request CreateScheduleRequestObject) (CreateScheduleResponseObject, error)
    // ~25 more
}
```

We implement business logic only:

```go
func (s *Handlers) ListUsers(ctx context.Context, req ListUsersRequestObject) (ListUsersResponseObject, error) {
    users, err := s.auth.ListUsers()
    if err != nil {
        return nil, newAPIError(500, "INTERNAL_ERROR", "failed to list users")
    }
    return ListUsers200JSONResponse{Data: users}, nil
}
```

The generated code handles: route registration, JSON decoding, parameter parsing, response encoding, error formatting.

Config:
```yaml
# apps/hellingd/oapi-codegen.yaml
package: api
output: api/helling.gen.go
generate:
  strict-server: true
  chi-server: true
  models: true
```

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

| Artifact | Generator | Output |
|---|---|---|
| Go server interface + types + router | oapi-codegen strict-server | `api/helling.gen.go` (~2k lines) |
| Go API client (for CLI) | oapi-codegen client | `cli/internal/client/client.gen.go` (~3k lines) |
| TypeScript types + React Query hooks | orval | `web/src/api/generated/` |
| Shell completions | Cobra built-in | `helling completion bash/zsh/fish` |
| Man pages | Cobra doc generation | `helling(1)` |

### Written by hand

| Artifact | Lines |
|---|---|
| Proxy middleware (the core) | ~300 |
| Helling handler implementations | ~500 |
| Auth service | ~1,000 |
| CLI commands | ~500 |
| Frontend pages | ~5,000 |
| **Total hand-written** | **~7,300** |

---

## Makefile

```makefile
generate:
    cd apps/hellingd && oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
    cd apps/helling-cli && oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
    cd web && bunx orval

check-generated: generate
    @git diff --exit-code apps/hellingd/api/*.gen.go apps/helling-cli/internal/client/*.gen.go web/src/api/generated/ || \
        (echo "Generated code is stale. Run 'make generate'" && exit 1)
```
