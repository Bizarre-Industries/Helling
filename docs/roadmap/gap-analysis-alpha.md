# Phase 0 — v0.1.0-alpha gap analysis

Exit gates for Phase 0 per [docs/roadmap/plan.md](plan.md). All green.

| Gate              | Evidence                                                                                                                                                                              |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go build          | `go build ./apps/hellingd/... ./apps/helling-cli/... ./apps/helling-proxy/...` exits 0                                                                                                |
| Generators stable | `make generate && git diff --stat apps/hellingd/api apps/helling-cli/internal/client web/src/api/generated` is empty                                                                  |
| Router eliminated | `router.go` deleted; replaced by [bootstrap.go](../../apps/hellingd/internal/api/bootstrap.go) + generated `ServerInterface` via 62-method adapter                                    |
| Auth gate         | [bootstrap_test.go](../../apps/hellingd/internal/api/bootstrap_test.go) walks every registered route and asserts `unauthRoutes` coverage (blocks typo-opens)                          |
| Legacy web/api    | `web/src/api/{types,queries}.ts` deleted; split into [helling/](../../web/src/api/helling/), [incus/](../../web/src/api/incus/), [podman/](../../web/src/api/podman/) behind a barrel |
| `make lint`       | 0 issues                                                                                                                                                                              |
| `make spec-lint`  | 0 errors (296 warnings — quality-score polish deferred to v0.2.0)                                                                                                                     |
| `make test`       | 10 pkgs ok, race-clean                                                                                                                                                                |
| Vitest            | 6 tests passing: tokenStore (×2), api barrel re-exports (×2), incus hook envelope unwrap, LoginPage smoke                                                                             |
| TODO/FIXME sweep  | 0 hits in non-test `apps/**/*.go`                                                                                                                                                     |
| Nilaway           | Job wired with `continue-on-error: true`; baseline findings tracked to v0.8.0                                                                                                         |

## Deliberate deferrals

- **Thumbnail routes** (`handlers_thumbnail.go`): absent from `api/openapi.yaml`, unregistered in `bootstrap.go`, guarded by a `//go:build phase1` tag so dead code does not ship on default builds. Unblocks when Phase 1 extends the spec and wires the UI tile.
- **OpenAPI spec warnings** (296): 112 missing `example`s, 87 missing component descriptions, 43 camelCase violations, 20 missing operation descriptions. Polish belongs to the v0.2.0 "Polish & Integrations" milestone; `make spec-lint` still gates on errors=0.
- **Orval for Helling endpoints**: [helling/hooks.ts](../../web/src/api/helling/hooks.ts) is still hand-written on top of axios. Orval-generated hooks (`web/src/api/generated/`) exist for reference; gradual migration is scheduled with the Phase 1 detail-page work.
- **Nilaway zero**: job runs with `continue-on-error: true`. Triaging to zero is a v0.8.0 gate per roadmap.

## Adapter shim — maintenance contract

`server_adapter.go` exists because 62 chi-style handlers predate the
generated `ServerInterface` and would each need signature changes to
accept the typed `params` structs. The compile-time assertion
`var _ apigen.ServerInterface = (*serverAdapter)(nil)` fails the build
whenever `api/openapi.yaml` adds, removes, or re-signatures an operation:
developers MUST add or update the matching adapter line in the same PR
that regenerates `apps/hellingd/api/server.gen.go`. Handlers adopt typed
params one at a time as their bodies change, dropping the adapter line in
the same PR — tracked under v0.2.0.

## Pre-push command bundle

```sh
make generate && make check-generated
make fmt-check lint spec-lint test web-test
act -W .github/workflows/integration.yml    # final gate before push
```
