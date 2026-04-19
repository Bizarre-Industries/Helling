# Helling Implementation Plan v4

> Proxy-first architecture (ADR-014). ISO-only deployment (ADR-021). Six Go dependencies.
>
> **📖 For detailed implementation steps, see:** [implementation-guide.md](./implementation-guide.md)
>
> **📋 For verification checklists, see:** [checklist.md](./checklist.md)
>
> See also: ADR-014 through ADR-021, full-automation-pipeline.md, tools-and-frameworks.md

---

## Architecture Decisions (ADRs 001-025)

| #   | Decision                                               | Replaces                                                 |
| --- | ------------------------------------------------------ | -------------------------------------------------------- |
| 001 | Incus over libvirt                                     | Custom QEMU/KVM/LXC management                           |
| 003 | React + Ant Design + refine over SvelteKit             | Manual CRUD boilerplate                                  |
| 010 | SPICE client over noVNC for VM VGA console             | Wrong protocol                                           |
| 011 | Proxy to Podman socket, no Go bindings                 | API version mismatch, `containers/podman/v5` dep         |
| 012 | Incus Network ACLs for VM/CT firewalling               | Custom nftables code                                     |
| 013 | Incus project limits for VM/CT quotas                  | Custom quota enforcement                                 |
| 014 | Authenticated reverse proxy over handler-per-endpoint  | ~150 handlers wrapping upstream APIs                     |
| 015 | Native upstream response formats for proxied endpoints | Re-enveloping every response                             |
| 016 | Helling CLI for Helling features only                  | ~392 CLI commands wrapping upstream                      |
| 017 | systemd timers over in-process cron (gocron)           | Custom Go scheduler                                      |
| 018 | Shell out over Go libraries for host ops               | google/nftables, go-systemd                              |
| 019 | systemd journal over SQLite for audit                  | Custom audit tables + files                              |
| 020 | Incus config keys over SQLite for tags                 | Custom tag tables + sync                                 |
| 021 | ISO-only installation                                  | Docker try-it mode + manual Debian install               |
| 022 | No CAPMVM / Flintlock                                  | K8s nodes = Incus VMs (CAPN), no microVM K8s             |
| 023 | No custom image format                                 | Native formats per runtime; app templates = compose YAML |
| 024 | Incus native auth in v0.5+                             | JWT + project injection in v0.1–v0.4                     |
| 025 | GitHub Releases as APT source                          | No aptly/reprepro infra; nfpm .deb on Release assets     |

---

## Automation & Tooling Index

All automation surfaces, with version assignments. See docs/design/full-automation-pipeline.md for details on each.

| #   | Tool                                | What It Automates                                   | Version      |
| --- | ----------------------------------- | --------------------------------------------------- | ------------ |
| 1   | oapi-codegen (strict-server)        | Go server interface + router from ~60-endpoint spec | v0.1.0-alpha |
| 2   | oapi-codegen (client)               | Go typed HTTP client for CLI                        | v0.1.0-alpha |
| 3   | orval                               | TS React Query hooks + types from Helling spec      | v0.1.0-alpha |
| 4   | Makefile generate + check-generated | Pipeline glue, CI gate                              | v0.1.0-alpha |
| 5   | vacuum / @redocly/cli               | OpenAPI spec linting                                | v0.1.0-alpha |
| 6   | nilaway + exhaustive                | Go static analysis                                  | v0.1.0-alpha |
| 7   | git-cliff                           | CHANGELOG from conventional commits                 | v0.1.0-alpha |
| 8   | .devcontainer                       | Reproducible dev environment                        | v0.1.0-alpha |
| 9   | Extended pre-commit hooks           | Spec lint, staleness, SPDX, tidy, commit lint       | v0.1.0-alpha |
| 10  | Atlas + GORM provider               | Database migrations (replaces AutoMigrate)          | v0.1.0-beta  |
| 11  | invopop/jsonschema                  | Config JSON Schema from Go struct                   | v0.1.0-beta  |
| 12  | cupaloy                             | API response snapshot testing                       | v0.1.0-beta  |
| 13  | SPICE console (spice-js)            | VM VGA console in browser                           | v0.1.0-beta  |
| 14  | Cloud-init templates                | Validated preseed templates per distro              | v0.1.0-beta  |
| 15  | tygo                                | Go → TypeScript event type generation               | v0.2.0       |
| 16  | Scalar / Redoc                      | Embedded API docs in dashboard                      | v0.2.0       |
| 17  | Prometheus promauto                 | Structured metric definitions                       | v0.3.0       |
| 18  | grafonnet                           | Grafana dashboard JSON from metrics                 | v0.3.0       |
| 19  | Schemathesis                        | API fuzzing from spec in CI                         | v0.8.0       |
| 20  | goss + packer                       | VM-level system validation                          | v0.8.0       |
| 21  | Cobra doc generation                | Man pages + markdown CLI reference                  | v1.0.0       |
| 22  | nfpm                                | .deb packaging                                      | v1.0.0       |
| 23  | live-build / mkosi                  | Bootable ISO image                                  | v1.0.0       |
| 24  | nfpm + GitHub Releases              | .deb on Release assets, ISO-configured APT source   | v1.0.0       |
| 25  | GoReleaser                          | Release pipeline                                    | v1.0.0       |
| 26  | Cosign + SLSA                       | Artifact signing + provenance                       | v1.0.0       |
| 27  | Syft                                | SBOM (CycloneDX + SPDX)                             | v1.0.0       |
| 28  | go-licenses + license-checker       | License compliance CI gate                          | v1.0.0       |

---

## Version Gates

### v0.1.0-alpha — Proxy Works, Dashboard Loads Real Data

**Gate:** Boot ISO → setup wizard → dashboard shows real Incus instances and Podman containers.

#### Backend

- [ ] Proxy middleware: `/api/incus/*` → Incus socket, `/api/podman/*` → Podman socket
- [ ] JWT validation on proxy requests
- [ ] RBAC: user → Incus project mapping via `?project=` param
- [ ] Audit logging to systemd journal (ADR-019)
- [ ] Auth handlers: setup, login (PAM), refresh, logout, TOTP, API tokens
- [ ] User handlers: CRUD (PAM useradd/userdel)
- [ ] System handlers: info, hardware, config, diagnostics
- [ ] Health endpoint
- [ ] SSE events endpoint (aggregates Incus events)
- [ ] OpenAPI spec: ~40 Helling endpoints with envelopes, pagination, error schemas
- [ ] oapi-codegen strict-server generation from spec
- [ ] Delete legacy: manual router, all handlers\_\*.go, strict_handlers.go, response.go
- [ ] Delete Docker mode: Dockerfile, devauth.go, entrypoint.sh
- [ ] Remove unused Go deps (podman bindings, google/nftables, gocron)

#### Frontend

- [ ] Three API clients: hellingClient, incusClient, podmanClient
- [ ] orval hooks for Helling API
- [ ] Dashboard page: system stats from Incus proxy
- [ ] Instances page: list from Incus proxy
- [ ] Containers page: list from Podman proxy
- [ ] Storage page: pool list from Incus proxy
- [ ] Networking page: network list from Incus proxy
- [ ] Images page: from both proxies
- [ ] Delete: VncConsole.tsx, novnc.d.ts, queries.ts, types.ts, PlaceholderPage.tsx

#### CLI

- [ ] Delete all commands that duplicate incus/podman/kubectl
- [ ] Keep: auth, user, system, version, completion
- [ ] Generated client from Helling spec

#### Automation

- [ ] `make generate` + `make check-generated` working
- [ ] vacuum spec linting in CI
- [ ] nilaway + exhaustive in golangci-lint config
- [ ] git-cliff changelog setup
- [ ] .devcontainer/devcontainer.json
- [ ] Extended pre-commit hooks

---

### v0.1.0-beta — Core Dashboard Feature-Complete

**Gate:** Create VM → SPICE console → exec into CT → browse storage pools → see network topology.

#### Backend

- [ ] WebSocket proxy for SPICE/serial/exec (upgrade forwarding)
- [ ] Auto-snapshot before destructive operations (proxy hook)
- [ ] VM screenshot thumbnails (capture + cache)
- [ ] Atlas database migrations (replace AutoMigrate)

#### Frontend

- [ ] SPICE console component (ADR-010)
- [ ] Serial console via xterm.js
- [ ] Exec terminal via xterm.js
- [ ] Instance detail page: 8 tabs
- [ ] Container detail page: 6 tabs
- [ ] Resource tree sidebar with live status
- [ ] Full columns/actions on all list pages
- [ ] App template gallery + one-click deploy

#### Automation

- [ ] Atlas + GORM provider configured
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

- [ ] K8s handlers: CAPN create/delete/scale/upgrade/kubeconfig
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
- [ ] Secrets management (AES-256-GCM encrypted KV)
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
- [ ] GitHub Releases APT source (ADR-025): ISO configures apt source at install time

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
