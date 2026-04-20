# Helling Implementation Plan v4

> Proxy-first architecture (ADR-014). ISO-only deployment (ADR-021). approximately 10-12 Go dependencies.
>
> **📋 For verification checklists, see:** [checklist.md](./checklist.md)
>
> See also: ADR-014 through ADR-021, full-automation-pipeline.md, tools-and-frameworks.md

---

## Implementation Sequence

Execution order aligned with accepted ADRs and active migration decisions:

1. Freeze hand-authored OpenAPI changes.
2. Implement Huma spike endpoints (`POST /api/v1/auth/login`, `GET /api/v1/users`).
3. Generate and validate `api/openapi.yaml` with vacuum.
4. Expand Huma coverage to all Helling-owned routes.
5. Migrate WebUI API generation to hey-api/openapi-ts.
6. Land tooling and hook package.
7. Complete drift cleanup in docs/standards/spec/design.

Mandatory validation loop for each phase:

```bash
make generate
make check-generated
make fmt-check
make lint
make test
vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml
```

---

## v0.1.0-alpha Prerequisites

Required baseline before alpha implementation is considered ready:

- Initial Go module scaffolding for core binaries
- API contract skeleton for Helling-specific endpoints
- Makefile targets for generation, lint, and test flow
- Web UI scaffold with build and dev commands
- CI workflow skeleton with basic validation gates

Documentation baseline:

- Auth model aligned to ADR-024/030/031/032
- Console model aligned to ADR-010
- Kubernetes model aligned to ADR-005
- MicroVM scope explicitly deferred (ADR-006)

Exit criteria:

- No contradictory references to superseded auth/query-param model
- No contradictory references to any non-SPICE default VM console path
- No contradictory references to CAPN as v0.1 default path
- No contradictory references to microVM runtime in v0.1 scope

---

## Architecture Decisions (ADRs 001-048)

| #   | Decision                                                      | Status                |
| --- | ------------------------------------------------------------- | --------------------- |
| 001 | Incus over libvirt                                            | Accepted              |
| 003 | React + Ant Design over SvelteKit                             | Accepted              |
| 010 | SPICE as in-browser VM console protocol                       | Accepted              |
| 011 | Proxy to Podman socket, no Go bindings                        | Superseded by ADR-035 |
| 012 | Incus Network ACLs for VM/CT firewalling                      | Accepted              |
| 013 | Incus project limits for VM/CT quotas                         | Accepted              |
| 014 | Authenticated reverse proxy over handler-per-endpoint         | Accepted              |
| 015 | Native upstream response formats for proxied endpoints        | Accepted              |
| 016 | Helling CLI for Helling features only                         | Accepted              |
| 017 | systemd timers over in-process cron (gocron)                  | Accepted              |
| 018 | Shell out over Go libraries for host ops                      | Accepted              |
| 019 | systemd journal over SQLite for audit                         | Accepted              |
| 020 | Incus config keys over SQLite for tags                        | Accepted              |
| 021 | ISO-only installation                                         | Accepted              |
| 022 | No CAPMVM / Flintlock                                         | Accepted              |
| 023 | No custom image format                                        | Accepted              |
| 024 | Incus per-user TLS auth from v0.1                             | Accepted              |
| 025 | Signed APT repository via GitHub Pages for updates            | Accepted              |
| 026 | SHA-pin all third-party GitHub Actions                        | Accepted              |
| 027 | Two-daemon split                                              | Accepted              |
| 028 | Unix socket between daemons                                   | Accepted              |
| 029 | Dedicated hellingprox system user                             | Accepted              |
| 030 | Argon2id password hashing                                     | Accepted              |
| 031 | Ed25519 JWT signing                                           | Accepted              |
| 032 | Three fixed roles for v0.1                                    | Accepted              |
| 034 | Lima dev environment                                          | Accepted              |
| 035 | Supersede ADR-011 with proxy-only Podman access               | Accepted              |
| 036 | Incus HTTPS loopback transport for delegated-user proxy calls | Accepted              |
| 037 | Caddy as edge service                                         | Accepted              |
| 038 | SQL-first with sqlc + goose over GORM + Atlas provider        | Accepted              |
| 039 | age over bespoke AES-256-GCM secret envelope                  | Accepted              |
| 040 | net/http ServeMux over chi                                    | Accepted              |
| 041 | URI major versioning for Helling-owned API surfaces           | Accepted              |
| 042 | Security scanning stack consolidation                         | Accepted              |
| 043 | Huma with humago for Helling-owned HTTP layer                 | Accepted              |
| 044 | hey-api/openapi-ts for WebUI code generation                  | Accepted              |
| 045 | APT repository tooling decision                               | Proposed              |
| 046 | ISO build tooling decision                                    | Proposed              |
| 047 | Dark mode scope for v0.1                                      | Proposed              |
| 048 | Mobile scope for v0.1                                         | Proposed              |

---

## Automation & Tooling Index

All automation surfaces, with version assignments. See docs/design/full-automation-pipeline.md for details on each.

| #   | Tool                                | What It Automates                                  | Version      |
| --- | ----------------------------------- | -------------------------------------------------- | ------------ |
| 1   | Huma v2 + humago                    | Go HTTP operations + generated OpenAPI from code   | v0.1.0-alpha |
| 2   | oapi-codegen (client)               | Go typed HTTP client for CLI                       | v0.1.0-alpha |
| 3   | hey-api/openapi-ts                  | TS SDK + fetch client + TanStack Query integration | v0.1.0-alpha |
| 4   | Makefile generate + check-generated | Pipeline glue, CI gate                             | v0.1.0-alpha |
| 5   | vacuum / @redocly/cli               | OpenAPI spec linting                               | v0.1.0-alpha |
| 6   | nilaway + exhaustive                | Go static analysis                                 | v0.1.0-alpha |
| 7   | git-cliff                           | CHANGELOG from conventional commits                | v0.1.0-alpha |
| 8   | .devcontainer                       | Reproducible dev environment                       | v0.1.0-alpha |
| 9   | Extended pre-commit hooks           | Spec lint, staleness, SPDX, tidy, commit lint      | v0.1.0-alpha |
| 10  | sqlc + goose                        | SQL-first query generation and migrations          | v0.1.0-beta  |
| 11  | invopop/jsonschema                  | Config JSON Schema from Go struct                  | v0.1.0-beta  |
| 12  | cupaloy                             | API response snapshot testing                      | v0.1.0-beta  |
| 13  | SPICE browser console               | VM VGA console in browser                          | v0.1.0-beta  |
| 14  | Cloud-init templates                | Validated preseed templates per distro             | v0.1.0-beta  |
| 15  | tygo                                | Go → TypeScript event type generation              | v0.2.0       |
| 16  | Scalar / Redoc                      | Embedded API docs in dashboard                     | v0.2.0       |
| 17  | Prometheus promauto                 | Structured metric definitions                      | v0.3.0       |
| 18  | grafonnet                           | Grafana dashboard JSON from metrics                | v0.3.0       |
| 19  | Schemathesis                        | API fuzzing from spec in CI                        | v0.8.0       |
| 20  | goss + packer                       | VM-level system validation                         | v0.8.0       |
| 21  | Cobra doc generation                | Man pages + markdown CLI reference                 | v1.0.0       |
| 22  | nfpm                                | .deb packaging                                     | v1.0.0       |
| 23  | live-build / mkosi                  | Bootable ISO image                                 | v1.0.0       |
| 24  | nfpm + signed APT repo              | .deb publish + indexed/signed repository updates   | v1.0.0       |
| 25  | GoReleaser                          | Release pipeline                                   | v1.0.0       |
| 26  | Cosign + SLSA                       | Artifact signing + provenance                      | v1.0.0       |
| 27  | Syft                                | SBOM (CycloneDX + SPDX)                            | v1.0.0       |
| 28  | go-licenses + license-checker       | License compliance CI gate                         | v1.0.0       |

---

## Version Gates

### v0.1.0-alpha — Proxy Works, Dashboard Loads Real Data

**Gate:** Boot ISO → setup wizard → dashboard shows real Incus instances and Podman containers.

#### Backend (Beta)

- [ ] Proxy middleware: `/api/incus/*` → Incus HTTPS loopback, `/api/podman/*` → Podman socket
- [ ] JWT validation on proxy requests
- [ ] RBAC: per-user Incus TLS certificate identity enforcement
- [ ] Audit logging to systemd journal (ADR-019)
- [ ] Auth handlers: setup, login (PAM), refresh, logout, TOTP, API tokens
- [ ] User handlers: CRUD (role/status and Incus trust identity lifecycle)
- [ ] System handlers: info, hardware, config, diagnostics
- [ ] Health endpoint
- [ ] SSE events endpoint (aggregates Incus events)
- [ ] OpenAPI spec: ~40 Helling endpoints with envelopes, pagination, error schemas
- [ ] Huma operation registration + generated OpenAPI from code
- [ ] Delete legacy: manual router, all handlers\_\*.go, strict_handlers.go, response.go
- [ ] Delete Docker mode: Dockerfile, devauth.go, entrypoint.sh
- [ ] Remove unused Go deps (podman bindings, google/nftables, gocron)

#### Frontend (Beta)

- [ ] Three API clients: hellingClient, incusClient, podmanClient
- [ ] hey-api/openapi-ts generated SDK/hooks for Helling API
- [ ] Dashboard page: system stats from Incus proxy
- [ ] Instances page: list from Incus proxy
- [ ] Containers page: list from Podman proxy
- [ ] Storage page: pool list from Incus proxy
- [ ] Networking page: network list from Incus proxy
- [ ] Images page: from both proxies
- [ ] Delete stale console/client placeholders that conflict with ADR-010 direction

#### CLI

- [ ] Delete all commands that duplicate incus/podman/kubectl
- [ ] Keep: auth, user, system, version, completion
- [ ] Generated client from Helling spec

#### Automation (Beta)

- [ ] `make generate` + `make check-generated` working
- [ ] vacuum spec linting in CI
- [ ] nilaway + exhaustive in golangci-lint config
- [ ] git-cliff changelog setup
- [ ] .devcontainer/devcontainer.json
- [ ] Extended pre-commit hooks

---

### v0.1.0-beta — Core Dashboard Feature-Complete

**Gate:** Create VM → SPICE browser console → exec into CT → browse storage pools → see network topology.

#### Backend

- [ ] WebSocket proxy for SPICE/serial/exec (upgrade forwarding)
- [ ] Auto-snapshot before destructive operations (proxy hook)
- [ ] VM screenshot thumbnails (capture + cache)
- [ ] SQL-first migrations with goose + generated queries via sqlc

#### Frontend

- [ ] SPICE browser console component (ADR-010)
- [ ] Serial console via xterm.js
- [ ] Exec terminal via xterm.js
- [ ] Instance detail page: 8 tabs
- [ ] Container detail page: 6 tabs
- [ ] Resource tree sidebar with live status
- [ ] Full columns/actions on all list pages
- [ ] App template gallery + one-click deploy

#### Automation

- [ ] sqlc + goose configured
- [ ] Config JSON Schema generation
- [ ] cupaloy response snapshot tests
- [ ] Cloud-init template library

---

### v0.2.0 — Platform Core

**Gate:** Schedule creates Incus backup. Host firewall rule blocks traffic. Non-admin restricted to project.

- [ ] Schedule handlers: CRUD for systemd timers (ADR-017)
- [ ] Webhook handlers: CRUD + HMAC delivery + retry
- [ ] Host firewall handlers: nft CLI (ADR-018)
- [ ] Tags via Incus user.\* config (ADR-020)
- [ ] Embedded API docs (Scalar/Redoc)
- [ ] tygo event type generation (Go → TypeScript)
- [ ] Dashboard: schedules page, webhooks page, firewall page, audit page

---

### v0.3.0 — Observability + Notifications

**Gate:** Webhook fires on instance.created. Warning shows for full disk. Prometheus scrapes /metrics.

- [ ] Warnings engine (goroutine, 5min checks, capacity forecasting)
- [ ] Prometheus /metrics endpoint (Helling + proxied Incus metrics)
- [ ] Notification channels (Discord, Slack, email, Gotify, ntfy)
- [ ] Notification handlers: CRUD + test send
- [ ] Grafana dashboard JSON (grafonnet)
- [ ] Dashboard: warnings banner, notification settings

---

### v0.4.0 — K8s + BMC + Clustering

**Gate:** K8s cluster created, nodes running. BMC powers on server.

- [ ] K8s handlers: k3s cloud-init create/delete/scale/upgrade/kubeconfig
- [ ] BMC handlers: CRUD + power + sensors + SEL (bmclib)
- [ ] Cluster status visible via Incus proxy
- [ ] Dashboard: K8s page with create wizard, BMC page, cluster page

---

### v0.5.0 — Enterprise Auth

**Gate:** LDAP user logs in. WebAuthn passkey registers. Quota blocks over-limit create.

- [ ] WebAuthn passkeys (`go-webauthn/webauthn`)
- [ ] LDAP/AD sync (`go-ldap/ldap/v3`)
- [ ] OIDC (PKCE, group mapping) (`coreos/go-oidc/v3`)
- [ ] Projects + quotas (Incus project limits)
- [ ] Secrets management (age-encrypted KV)
- [ ] Internal CA (root cert generation, issuance, CRL, auto-renewal)
- [ ] Dashboard: LDAP/OIDC/WebAuthn login options, project switcher, quota display

---

### v0.8.0 — Production Hardening

**Gate:** All E2E pass. API p95 <200ms. Zero nil-pointer panics. Zero govulncheck findings.

- [ ] Schemathesis API fuzzing in CI
- [ ] goss + packer VM-level system validation
- [ ] Performance tuning (profiling, connection pooling)
- [ ] Security audit (nilaway results clean, govulncheck clean)
- [ ] Memory leak testing (24h soak test)

---

### v1.0.0 — Ship

**Gate:** ISO boots, installs, dashboard works. All E2E pass. SBOM attached. OpenSSF badge earned.

#### Packaging

- [ ] nfpm: .deb packages (binaries, systemd units, config, AppArmor profile)
- [ ] Man pages (Cobra doc.GenManTree) installed via .deb
- [ ] Shell completions installed via .deb postinst
- [ ] live-build/mkosi: bootable ISO (`make iso`)
- [ ] Signed APT repository (ADR-025): ISO configures apt source and keyring at install time

#### Release

- [ ] GoReleaser: binaries, checksums, .deb, changelog
- [ ] Cosign: sign all artifacts
- [ ] SLSA: build provenance
- [ ] Syft: SBOM (CycloneDX + SPDX) attached to release

#### Quality

- [ ] go-licenses + license-checker: clean
- [ ] govulncheck: clean
- [ ] gitleaks: clean
- [ ] OpenSSF Best Practices badge: passing

#### Documentation

- [ ] installation.md (ISO boot)
- [ ] configuration.md (helling.yaml reference)
- [ ] upgrading.md
- [ ] troubleshooting.md
- [ ] CHANGELOG.md (auto-generated by git-cliff)
- [ ] API docs embedded in dashboard (Scalar)

---

## Post-v1

Cost tracking, maintenance windows, incident tracking, infrastructure blueprints, GPU sharing, VM fault tolerance, Terraform provider, Ansible collection.
