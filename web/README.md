# Helling WebUI

The Helling Datacenter Console — a single-page React app served by [hellingd](../apps/hellingd) and edged by [Caddy](../docs/decisions/037-caddy-edge-service.md).

> Stack reference: [`docs/spec/webui-spec.md`](../docs/spec/webui-spec.md). Stack lock-in: [ADR-051](../docs/decisions/051-webui-stack-lock-antd-pro-components.md). Audit: [`docs/audits/webui-2026-04-27.md`](../docs/audits/webui-2026-04-27.md).

## Bootstrap

```sh
cd web
bun install
```

`bun install` runs the `prepare` lifecycle script, which executes `bun run gen:api` to generate the typed API client from [`api/openapi.yaml`](../api/openapi.yaml) into `src/api/generated/`.

The generated directory is gitignored (`.gitignore` line `web/src/api/generated/`). Without the codegen step, imports like `./api/generated/sdk.gen` fail with module-not-found. This is the cause of audit finding F-41.

## Daily commands

```sh
bun run dev          # Vite dev server on :5173 (proxies /api to hellingd)
bun run build        # tsc -b + vite build → dist/
bun run gen:api      # regenerate ./src/api/generated/ from api/openapi.yaml
bun run check        # biome check (lint + format)
bun run fmt          # biome format --write
```

## Regenerating the API client

Whenever `api/openapi.yaml` changes (route shape, schema, or error envelope), regen:

```sh
bun run gen:api
```

CI fails the build if the generated artifacts drift from the spec — the `make check-generated` target at the repo root verifies this.

## Architecture

This WebUI is mid-migration. The current codebase is hand-rolled (`shell.jsx`, `infra.jsx`, `pages.jsx`, `pages2.jsx`, `app.jsx`); the locked target is **antd 6 + @ant-design/pro-components + @ant-design/charts + @tanstack/react-query** per [ADR-051](../docs/decisions/051-webui-stack-lock-antd-pro-components.md). Migration phases are tracked in [`docs/roadmap/checklist.md`](../docs/roadmap/checklist.md).

Until migration completes:

- New pages **must** be added on the spec'd stack (antd + pro-components), not the hand-rolled foundation.
- Existing pages stay on the hand-rolled stack until ported in batches.
- Theme is configured via `src/styles/app.css` today; will move to `src/theme/tokens.ts` driving an antd `<ConfigProvider>` once ADR-051 Phase 3 lands.

## Auth model

Per [`docs/spec/auth.md`](../docs/spec/auth.md) §2.2:

- **Access token** lives in memory only (`src/api/auth-store.ts`). Never localStorage. XSS-safe.
- **Session cookie** lives in an `httpOnly`, `Secure`, `SameSite=Lax` cookie set by hellingd.
- A page reload starts unauthenticated in v0.1 because there is no separate `/auth/refresh` endpoint.

## Tooling

- **bun**: package manager + script runner. Use bun, not npm/yarn/pnpm.
- **vite**: dev server + bundler.
- **biome**: lint + format. Pre-commit hooks (lefthook) enforce.
- **typescript**: strict mode. Mixed `.tsx`/`.jsx` today (audit F-09); migration to all `.tsx` is Phase 6.
- **@hey-api/openapi-ts**: SDK codegen from OpenAPI YAML.

## Useful entry points

| File                               | Role                                                                        |
| ---------------------------------- | --------------------------------------------------------------------------- |
| `index.html`                       | Vite entry, viewport meta + viewport-too-small gate                         |
| `src/main.tsx`                     | Root render, QueryClient setup, root ErrorBoundary                          |
| `src/app.jsx`                      | App component, routing state, auth subscription                             |
| `src/shell.jsx`                    | TopBar, Sidebar/ResourceTree, primitives, mock data                         |
| `src/infra.jsx`                    | Modals, toasts, charts                                                      |
| `src/pages.jsx` + `src/pages2.jsx` | All page components (split historically; audit F-05)                        |
| `src/error-boundary.tsx`           | Class-based ErrorBoundary used at root + per-route                          |
| `src/api/client.ts`                | hey-api fetch client + interceptors                                         |
| `src/api/auth-store.ts`            | In-memory access-token store + change events                                |
| `src/api/queries.ts`               | TanStack Query hooks for the few real-API integrations                      |
| `src/styles/app.css`               | App stylesheet (will collapse into antd ConfigProvider tokens post-ADR-051) |

## Reporting issues

Open an issue against the repo with the `webui` label. The audit catalog (`docs/audits/webui-2026-04-27.md`) lists known findings by F-ID; reference the F-ID in your report if it overlaps.
