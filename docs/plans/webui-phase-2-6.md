# Helling WebUI — Phase 2 through Phase 6 Plan

> Captured: 2026-04-27
> Source: `docs/audits/webui-2026-04-27.md`
> Stack lock: ADR-051 (`docs/decisions/051-webui-stack-lock-antd-pro-components.md`)
> Status: Plan accepted; Phase 2 next.

## Context

The Helling WebUI v0.2 audit (`docs/audits/webui-2026-04-27.md`) catalogued 51 findings across 9 areas. Three landed already this session:

- **Phase 0 (stack lock)** committed by ADR-051 — WebUI commits to antd 6 + @ant-design/pro-components per `docs/spec/webui-spec.md`. No new pages on the hand-rolled stack.
- **Phase 1 (safety fix-pass)** shipped at commits `5fd90aa`, `1992e3c`, `68fb40b` on main — closes 12 of 51 findings (F-15, F-22, F-36, F-37, F-38, F-39, F-41, F-44, F-45, F-47, F-50, R-03).

The **39 remaining findings** are sequenced into Phases 2–6 per audit Section 13. This plan turns each phase into concrete file edits, ordering decisions, and verification gates. The work runs ~4–6 weeks elapsed.

**Critical constraint surfaced during planning:** `api/openapi.yaml` L777 shows `eventsSse` is currently a **snapshot-GET**, not a real Server-Sent-Events stream — _"Full Server-Sent-Events streaming lands in v0.1-beta."_ Phase 3's F-42 SSE consumer is therefore **blocked on backend** (hellingd ships actual streaming). Phase 3 ships the consumer hook in two stages: snapshot-poll fallback first, then upgrade to EventSource once backend lands the stream.

## Scope

In scope: Phase 2 (untangle foundation) → Phase 6 (hardening). All on the current hand-rolled stack except Phase 4 which is the antd migration spike per ADR-051.

Out of scope: any new feature work, the migration _port_ (Phase 4 only spikes 3 pages — full port is post-this-plan), backend SSE implementation (own ticket).

---

## Phase 2 — Untangle the foundation (~1 sprint)

**Goal:** Make the codebase reviewable and lazy-loadable before Phase 3 + 4 multiply the diff load. Pure mechanical wins. No behavior change.

**Closes:** F-05, F-07, F-08, F-09 (partial), F-29, F-30, F-40, F-41 (already shipped — verify).

### 2A. Split `pages.jsx` + `pages2.jsx` into per-route folders [F-05]

Current state: 33 Page\* components in two flat 3,848 + 4,752-line files (`web/src/pages.jsx`, `web/src/pages2.jsx`). Page sections per `web/src/app.jsx` `CRUMBS`:

- Datacenter: Dashboard, Instances, InstanceDetail, Containers, ContainerDetail, Kubernetes, Cluster, Console, NewInstance
- Resources: Storage, Networking, Firewall, FirewallEditor, Images, Backups, Schedules, Templates, BMC, Marketplace, FileBrowser
- Observability: Metrics, Alerts, Logs
- Admin: Users, UserDetail, Audit, Ops, Settings, RBAC
- Auth: Login, Setup
- Search: Search, SearchResults

**Target structure:**

```text
web/src/pages/
  dashboard/       index.tsx
  instances/       index.tsx   detail.tsx   new.tsx   console.tsx
  containers/      index.tsx   detail.tsx
  kubernetes/      index.tsx
  cluster/         index.tsx
  storage/         index.tsx
  networking/      index.tsx
  firewall/        index.tsx   editor.tsx
  images/          index.tsx
  backups/         index.tsx
  schedules/       index.tsx
  templates/       index.tsx
  bmc/             index.tsx
  marketplace/     index.tsx
  file-browser/    index.tsx
  metrics/         index.tsx
  alerts/          index.tsx
  logs/            index.tsx
  users/           index.tsx   detail.tsx   rbac.tsx
  audit/           index.tsx
  ops/             index.tsx
  settings/        index.tsx
  auth/            login.tsx   setup.tsx
  search/          index.tsx   results.tsx
  index.ts                     (barrel — named ES exports for `app.jsx`)
```

Convert to `.tsx` while at it (subsumes audit F-09 partially). Drop the per-file `/* eslint-disable */` banner from migrated files — let biome a11y lints fire (audit F-08).

Modify `web/src/app.jsx` to import via the `pages/index.ts` barrel instead of destructuring from `window.*`. This is the primary F-07 reduction (window coupling map below in 2B).

### 2B. Drop `window.*` coupling [F-07]

Verified `window.*` write sites:

- `web/src/shell.jsx`: `window.openModal`, `window.closeModal`, `window.toast`, `window.__nav`, `window.useTick`, `window.__jitter`, `window.pushTask`, `window.instanceAction`, `window.useStore`, `window.__getLatency`, `window.pushAlert`, `window.__searchQuery`
- `web/src/app.jsx`: `window.openModal`, `window.closeModal`, `window.__nav`
- `web/src/main.tsx`: `(window as any).useDashboardCounts` — comment in main.tsx already calls this out as a backlog item

**Replace with two zustand-style stores** (avoid heavy deps; use `useSyncExternalStore`):

- `web/src/stores/ui-store.ts` — modal state, palette open, drawer open, toast queue, openModal/closeModal/toast actions. Hooks: `useToast()`, `useModal()`, `useNav()` (the latter wraps `setPage`).
- `web/src/stores/system-store.ts` — tick, latency, instance/task/alert mock pools (will move to `queries.ts` in Phase 3 — short-lived store).

Removes `(window as any)` cast from `main.tsx`. Removes `Object.assign(window, ...)` calls. After Phase 2, grep `web/src` for `window\.` should hit only browser-DOM uses (`window.matchMedia`, `window.location.reload`, `window.addEventListener` for `auth:session-expired` event).

### 2C. Lazy-load each route + Suspense boundary [F-29]

Update `web/src/main.tsx` and the new `web/src/app.jsx`:

```tsx
const PageDashboard = lazy(() => import("./pages/dashboard"));
// ... per route
```

Wrap `body` (already in `<ErrorBoundary>` from Phase 1) with `<Suspense fallback={<PageSkeleton />}>`. Add a tiny `web/src/skeleton.tsx` that renders a shell-shaped placeholder using existing `--h-surface` tokens.

**Verify bundle drops:** baseline `dist/assets/index-*.js` is 1.26MB pre-this-plan. Target after Phase 2: initial chunk under 600KB, with per-route chunks under 100KB each except PageConsole (heavy VNC canvas).

### 2D. Lucide icon barrel [F-30]

Today `web/src/shell.jsx` does `import * as LucideIcons from 'lucide-react'` then `LucideIcons[pascal]` lookup — defeats tree-shake.

Create `web/src/icons.ts`:

```ts
import { Activity, AlertTriangle, ArrowRight /* ~40 names */ } from "lucide-react";
export const ICONS = { activity: Activity /* ... */ } as const;
export type IconName = keyof typeof ICONS;
```

Update the `I` component in `web/src/shell.jsx` (or its new home `web/src/primitives/icon.tsx`) to look up from `ICONS` instead of `LucideIcons`. Audit fix F-51 (stroke width drift outside the wrapper) lands free if every direct lucide import is removed.

### 2E. Test scaffold + 3 smoke tests [F-40]

Add deps:

```bash
bun add -d vitest @testing-library/react @testing-library/jest-dom jsdom
```

Add `web/vitest.config.ts` (point at `src`, jsdom env, setup file imports `@testing-library/jest-dom`). Add `"test": "vitest run"` and `"test:watch": "vitest"` to `web/package.json` scripts.

**Three smoke tests minimum:**

1. `web/src/pages/instances/__tests__/keyboard.test.tsx` — j/k navigates focus, Enter opens detail, x triggers Stop confirm, s triggers Snapshot. Uses `@testing-library/user-event`.
2. `web/src/components/__tests__/confirm-modal.test.tsx` — `ConfirmModal` with `confirmMatch="vm-1"` keeps the Confirm button disabled until "vm-1" is typed. Critical regression guard for Phase 1's F-15 work.
3. `web/src/api/__tests__/auth-store.test.ts` — `setAccessToken` fires `auth:session-changed`; `clearAccessToken` resets; `subscribeAuthChange` returns an unsubscriber that actually unsubscribes. Regression guard for Phase 1's F-37/F-38 work.

CI: extend `.github/workflows/quality.yml` to run `bun --cwd web run test`.

### 2F. Verify F-41 fresh-clone build still works after restructure

Phase 1 added `prepare = "bun run gen:api"` to `web/package.json`. Restructure must not break it. Manual verify:

```sh
git clean -fdx web/
cd web && bun install && bun run dev
```

Should boot Vite without module-not-found.

### Phase 2 verification

- [ ] `find web/src/pages -name 'index.tsx' | wc -l` ≥ 25 (per-route folders exist)
- [ ] `grep -r 'window\.' web/src --include='*.ts*' | grep -v 'matchMedia\|location\|addEventListener\|removeEventListener'` returns 0 hits
- [ ] `bun run build` — initial chunk under 600KB gzipped
- [ ] `bun run test` — 3 tests pass
- [ ] `bun run check` — biome a11y errors no longer suppressed by per-file disable banners
- [ ] `bun run dev` from a fresh clone — boots without module-not-found
- [ ] Manual smoke: every route loads, no whitescreen, ErrorBoundary still catches throws

---

## Phase 3 — Real data layer + SSE consumer (~1 sprint, partial-blocked)

**Goal:** Replace mock arrays with hooks. Define canonical types from real API shapes. Wire SSE consumer (snapshot-poll first, EventSource upgrade when backend ships).

**Closes:** F-01, F-02, F-03, F-04, F-31, F-42 (partial), F-43.

### 3A. Mock adapter behind hooks [F-01, F-31]

Today `web/src/shell.jsx` exports `INSTANCES`, `CONTAINERS`, `TASKS`, `ALERTS`, `BACKUPS`, `SNAPSHOTS` arrays imported directly by ~20 page sites. `useStore()` is a global re-render trigger.

**Replace with:**

- Add `bun add -d msw` (Mock Service Worker).
- Move all mock arrays into `web/src/api/mocks/seed.ts` (typed seeds, no React).
- New `web/src/api/mocks/handlers.ts` defines MSW handlers: `GET /api/incus/1.0/instances` returns seed → mutation handlers (POST/PUT/DELETE) update seed in place + emit a fake `events.onmessage` event.
- New `web/src/api/queries.ts` extensions: add `useInstancesQuery` (already exists, expand types per F-43), `useContainersQuery` (exists), plus `useTasksQuery`, `useAlertsQuery`, `useBackupsQuery`, `useSnapshotsQuery`. Each is a thin `useQuery` over the existing `authedFetch` pattern.
- Mutations: `useInstanceMutation()`, `useTaskMutation()` etc. via `useMutation`. Optimistic update + rollback on error. Reuses Phase 1 toast bus for the undo affordance.

After this, page components stop importing global arrays. They consume hooks. The current `useStore()` global Set goes away — TanStack handles per-key invalidation.

### 3B. Type harmonization [F-02, F-43]

`web/src/api/queries.ts` `IncusInstance` is a 3-field stub. Real Incus instances carry `architecture`, `config`, `created_at`, `description`, `devices`, `ephemeral`, `expanded_config`, `expanded_devices`, `last_used_at`, `location`, `profiles`, `project`, `stateful`, plus a nested `state` substructure.

**Approach:**

- Keep current 3-field type as a _summary_ projection used by dashboard counts.
- Add full `IncusInstanceDetail` type to `web/src/api/types.ts` (new file). Hand-write from official Incus REST docs unless we can pull their OpenAPI directly.
- Add a single normalizer at the API boundary: `normalizeIncusInstance(raw): IncusInstance` (the projection) + the detail flows through unmapped.
- Mirror for Podman: add full `PodmanContainerDetail` separate from list-summary.
- Mock seed shape **must match** real shape. Seeds use the same `IncusInstanceDetail` type.

This collapses F-02 (casing) into F-43 (shape) — one consolidated migration.

### 3C. SSE consumer (two-stage) [F-42, F-03]

**Stage 1 (immediate, ships in Phase 3): snapshot-poll fallback.**

Backend `eventsSse` is currently a snapshot GET (`api/openapi.yaml` L777). Build `web/src/api/use-events-stream.ts`:

```ts
export function useEventsStream() {
  // Poll /api/v1/events?limit=50 every 5s, dedupe by event id,
  // dispatch each new event by type via queryClient.setQueryData /
  // invalidateQueries on the matching key. Returns nothing — fire-and-forget.
}
```

Mounted from a single hook at App root (or QueryClientProvider sibling). Replaces the 1.5s mock interval in shell.jsx (F-03 follow-up).

**Stage 2 (queued, behind backend): EventSource.**

When hellingd ships actual SSE on the same path:

```ts
const es = new EventSource("/api/v1/events");
es.addEventListener("task.update", (e) => queryClient.setQueryData(/* ... */));
// reconnect-on-error with exponential backoff
```

Drop in place. Single switch behind a feature flag in `web/src/api/use-events-stream.ts`. Snapshot-poll stays as the fallback for SSE-blocked corporate proxies (per audit reframe R-01).

**Cross-team: open issue against hellingd** to deliver real SSE (links the openapi.yaml `description` line that says "lands in v0.1-beta"). This plan does not include the backend work but flags the dependency.

### 3D. Loading / error / empty states [F-04]

Now that auth boundary exists (Phase 1 F-38) and queries can fail loudly:

- Generic `<QueryStateView query={q}>{(data) => ...}</QueryStateView>` wrapper in `web/src/components/query-state.tsx`. Renders skeleton on `isLoading`, error card on `isError`, empty card on `data.length === 0`, children otherwise.
- The empty-state primitive in current `infra.jsx` becomes `EmptyState` in `web/src/components/empty-state.tsx` with new `variant: "empty" | "error"` prop.
- Apply `<QueryStateView>` across every list page (instances, containers, storage, networking, firewall, images, backups, schedules, templates, etc.).

### 3E. Wire real Incus + Podman + Helling endpoints beyond dashboard counts

Per audit: 49 OpenAPI ops in `api/openapi.yaml`, 2 wired today (instances + containers, both list-only).

Phase 3 expands wiring incrementally:

| Page               | Endpoints                                                        | Notes                                |
| ------------------ | ---------------------------------------------------------------- | ------------------------------------ |
| Instances list     | `GET /api/incus/1.0/instances?recursion=1` (have)                | Already wired; expand fields per 3B. |
| Instance detail    | `GET /api/incus/1.0/instances/{name}?recursion=1`                | New hook `useInstanceQuery(name)`.   |
| Instance state ops | `PUT /api/incus/1.0/instances/{name}/state` (start/stop/restart) | Mutation hooks.                      |
| Snapshots          | `GET .../instances/{name}/snapshots`, `POST .../snapshots`       | New hooks.                           |
| Containers list    | `GET /api/podman/libpod/containers/json?all=true` (have)         | Already wired.                       |
| Container detail   | `GET /api/podman/libpod/containers/{name}/json`                  | New hook.                            |
| Storage pools      | `GET /api/incus/1.0/storage-pools`                               | New.                                 |
| Networks           | `GET /api/incus/1.0/networks`                                    | New.                                 |
| Tasks              | `GET /api/v1/tasks` (Helling-owned, check OpenAPI)               | New.                                 |
| Alerts             | `GET /api/v1/alerts` (Helling-owned)                             | New.                                 |
| Audit              | `GET /api/v1/audit` (Helling-owned)                              | New.                                 |

Each page gains real-data hook + mutation set + `QueryStateView` wrapper. Mock seed data continues to fill via MSW for dev where backend is unreachable.

### 3F. F-45 already shipped — verify

Phase 1 set `refetchOnWindowFocus: true` in `main.tsx` QueryClient defaults. Ensure no new query in Phase 3 reverts this default.

### Phase 3 verification

- [ ] `grep -r "import.*INSTANCES\|import.*CONTAINERS\|import.*TASKS\|import.*ALERTS" web/src/pages` returns 0 hits
- [ ] MSW dev handler boots: `bun run dev` works without hellingd running, all pages render with seeded data
- [ ] Real-mode: with hellingd at `:8080` and `/api/v1/auth/login` returning a token, instance list shows real Incus instances (not mock)
- [ ] Stage-1 SSE consumer polls `/api/v1/events?limit=50` every 5s, dispatches by event type to query cache (verify via `preview_console_logs`)
- [ ] List pages show skeleton on first load, error card when proxy 5xx's, empty card when zero results
- [ ] No regression in Phase 1 work: login still wires, ErrorBoundary still catches, viewport gate still triggers

---

## Phase 4 — Layout primitives + ADR-051 antd migration spike (~1 sprint)

**Goal:** Three-page spike onto antd 6 + pro-components per ADR-051. Validates the token bridge before committing to full port.

**Closes:** F-06 (partial — three pages), F-11, F-20, F-21, F-23, F-24 (subsumed).

### 4A. Add antd deps

```bash
cd web
bun add antd@^6 @ant-design/pro-components @ant-design/charts dayjs
```

Bundle budget: validate first-load JS stays under 400KB gzipped after Phase 2 lazy-loading + antd. If not, file a follow-up to lazy-load pro-components subset.

### 4B. Token bridge

New `web/src/theme/tokens.ts`:

```ts
import type { ThemeConfig } from "antd";
export const helling: ThemeConfig = {
  token: {
    colorPrimary: "var(--bzr-lime)",
    colorBgLayout: "var(--bzr-void)",
    colorBgContainer: "var(--bzr-ash)",
    fontFamily: "Hanken Grotesk, system-ui, sans-serif",
    fontFamilyCode: "JetBrains Mono, ui-monospace, monospace",
    borderRadius: 4,
    fontSize: 13
  },
  components: {
    Table: { rowHoverBg: "rgba(255,255,255,0.04)", cellPaddingBlock: 8 }
  }
};
```

`web/src/main.tsx` wraps `<App />` (already inside `<ErrorBoundary>`) with `<ConfigProvider theme={helling}>`. Order: `ErrorBoundary > ConfigProvider > QueryClientProvider > App`.

### 4C. Spike three pages

Pick representatives (per ADR-051):

1. **Instances list** → ProTable with sort/filter/density/column-toggle for free. Closes audit F-26 (table semantics) for this page.
2. **Instance Detail** → ProDescriptions + Tabs (closes audit F-23 page-header drift, F-25 modal a11y for detail-page modals, F-17 keyboard parity for tabs).
3. **New Instance wizard** → StepsForm with per-step validation + preview hooks (closes F-14 wizard validation).

Each spike page lives at `web/src/pages/<route>/index.tsx` next to its hand-rolled sibling (Phase 2 split). Feature-flag the antd version behind `localStorage.getItem('helling.spike') === '1'` so the team can A/B compare without breaking demos.

### 4D. Light mode resolution [F-20]

After Phase 4B + the spike, light mode becomes a `ConfigProvider` `algorithm: theme.defaultAlgorithm` swap. Existing `body.light-mode` CSS class becomes vestigial — keep it during Phase 4 transition; remove in Phase 6.

Also resolves: load logo asset variant per theme (`mark-inverse.png` on dark, `mark.png` on light — needs new asset).

### 4E. Real router [F-11]

Concurrent with the spike: adopt **TanStack Router** (pairs naturally with TanStack Query already in the stack). Routes derived from ADR-051 spec section. Crumbs derive from a route → label map.

URL becomes the source of truth: `/instances`, `/instances/:name`, `/instances/:name/console`, `/containers/:name`, etc. Bookmarkable and refresh-safe.

This unblocks audit F-11. Replaces the current `setPage('instance:vm-1')` string-route hack.

### 4F. Hardcoded color hunt [F-21, F-24]

Once tokens flow through ConfigProvider, replace the 87 hardcoded color literals across `pages.jsx`/`pages2.jsx` (per Phase 2 split: across `web/src/pages/*`) with theme tokens or the new tint set:

- Add `--h-tint-hover`, `--h-tint-pressed`, `--h-tint-selected`, `--h-divider-soft` to `web/src/styles/app.css` `:root`.
- Map `--h-success` → `var(--bzr-success)` etc. (audit F-24).
- Sweep `rgba(255,255,255,0.0X)` → `var(--h-tint-...)`.

### Phase 4 verification

- [ ] antd ConfigProvider wraps app at root; visual signature preserved (stencil headings, lime accent, dot indicators, 1px borders, 4px radius)
- [ ] Bundle initial chunk under 400KB gzipped (or follow-up issue filed if not)
- [ ] Three spike pages on antd toggle via `localStorage.helling.spike = '1'` and visually match (or improve) the hand-rolled versions
- [ ] TanStack Router renders all routes with bookmarkable URLs; back/forward works; refresh preserves location
- [ ] Light mode reads `prefers-color-scheme` (Phase 1 F-44) → `ConfigProvider` algorithm swap shows the light theme without manual CSS class flip
- [ ] `grep -r 'rgba(255,255,255' web/src` returns 0 hits

---

## Phase 5 — Operator polish (~1 sprint)

**Goal:** The features the audit's operator-ergonomics section calls out as missing.

**Closes:** F-10, F-14, F-16, F-17, F-18, F-26 (full — applies a11y to tables not yet in pro-components), F-32, F-33, F-34, F-35.

### 5A. Backup SLA on instance list [F-33]

- Add `backupAge` column to ProTable (auto from instance metadata).
- New `web/src/components/sla-cell.tsx` colors by configured threshold (amber/red).
- Settings page gains an SLA configuration section: "backup older than X is amber, Y is red". Persist server-side per Helling settings spec.
- Backups page header: "**N instances out of SLA** · M failed scheduled backups last 24h".

### 5B. Dashboard "what changed" digest [F-32]

Replace the dashboard greeter with a digest stripe consuming the audit + tasks + alerts query keys. Click any segment to filter the destination page to that scope.

### 5C. Bulk-action menu [F-16]

Extend the toolbar Stop/Start (Phase 1 already supports Stop with confirmMatch ≥3) with Snapshot, Backup, Restart, Delete, Migrate. When >3 items, open a single tracked task in the drawer aggregating child task progress.

### 5D. Diff viewer [F-34]

`bun add react-diff-viewer-continued`. Use in: snapshot detail (vs current), cloud-init pre-apply (vs last applied), firewall rule preview (vs current rules).

### 5E. Sidebar collapse + real Recent + pin/unpin [F-10]

- Collapsible section headers persisted to settings.
- Real "Recent" backed by route history (last 8 routes).
- Pin/unpin from any row, persisted server-side.

### 5F. Keyboard parity on detail pages [F-17]

Standard kit: `1`–`9` jumps to tab N, `[`/`]` previous/next tab, `Esc` cancel/close, `Enter` primary action. Single keyboard overlay (`?` key) documents shortcuts.

### 5G. Toast inbox [F-18]

Persist last 20 toasts behind the bell, alongside alerts, in a "history" tab.

### 5H. Tables a11y for any non-ported pages [F-26 leftovers]

If Phase 4 only ported three pages, the other ~30 list pages still use div grids. Add `role="table"`/`role="row"`/`role="cell"` and `aria-sort` on header cells until they're ported in the post-this-plan full migration.

### Phase 5 verification

- [ ] Instance list shows Backup-age column with severity color
- [ ] Dashboard digest stripe replaces greeter; click navigates with filter
- [ ] Bulk-action menu fires aggregate task; partial-failure produces a single reconcilable view
- [ ] Snapshot detail shows diff against current
- [ ] Sidebar collapses; Recent updates as routes change; pinned rows persist
- [ ] Detail pages: `1`-`9` cycles tabs; `?` shows keyboard overlay
- [ ] Bell history tab lists last 20 toasts

---

## Phase 6 — Hardening + leftover decisions (~1 week)

**Goal:** Supply-chain + security hygiene + IA cleanup. Last mile.

**Closes:** F-09 (full), F-12, F-13, F-19, F-22 (responsive policy), F-46, F-47 (verified Phase 1; revisit if regressed), F-48, F-49, F-51 (verified Phase 2; revisit).

### 6A. CSP + noscript + SRI [F-46]

- `<meta http-equiv="Content-Security-Policy">` in `web/index.html` as belt-and-braces alongside Caddy headers: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'`. Tighten `style-src` to remove `unsafe-inline` once Phase 4 has eliminated inline styles.
- `<noscript>` already shipped in Phase 1 — verify still present.
- SRI policy documented in `CONTRIBUTING.md`. Required for any future CDN-loaded asset.

### 6B. Renovate / Dependabot config [F-49]

- New `.github/renovate.json` (or extend existing `.github/dependabot.yml`). Group bumps: react, query, build (vite/biome/typescript), antd. Weekly minor/patch; majors gated on manual review.

### 6C. HMR cleanup [F-48]

`web/src/shell.jsx` (or wherever the 1s tick lives after Phase 2 split) wraps intervals in `import.meta.hot?.accept(() => clearInterval(handle))`.

After Phase 3 SSE-consumer migration, the 1.5s mock-mutation interval is gone — only the 1s heartbeat tick may remain.

### 6D. IA cleanup [F-12, F-13]

- Firewall list + editor: consolidate into one page with list + drawer (Portainer model) or make editor a modal.
- Marketplace + Templates: rename — "Templates" → "VM Images", "Marketplace" → "App Catalog". Or merge into a single Catalog page with a Type filter.

### 6E. Toast-only landmines [F-19]

Convert `window.toast.info('Coming soon...')` calls to a `notImplemented('feature-name')` helper that:

- Greys the button (cursor: not-allowed).
- Toasts "Not yet wired" on click (warning style).
- Surfaces in a single grep target for future wiring.

### 6F. Responsive policy [F-22]

Phase 1 fixed the viewport meta and added the 1440-or-larger gate. Phase 6 confirms or relaxes:

- Either: stay with the current "1440 desktop minimum" hard gate (the audit's recommended path; current state).
- Or: add 1–2 breakpoints (sidebar collapses below 1280, console sidebar drops below 1180).

Decision: stay with 1440 desktop minimum. Document in `web/README.md`.

### 6G. Convert remaining `.jsx` to `.tsx` [F-09 full]

After Phase 2 split + Phase 4 spike, the only legacy `.jsx` files are the un-ported pages. Phase 6 converts them all to `.tsx` (no antd port — just the file extension flip + minimal Props typing).

### Phase 6 verification

- [ ] CSP meta + noscript + SRI policy in place
- [ ] Renovate runs weekly; first PR opened
- [ ] No HMR interval leak after 30 minutes of edits in dev mode
- [ ] Sidebar shows single Firewall entry with consolidated list+editor; Marketplace/Templates renamed or merged
- [ ] Disabled buttons via `notImplemented(...)` show greyed cursor + warning toast
- [ ] All `.jsx` files in `web/src` converted to `.tsx`; `find web/src -name '*.jsx' | wc -l` returns 0
- [ ] `bun run check` zero errors (Phase 1 had ~84 pre-existing biome errors; this is the cleanup gate)

---

## Cross-cutting risk register

1. **SSE backend dependency** — F-42 stage-2 blocked. Plan ships stage-1 (snapshot-poll) so progress isn't blocked on backend. Open a hellingd ticket as a first-day Phase 3 action.
2. **Bundle budget** — Phase 4 antd + pro-components could balloon initial chunk. Mitigated by Phase 2 lazy-loading + per-route chunks. Hard budget: 400KB gzipped initial. If exceeded after spike, file follow-up to lazy-load pro-components subset.
3. **Migration regression** — three-page spike (Phase 4) might miss visual signature (stencil/lime/dot). Mitigated by `localStorage.helling.spike = '1'` flag for A/B; ADR-051 reopens if the visual bar isn't met.
4. **Test scaffold churn** — adding vitest may surface flakiness in the existing j/k/Enter/x/s keyboard handler. Treat any test-uncovered bug as a Phase 1 follow-up commit, not a Phase 2 blocker.
5. **Type harmonization scope creep** — F-43 expands Instance/Container types to ~14 fields each. Tempting to over-engineer. Stick to: hand-write from official Incus REST docs once, no codegen for the proxy layer.
6. **Logout flow** — Phase 1's TopBar `onLogout` calls `clearAccessToken()` locally. It does **not** call `authLogout` on the server to revoke the refresh token cookie. Add to Phase 3 scope as an addendum: wire `useLogoutMutation` calling `POST /api/v1/auth/logout` then `clearAccessToken()`.

## Critical files (quick reference)

**Phase 1 (already shipped):**

- `web/src/api/auth-store.ts` — memory-only token
- `web/src/error-boundary.tsx` — typed boundary
- `web/src/main.tsx` — root render + QueryClient defaults
- `web/src/app.jsx` — auth subscription + theme bootstrap + density persistence
- `web/src/pages.jsx` — PageLogin wired, confirmMatch added
- `web/index.html` — viewport + noscript + ResizeObserver suppression dropped
- `web/src/styles/app.css` — viewport-too-small + prefers-\* media

**Phase 2 targets:**

- New: `web/src/pages/*/index.tsx` (per-route folders)
- New: `web/src/stores/ui-store.ts`, `web/src/stores/system-store.ts`
- New: `web/src/icons.ts`, `web/src/skeleton.tsx`
- New: `web/vitest.config.ts`, `web/src/**/__tests__/*.test.tsx`
- Modified: `web/src/app.jsx` (drop window.\* destructure, lazy + Suspense)
- Modified: `web/src/main.tsx` (drop `(window as any)` cast)

**Phase 3 targets:**

- Expanded: `web/src/api/queries.ts` — many new hooks
- New: `web/src/api/types.ts` (canonical Instance/Container shapes)
- New: `web/src/api/mocks/seed.ts`, `web/src/api/mocks/handlers.ts`, `web/src/api/mocks/browser.ts` (MSW)
- New: `web/src/api/use-events-stream.ts` (snapshot-poll → EventSource)
- New: `web/src/components/query-state.tsx`, `web/src/components/empty-state.tsx`

**Phase 4 targets:**

- New: `web/src/theme/tokens.ts`, `web/src/theme/index.tsx`
- Spike: `web/src/pages/instances/index.tsx` (ProTable), `web/src/pages/instances/detail.tsx` (ProDescriptions + Tabs), `web/src/pages/instances/new.tsx` (StepsForm)
- New: TanStack Router config replacing `app.jsx` route table

**Phase 5 / 6 targets:** scattered, see per-section file lists above.

## Existing utilities to reuse (do not reimplement)

- `web/src/api/auth-store.ts` `subscribeAuthChange` — Phase 2 ui-store should mirror this `useSyncExternalStore`-friendly pattern.
- `web/src/error-boundary.tsx` `ErrorBoundary` — Phase 2 Suspense fallback nests inside this; do not write a second boundary class.
- `web/src/api/queries.ts` `authedFetch` — Phase 3 hooks reuse this rather than building a parallel auth-injecting fetch.
- `web/src/infra.jsx` `EmptyState` — Phase 3 `<QueryStateView>` builds on this with an `error` variant.
- `web/src/infra.jsx` `ConfirmModal` `confirmMatch` — Phase 5 bulk Delete uses this; do not invent a new typed-confirm primitive.
- `docs/spec/webui-spec.md` ProTable / ProForm / StepsForm references — Phase 4 spike targets these specifically.

## Notes

- Each phase ships as 3–5 PRs to keep diff size reviewable. Do not bundle Phase 2 split + Phase 3 data-layer in one PR.
- Audit findings get closed in `docs/roadmap/checklist.md` as PRs land. Cross-reference by F-ID in commit messages (Phase 1 used `Refs: F-15, F-22, ...` format — keep this convention).
- The full antd port (post Phase 4 spike) is **not** in this plan. After Phase 4 ships and the spike validates, file a follow-up plan for the page-by-page port.
