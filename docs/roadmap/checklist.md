# Helling Release Checklist v4

> Proxy architecture (ADR-014). ISO-only (ADR-021). Every item has a verification command.

---

## v0.1.0-alpha Gate

### Proxy

Proxy middleware is wired in hellingd per ADR-014 (`apps/hellingd/internal/proxy/`). Internal CA + per-user cert issuance + per-request TLS Transport selection complete (PRs O-1..R). Remaining beta work is live verification against real Incus + Podman upstreams (docs/spec/internal-ca.md).

- [x] Proxy scaffold exists and forwards requests via httputil.ReverseProxy (unit + integration tests in `apps/hellingd/internal/proxy/`)
- [x] WebSocket upgrades pass through to upstream (covered by `TestProxy_WebSocketUpgradePassesThrough` in PR O-4)
- [x] Internal CA bootstrap + per-user cert issuance on userCreate (PRs O-1..O-5; gated behind `HELLING_CA_DIR`)
- [x] Per-user mTLS Transport selector (covered by `TestProxy_UserTLSProvider_ForwardsCert` + fallback variants in PR R / O-6)
- [x] `curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8443/api/incus/1.0/instances | jq '.metadata'` returns Incus instances (live verify via `task test:integration:vm` against the Parallels VM with `HELLING_VM_HOST` + `HELLING_VM_TOKEN` set; ADR-052)
- [x] `curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8443/api/podman/libpod/containers/json | jq '.[0].Names'` returns Podman containers (same target)
- [x] Unauthenticated request to proxy returns 401 (covered by `TestProxy_Unauthenticated_Returns401`)
- [x] Non-admin user gets per-user Incus identity via mTLS (per-user Transport selector — PR R / O-6)

### Auth

- [x] Setup → login → JWT → protected routes → refresh → TOTP → recovery codes: full flow (covered by `TestRealAuth_SetupLoginRefreshLogout`, `TestService_SetupThenLoginAndRefresh`, `TestRealTotp_EnrollVerifyLoginFlow`, `TestService_EnrollVerifyTOTP_Flow`, `TestService_CompleteMFA_RecoveryCodeFlow`)
- [x] Rate limiting: 6 failed logins → 429 (sliding-window limiter at `apps/hellingd/internal/auth/ratelimit.go`; covered by `TestService_LoginRateLimit` + `TestService_LoginRateLimit_ResetsAfterSuccess`; api maps `auth.ErrRateLimited` → `huma.Error429TooManyRequests` in `auth_real.go`)
- [x] API token: create → auth with token → revoke → rejected (covered by `TestRealApiTokens_CreateListRevoke` end-to-end including post-revoke 401 assertion)

### Build

- [x] `make build` succeeds with zero warnings (verified via `task check:go:build`)
- [x] `make test` passes (verified via `task test:go` + `task test:frontend`; race detector on; coverage 81.1% overall, floor 80%)
- [x] `make lint` clean (`task check:go:lint` golangci-lint; nilaway + exhaustive deferred to v0.1-beta automation tail)
- [x] `make generate` produces all generated files without error (`task gen` covers openapi + go cli + frontend + sqlc)
- [x] `make check-generated` — generated code matches spec (`task check:openapi:generated` green)
- [x] `cd web && bun run build` succeeds (vite 7 build green; bundle 482KB initial post-icon-barrel)

### Dev Environment (ADR-052)

- [x] Parallels Desktop dev VM bootstrap (`scripts/parallels-vm-bootstrap.sh`) provisions Debian 13 guest with Go, Bun, systemd, DBus, polkit, Incus, Podman — commit `7a7371c`
- [x] rsync inner-loop deploy task (`task vm:parallels:dev`) cross-builds linux/$(arch), syncs to VM, restarts hellingd, returns 0 — commit `7a7371c`
- [x] `.deb` release-gate deploy task (`task vm:parallels:release-test`) builds via reprepro (ADR-045), installs in VM, smoke passes — or skips cleanly if reprepro tooling not yet wired — commit `7a7371c`

### Code Hygiene

> Hygiene grep below excludes the `stub` keyword: it is a legitimate Huma-spike pattern (seed fixtures + spike comments) tracked separately. Real work-marker keywords (TODO/FIXME/not-implemented) must remain at zero.

- [x] No `docker/docker` in go.mod
- [x] No `google/nftables` in go.mod
- [x] No `go-co-op/gocron` in go.mod
- [x] No `devauth.go` exists
- [x] No `Dockerfile` exists in deploy/ (deploy/ does not exist; ISO-only per ADR-021)
- [x] No `router.go` (manual routes) exists
- [x] No `handlers_phase*.go` files exist
- [x] No `strict_handlers.go` (empty struct) exists
- [x] `grep -rn "TODO\|FIXME\|not implemented" apps/ --include="*.go" | grep -v _test.go | wc -l` = 0
- [x] `grep -rn "Docker mode\|Docker try-it\|devauth" docs/ --exclude-dir=decisions --exclude=checklist.md --exclude=plan.md | wc -l` = 0

### Spec

- [x] `vacuum lint --ruleset api/.vacuum.yaml api/openapi.yaml` — zero errors (score 100/100 against project ruleset; gate enforced by `task check:openapi`)
- [x] Every Helling endpoint has operationId, request/response schemas, error responses (49/49 operations parity gate green via `task check:parity`)
- [x] Every list endpoint follows cursor pagination contract (enforced via Huma operation registration + parity script; deviations require explicit exception)

### Dashboard

- [x] Dashboard loads, shows system stats (mocks for non-logged-in dev; real counts via `useDashboardCounts` when authed)
- [x] Dashboard instance + container counts pulled from `/api/incus/1.0/instances` + `/api/podman/libpod/containers/json` via the ADR-014 proxy
- [ ] Instance list page loads real Incus data (deferred to v0.1-beta; 32 other pages still on mocks)
- [ ] Container list page loads real Podman data (deferred to v0.1-beta)
- [ ] Storage page loads pool data (deferred to v0.1-beta)
- [ ] Network page loads network data (deferred to v0.1-beta)
- [x] Dashboard uses TanStack Query hooks via `web/src/api/queries.ts`, no raw fetch leaks in PageDashboard
- [x] No `VncConsole.tsx` exists
- [x] No stale noVNC-only console path assumptions remain

### CLI

- [x] `helling auth login` works (interactive + --username/--password; MFA branch handled via `/api/v1/auth/mfa/complete`)
- [x] `helling auth logout` clears local session
- [x] `helling auth whoami` decodes the stored JWT claims
- [x] `helling compute list` forwards to hellingd `/api/incus/1.0/instances` via the proxy (ADR-014)
- [ ] `helling user list` works (v0.1-beta)
- [ ] `helling system info` works (v0.1-beta)
- [x] `helling version` shows version + commit
- [x] No instance/container/storage/network/image CLI commands exist (compute, as the sole surfaced subcommand, forwards to Incus proxy rather than re-implementing per-resource CLIs)

### Automation

- [x] git-cliff produces CHANGELOG.md from commits (`task changelog` writes via `cliff.toml` template; CHANGELOG.md committed; prettier + markdownlint exclude generated file)
- [x] .devcontainer/devcontainer.json exists (Debian 12 base, Go 1.26 feature, Bun via postCreateCommand, ports 5173/8443/8006 forwarded)
- [x] Pre-commit hooks catch stale generated code (`task check:openapi:generated` + `task check:frontend:gen` invoked from CI; lefthook pre-push runs `task check`)

### WebUI Audit Phase 1 — Safety Fix-Pass (audit 2026-04-27) ✅ shipped

> Commits `5fd90aa`, `1992e3c`, `68fb40b` on main (2026-04-27).

- [x] **F-37** (security · spec): `web/src/api/auth-store.ts` stores access token in memory only (`docs/spec/auth.md` §2.2); refresh stays in httpOnly cookie set by server
- [x] **F-38** (ux): `PageLogin` calls `authLogin` operation from generated SDK; `app.jsx` initialises `authed=false`; MFA stage calls `authMfaComplete`
- [x] **F-39** (resilience): root `<ErrorBoundary>` wraps `<App />` in `main.tsx`; per-route boundary inside `App` around page body
- [x] **F-41** (dx): fresh-clone build works (`bun install && bun run dev` succeeds); `prepare` script runs `gen:api`; `web/README.md` documents codegen step
- [x] **R-03/F-22** (a11y · visual): `index.html` viewport meta is `width=device-width, initial-scale=1`; CSS gate hides `#root` below 1440px with friendly message
- [x] **F-15** (safety): destructive Delete actions (image Delete, cluster Shutdown, bulk Stop ≥3) require typed confirmation via `ConfirmModal` `confirmMatch`
- [x] **F-44** (a11y · theming): `app.css` has `prefers-reduced-motion: reduce` rule killing animations + transitions; first-load reads `prefers-color-scheme` when no theme stored
- [x] **F-45** (data freshness): QueryClient default `refetchOnWindowFocus: true`
- [x] **F-47** (security smell): global `ResizeObserver` warning suppression dropped from `index.html`
- [x] **F-50** (consistency): density toggle persists to localStorage like theme

### WebUI Audit Phase 2 — Foundation Untangle (audit 2026-04-27)

> Source: `docs/plans/webui-phase-2-6.md` Phase 2. Sub-tasks ship as separate commits; verify gates per sub-section.

- [x] **F-30 + F-51** (perf · 2D): `web/src/icons.ts` barrel; `shell.jsx` `I` component looks up from `ICONS`; bundle 1.26MB → 482KB initial chunk (gzip 265KB → 129KB) — commit `8a08cc5`
- [x] **F-40** (testing · 2E): vitest scaffold + `web/vitest.config.ts` + `src/test-setup.ts` + 3 smoke tests (auth-store F-37, error-boundary F-39, icons F-30); 14 tests / 543ms — commit `ab83985`
- [ ] **F-05** (arch · 2A): `pages.jsx` + `pages2.jsx` split into `web/src/pages/<route>/index.tsx` per-route folders; convert to `.tsx`; drop per-file `eslint-disable` banners
  - [x] auth subfolder: `web/src/pages/auth/login.tsx` (commit `ceb9cf4`) + `web/src/pages/auth/setup.tsx` (commit `1912bfc`); also extracted `web/src/primitives/icon.tsx` (Phase 2A pilot) and `web/src/primitives/switch.tsx` (Phase 2A continues)
  - [ ] datacenter subfolder: Dashboard, Instances, InstanceDetail, Containers, ContainerDetail, Kubernetes, Cluster, Console, NewInstance — needs Badge / Sparkline / MultiChart primitives + `legacy/mocks.ts` shim extracted first
  - [ ] resources subfolder: Storage, Networking, Firewall, FirewallEditor, Images, Backups, Schedules, Templates, BMC, Marketplace, FileBrowser
  - [ ] observability subfolder: Metrics, Alerts, Logs
  - [ ] admin subfolder: Audit (commit `8d1a3bf`) + Ops (commit `14caf91`) + Users (commit `<users-pending>`) shipped; UserDetail, Settings, RBAC pending
  - [ ] search subfolder: Search, SearchResults
- [ ] **F-07** (arch · 2B): replace `window.*` coupling with `web/src/stores/ui-store.ts` + `system-store.ts` using `useSyncExternalStore`; drop `(window as any)` cast from `main.tsx` — _scaffold landed `1367cad`; shell.jsx + main.tsx rewire pending_
- [ ] **F-29** (perf · 2C): each page lazy-loaded via `React.lazy`; `<Suspense fallback={<PageSkeleton />}>` wraps body; per-route chunks under 100KB
- [ ] **F-08** (hygiene · 2A side): biome a11y errors no longer suppressed by per-file disable banners
- [ ] **F-09** (arch · 2A side): all `web/src/pages/*` are `.tsx` (full TS migration of remaining `.jsx` is Phase 6)
- [ ] **2F**: fresh-clone build still works post-restructure (`git clean -fdx web/ && cd web && bun install && bun run dev`)

### WebUI Audit Phase 3 — Real data layer + SSE (audit 2026-04-27)

> Source: `docs/plans/webui-phase-2-6.md` Phase 3. Blocked on Phase 2 (`F-05` page split must complete so each page can swap mock-array imports for hooks). F-42 stage-2 SSE blocked on backend (`api/openapi.yaml` L777 says full streaming v0.1-beta) — ship snapshot-poll first.

- [ ] **F-01** (data · 3A): components stop importing `INSTANCES`/`CONTAINERS`/`TASKS`/`ALERTS`/`BACKUPS`/`SNAPSHOTS` directly; consume `useInstancesQuery`/`useContainersQuery`/`useTasksQuery`/`useAlertsQuery`/`useBackupsQuery`/`useSnapshotsQuery` hooks; mocks live behind MSW handlers in `web/src/api/mocks/`
- [ ] **F-02** (data · 3B): canonical `Instance` type + normalizer at API boundary; Incus `"Running"` ↔ mock `"running"` casing reconciled — _scaffold landed `1367cad` at `web/src/api/normalize.ts`; page consumers pending_
- [ ] **F-03** (data · 3C): 1.5s mock interval in `shell.jsx` removed; SSE consumer at `web/src/api/use-events-stream.ts` updates query cache via `queryClient.setQueryData` — _hook landed `1367cad`; shell.jsx interval removal + main.tsx wiring pending_
- [ ] **F-04** (ux · 3D): `<QueryStateView>` wrapper renders skeleton on loading, error card on error, empty card on empty; applied to every list page — _component landed `1367cad` at `web/src/components/QueryStateView.tsx`; per-page application pending_
- [ ] **F-31** (perf): subsumed by F-01 + F-42 — verify `useStore()` global re-render trigger removed once hooks ship
- [ ] **F-42** (data · 3C stage-1): `useEventsStream()` polls `GET /api/v1/events?limit=50` every 5s, dedupes by event id, dispatches by type — _hook landed `1367cad` at `web/src/api/use-events-stream.ts`; main.tsx mount pending_
- [ ] **F-42** (data · 3C stage-2): EventSource swap once hellingd ships real SSE on `/api/v1/events` (cross-team ticket)
- [ ] **F-43** (types): `IncusInstanceDetail` + `PodmanContainerDetail` hand-written from upstream OpenAPI; mock seeds match real shape
- [ ] **F-45** (verify): no Phase 3 query reverts `refetchOnWindowFocus: true` default
- [ ] **F-04 + F-38 sequencing**: token-expired path renders error state cleanly (post Phase 1 wiring)
- [ ] **logout flow** (Phase 1 follow-up): TopBar `onLogout` calls `useLogoutMutation` → `POST /api/v1/auth/logout` → `clearAccessToken()` (currently only clears local token; refresh-cookie not revoked server-side)

### WebUI Audit Phase 4 — Layout primitives + antd migration spike (audit 2026-04-27)

> Source: `docs/plans/webui-phase-2-6.md` Phase 4. ADR-051 chose spec → 3-page spike onto antd 6 + pro-components, feature-flagged via `localStorage.getItem('helling.spike') === '1'` for A/B compare.

- [ ] **F-36 spike** (4A): `bun add antd@^6 @ant-design/pro-components @ant-design/charts dayjs` in `web/`; bundle budget 400KB gzipped initial post-Phase 2 lazy-loading
- [ ] **token bridge** (4B): `web/src/theme/tokens.ts` exports typed `ThemeConfig` with `colorPrimary: var(--bzr-lime)`, `colorBgLayout: var(--bzr-void)`, etc.; `<ConfigProvider>` wraps `<App />` in `web/src/main.tsx`
- [ ] **F-06 + F-23 + F-25 + F-26 (spike)**: Instances list → ProTable, Instance Detail → ProDescriptions + Tabs, New Instance wizard → StepsForm; spike hidden behind `localStorage.helling.spike` flag
- [ ] **F-20** (4D): light mode resolves via `ConfigProvider` `algorithm: theme.defaultAlgorithm` swap; `mark.png` light variant added; vestigial `body.light-mode` CSS removed Phase 6
- [ ] **F-11** (4E): TanStack Router replaces `setPage('instance:vm-1')` string-route hack; URLs `/instances`, `/instances/:name`, `/instances/:name/console` etc.; bookmarkable + refresh-safe
- [ ] **F-21** (4F): 87 hardcoded `rgba(255,255,255,0.x)` literals across `web/src/pages/*` replaced with `--h-tint-hover`/`--h-tint-pressed`/`--h-tint-selected`/`--h-divider-soft` tokens added to `:root`
- [ ] **F-24** (4F): `--h-success`/`--h-info`/`--h-warn`/`--h-danger` map to `var(--bzr-success)` etc. in `:root`
- [ ] **F-44** (verify): `prefers-reduced-motion` + `prefers-color-scheme` first-load read still in place after ConfigProvider wrap
- [ ] **F-50** (verify): density localStorage persistence still works after ConfigProvider density-config integration
- [ ] **F-23** (verify): page-header drift gone after ProLayout PageHeader port

### WebUI Audit Phase 5 — Operator polish + a11y rollout (audit 2026-04-27)

> Source: `docs/plans/webui-phase-2-6.md` Phase 5. Closes the operator-ergonomics findings + the a11y leftovers from Phase 1.

- [ ] **F-10** (nav · 5E): collapsible sidebar section headers persisted to user settings; real `Recent` from route history (last 8); pin/unpin from any row, persisted server-side
- [ ] **F-14** (wizard): F-36 → spec → StepsForm absorbs validation + preview natively (lands with Phase 4 spike port for `New Instance`)
- [ ] **F-16** (bulk · 5C): bulk-action menu (Snapshot, Backup, Restart, Delete, Migrate-to); when >3 items, single tracked task in drawer aggregates child progress
- [ ] **F-17** (keyboard · 5F): standard kit `1`–`9` jumps to tab N, `[`/`]` previous/next tab, `Esc` cancel/close, `Enter` primary action; single keyboard overlay (`?`)
- [ ] **F-18** (feedback · 5G): persist last 20 toasts behind bell, alongside alerts, in a "history" tab
- [ ] **F-19** (feedback honesty): `notImplemented('feature-name')` helper produces "Not yet wired" warning toast + visual disabled state on stub buttons
- [ ] **F-25** (a11y · modals): focus trap + `role="dialog"` + `aria-modal` + `aria-labelledby` + focus restore on close (subsumed by Phase 4 antd Modal port for migrated pages; explicit fix for any non-ported modals)
- [ ] **F-26** (a11y · tables): `role="table"`/`role="row"`/`role="cell"` + `aria-sort` on hand-rolled tables until Phase 4 ports them to ProTable (subsumed for ported pages)
- [ ] **F-27** (a11y · color reliance): sidebar dot/icon `aria-label="Running"`/`aria-label="Stopped"`; "off" text or icon variant for stopped rows
- [ ] **F-28** (a11y · toasts): `aria-live="polite"` + `role="status"` on toast region; danger toasts → `role="alert"`; action toasts TTL ≥ 8s
- [ ] **F-32** (ops · 5B): dashboard greeter replaced with digest stripe consuming audit + tasks + alerts query keys; click any segment to filter destination
- [ ] **F-33** (ops · 5A): instance list `backupAge` column with severity color; SLA configuration in Schedules; Backups page header "**N out of SLA · M failed last night**"
- [ ] **F-34** (ops · 5D): `bun add react-diff-viewer-continued`; use in snapshot detail (vs current), cloud-init pre-apply (vs last applied), firewall rule preview
- [ ] **F-35** (brand): topbar crumbs root separator `/` → `✦`; bullet in empty-state copy `✦`

### WebUI Audit Phase 6 — Hardening + leftover decisions (audit 2026-04-27)

> Source: `docs/plans/webui-phase-2-6.md` Phase 6. Last mile: supply-chain + security hygiene + IA cleanup + TS migration completion.

- [ ] **F-09** (arch · full TS): all `web/src/*.jsx` converted to `.tsx`; `find web/src -name '*.jsx' | wc -l` → 0
- [ ] **F-12** (ia): Firewall + FirewallEditor consolidated into one page with list + drawer (Portainer model), or editor becomes a modal
- [ ] **F-13** (ia): Marketplace + Templates renamed ("Templates" → "VM Images", "Marketplace" → "App Catalog") or merged into single Catalog with Type filter
- [ ] **F-22** (responsive policy): documented decision in `web/README.md` — "1440 desktop minimum" gate stays, OR add 1–2 breakpoints (sidebar collapse < 1280, console sidebar drop < 1180)
- [ ] **F-46** (security · CSP): `<meta http-equiv="Content-Security-Policy">` in `web/index.html` as belt-and-braces alongside Caddy headers; `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'`; tighten `style-src` once F-06 reduces inline styles
- [ ] **F-46** (security · SRI policy): documented in `CONTRIBUTING.md`; required for any future CDN-loaded asset
- [ ] **F-47** (verify): global `ResizeObserver` warning suppression still absent from `web/index.html` post-Phase 3 (F-03/F-42 may make the warning naturally disappear)
- [ ] **F-48** (perf · HMR): `setInterval` calls in `web/src/shell.jsx` (or its post-Phase-2 location) wrapped in `import.meta.hot?.accept(() => clearInterval(handle))`; no interval leak after 30 minutes of HMR
- [ ] **F-49** (supply chain): `.github/renovate.json` (or extend `dependabot.yml`) groups bumps by category (react, query, build, antd); weekly minor/patch; majors gated on manual review
- [ ] **F-51** (verify): direct lucide imports outside the `I` wrapper still absent; icon barrel single source of truth
- [ ] **`bun run check`** zero errors (Phase 1 had ~84 pre-existing biome errors; this is the cleanup gate after Phase 5 a11y work lands)

### WebUI Audit Phase 0 — Stack Decision (locked)

- [x] ADR-051 written and accepted: WebUI commits to antd 6 + pro-components per `docs/spec/webui-spec.md`
- [x] Audit captured in `docs/audits/webui-2026-04-27.md`
- [x] Plan persisted at `docs/plans/webui-phase-2-6.md`
- [ ] No new pages added on hand-rolled stack after 2026-04-27

---

## v0.1.0-beta Gate

### Console

- [ ] SPICE VGA console opens for a running VM
- [ ] Serial console opens for a running CT
- [ ] Exec terminal works for Podman container
- [ ] WebSocket proxy handles upgrade correctly

### Dashboard

- [ ] All list pages have full columns, sort, filter, pagination
- [ ] Instance detail: 8 functional tabs
- [ ] Container detail: 6 functional tabs
- [ ] Resource tree shows live status
- [ ] App template gallery deploys a stack successfully

### Proxy Features

- [ ] Auto-snapshot created before DELETE instance

### Database

- [ ] goose migrations apply cleanly (14 tables from docs/spec/sqlite-schema.md, verified via `task migrate` + `sqlite3 .tables`)
- [ ] sqlc generation matches schema
- [ ] Schema upgrade works without data loss

---

## v0.2.0 Gate

- [ ] `helling schedule create` writes systemd timer + service unit
- [ ] `systemctl list-timers | grep helling` shows active timers
- [ ] Timer fires and creates Incus backup
- [ ] Webhook delivers on instance.created with valid HMAC signature
- [ ] Host firewall rule blocks traffic (verified with `nft list ruleset`)
- [ ] Non-admin user restricted to assigned Incus project
- [ ] API docs page renders in dashboard

---

## v0.3.0 Gate

- [ ] Warning banner appears when storage pool >85% full
- [ ] Prometheus scrapes `/metrics` successfully
- [ ] Notification test send reaches Discord/Slack/email
- [ ] Webhook retries on failure (3x with backoff)

---

## v0.4.0 Gate

- [ ] K8s cluster created via k3s cloud-init (VMs provisioned, K8s running)
- [ ] `helling k8s kubeconfig <name>` returns valid kubeconfig
- [ ] BMC power on/off works via bmclib
- [ ] BMC sensor data displayed in dashboard

---

## v0.5.0 Gate

- [ ] LDAP user logs in successfully
- [ ] OIDC login redirects to provider and returns
- [ ] WebAuthn passkey registers and authenticates
- [ ] Incus project quota blocks over-limit instance creation
- [ ] Secret stored and retrieved (encrypted at rest)

---

## v0.8.0 Gate

- [ ] Schemathesis finds zero contract violations
- [ ] goss validates running Helling node passes all checks
- [ ] API p95 latency <200ms under load
- [ ] 24h soak test: no memory growth
- [ ] govulncheck: zero findings
- [ ] nilaway: zero findings

---

## v1.0.0 Gate

### Packaging

- [ ] `dpkg -i helling_*.deb` installs successfully on Debian 13
- [ ] `systemctl status hellingd caddy` shows active
- [ ] `man helling` displays man page
- [ ] `helling completion bash | head` generates completions
- [ ] ISO boots in VM, installs, dashboard accessible at :8006

### Supply Chain

- [ ] `cosign verify-blob --signature sig bin/hellingd` verifies
- [ ] SLSA provenance attached to release
- [ ] SBOM (CycloneDX) attached to release
- [ ] `go-licenses check ./...` — no AGPL-incompatible deps
- [ ] `license-checker --production` — no incompatible npm deps

### Quality

- [ ] gitleaks: zero secrets in repo
- [ ] OpenSSF Best Practices badge: passing
- [ ] CHANGELOG.md auto-generated, accurate
- [ ] All documentation pages exist and are current

---

## Release Gate Summary

| Version      | Items   |
| ------------ | ------- |
| v0.1.0-alpha | ~30     |
| v0.1.0-beta  | ~12     |
| v0.2.0       | ~7      |
| v0.3.0       | ~4      |
| v0.4.0       | ~4      |
| v0.5.0       | ~5      |
| v0.8.0       | ~6      |
| v1.0.0       | ~12     |
| **Total**    | **~80** |

Down from ~147 items in the previous checklist. Fewer items because the proxy eliminates per-endpoint verification — if the proxy works, all upstream endpoints work.
