---
name: openapi-workflow
description: Spec-first API workflow for Helling. api/openapi.yaml is the single source of truth — never hand-edit generated code, always regen via `make generate`, and verify drift with `make check-generated` before committing.
origin: helling
---

# OpenAPI-driven workflow (Helling)

`api/openapi.yaml` is the contract. Three artifacts are generated from it:

- `apps/hellingd/api/*.gen.go` — chi-compatible server stubs + models (oapi-codegen v2)
- `apps/helling-cli/internal/client/*.gen.go` — Go HTTP client
- `web/src/api/generated/` — TypeScript client + React Query hooks (hey-api/openapi-ts)

## When to activate

- User asks to add, modify, or remove a Helling API endpoint
- Editing `api/openapi.yaml`
- Editing files matching `apps/*/api/*.gen.go` or `web/src/api/generated/**` (these are READ-ONLY)
- `make check-generated` reports drift
- New OpenAPI schema, response, or parameter is needed

## The contract

1. **Never hand-edit generated files.** They will be overwritten on next `make generate`.
2. **Spec changes happen first.** Edit `api/openapi.yaml`, then run `make generate`, then implement the handler / consume the new client.
3. **Drift is a CI failure.** `make check-generated` runs `make generate` and refuses to pass if `git status` shows uncommitted changes after.
4. **Lint the spec.** `vacuum lint --ruleset api/.vacuum.yaml api/openapi.yaml` must pass with zero errors before merge (run via `make check`).

## Standard loop

```bash
# 1. Edit the spec
$EDITOR api/openapi.yaml

# 2. Regenerate everything (server, Go client, TS client)
make generate

# 3. Verify the spec is clean
vacuum lint --ruleset api/.vacuum.yaml api/openapi.yaml

# 4. Confirm drift is absent
make check-generated

# 5. Implement the handler in apps/hellingd/internal/server/<thing>_handlers.go
# 6. Add tests
# 7. Commit spec + generated + handler + tests in one PR
```

## Adding a new endpoint — step by step

1. Define the operation in `api/openapi.yaml` under `paths:`. Required keys:
   - `operationId` (camelCase, used for generated function name)
   - `tags` (`auth | instances | operations | meta` for v0.1)
   - `summary` (one line)
   - `description` (paragraph)
   - request body schema (if write op) referencing `components/schemas/`
   - all expected response codes referencing `components/responses/` or inline schemas
2. Add new schemas to `components/schemas/`. Reuse existing ones aggressively — every new schema doubles the generated surface area.
3. Run `make generate`. The new server stub appears in `apps/hellingd/api/server.gen.go` as `Get<OperationId>` etc.
4. Wire the handler in `apps/hellingd/internal/server/<feature>_handlers.go`. Pattern (from existing handlers):
   ```go
   func (s *Server) handleFoo(w http.ResponseWriter, r *http.Request) {
       user, ok := UserFromContext(r.Context())
       if !ok { writeError(w, http.StatusUnauthorized, "no_session", "no session"); return }
       // ... call store/incus, shape response
       writeJSON(w, http.StatusOK, response)
   }
   ```
5. Register the route in `routes()` in `apps/hellingd/internal/server/server.go` inside the authed `r.Group`.
6. Add tests in the same package, file `<feature>_handlers_test.go`. Use `httptest.NewServer(srv.Handler())` + `loginCookie` helper from `server_test.go`.
7. Run `make check && make check-generated`.

## Removing or breaking an endpoint

- Breaking changes to `/v1/*` endpoints are FORBIDDEN within the v0.1 line.
  Add a non-breaking alternative (new field; deprecate the old one) or wait for `/v2/`.
- To deprecate, add `deprecated: true` to the operation in the spec and add
  the deprecation reason to the description. Do not remove until next major.

## Common drift causes

| Cause                                           | Fix                                                                               |
| ----------------------------------------------- | --------------------------------------------------------------------------------- |
| Edited a `*.gen.go` file by hand                | `make generate` to overwrite; never edit again                                    |
| Forgot to run `make generate` after spec change | Run it; commit the generated diff                                                 |
| TypeScript client out of date                   | `cd web && bun install && cd .. && make generate`                                 |
| `oapi-codegen` version mismatch                 | Pinned via `apps/hellingd/go.mod` tool dep; run `cd apps/hellingd && go mod tidy` |

## Don't

- Don't add endpoints outside the `/v1/` group in v0.1.
- Don't bypass the generated client to call hellingd directly from web/cli — defeats the type safety the workflow exists for.
- Don't add `additionalProperties: true` to object schemas without a stated reason — generated TS types lose the closed-world guarantee.
- Don't re-order properties inside schemas to "clean up" — diff churn is a review hazard and the order is preserved by the generator.
