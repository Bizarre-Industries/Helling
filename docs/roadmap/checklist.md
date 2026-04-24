# Helling Release Checklist v4

> Proxy architecture (ADR-014). ISO-only (ADR-021). Every item has a verification command.

---

## v0.1.0-alpha Gate

### Proxy

Proxy middleware is wired in hellingd per ADR-014 (`apps/hellingd/internal/proxy/`). Internal CA + per-user cert issuance complete (PRs O-1..O-5); the remaining v0.1-beta gate is per-request TLS Transport selection (PR O-6) and live verification against real Incus + Podman (docs/spec/internal-ca.md).

- [x] Proxy scaffold exists and forwards requests via httputil.ReverseProxy (unit + integration tests in `apps/hellingd/internal/proxy/`)
- [x] WebSocket upgrades pass through to upstream (covered by `TestProxy_WebSocketUpgradePassesThrough` in PR O-4)
- [x] Internal CA bootstrap + per-user cert issuance on userCreate (PRs O-1..O-5; gated behind `HELLING_CA_DIR`)
- [ ] `curl -H "Authorization: Bearer $TOKEN" http://unix:/var/lib/helling/hellingd.sock:/api/incus/1.0/instances | jq '.metadata'` returns Incus instances
- [ ] `curl -H "Authorization: Bearer $TOKEN" http://unix:/var/lib/helling/hellingd.sock:/api/podman/libpod/containers/json | jq '.[0].Names'` returns Podman containers
- [x] Unauthenticated request to proxy returns 401 (covered by `TestProxy_Unauthenticated_Returns401`)
- [ ] Non-admin user sees only their Incus project resources (blocked on per-user TLS Transport selector — PR O-6, v0.1-beta)

### Auth

- [ ] Setup → login → JWT → protected routes → refresh → TOTP → recovery codes: full flow
- [ ] Rate limiting: 6 failed logins → 429
- [ ] API token: create → auth with token → revoke → rejected

### Build

- [ ] `make build` succeeds with zero warnings
- [ ] `make test` passes
- [ ] `make lint` clean (includes nilaway, exhaustive)
- [ ] `make generate` produces all generated files without error
- [ ] `make check-generated` — generated code matches spec
- [ ] `cd web && bun run build` succeeds

### Code Hygiene

- [ ] No `docker/docker` in go.mod
- [ ] No `google/nftables` in go.mod
- [ ] No `go-co-op/gocron` in go.mod
- [ ] No `devauth.go` exists
- [ ] No `Dockerfile` exists in deploy/
- [ ] No `router.go` (manual routes) exists
- [ ] No `handlers_phase*.go` files exist
- [ ] No `strict_handlers.go` (empty struct) exists
- [ ] `grep -rn "TODO\|FIXME\|stub\|not implemented" apps/ --include="*.go" | grep -v _test.go | wc -l` = 0
- [ ] `grep -rn "Docker mode\|Docker try-it\|devauth" docs/ --exclude-dir=decisions --exclude=checklist.md --exclude=plan.md | wc -l` = 0

### Spec

- [ ] `npx @redocly/cli lint api/openapi.yaml` or `vacuum lint api/openapi.yaml` — zero errors
- [ ] Every Helling endpoint has operationId, request/response schemas, error responses
- [ ] Every list endpoint follows cursor pagination contract (`limit`, `cursor`, `meta.page`)

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

- [ ] git-cliff produces CHANGELOG.md from commits
- [ ] .devcontainer/devcontainer.json exists
- [ ] Pre-commit hooks catch stale generated code

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
