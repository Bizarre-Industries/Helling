# Helling Full Automation Pipeline

**Date:** 2026-04-15
**Principle:** Define things once, generate everything else. Helling is an OS — the automation surface spans code generation, packaging, ISO building, system testing, and release signing, not just the API layer.

---

## The Core: Proxy + Small Spec

```text
                    api/openapi.yaml (~40 endpoints)
                              │
           ┌──────────────────┼──────────────────┐
           │                  │                  │
                oapi-codegen       oapi-codegen     hey-api/openapi-ts
                   (strict-server)      (client)         (react-query)
           │                  │                  │
      ~25 Go handlers    ~15 CLI commands    TS hooks
           │                  │                  │
      ┌────┴────┐       helling CLI       hellingClient
      │         │                               +
   Helling   Proxy ─────────────────→ incusClient
   handlers  middleware               podmanClient
                │
          Unix sockets
       (Incus + Podman)
```

The proxy middleware (~300 lines) replaces ~150 handlers. One spec change → `make generate` → backend + CLI + frontend Helling hooks updated. Incus/Podman features work automatically through the proxy with zero Helling changes.

---

## All Automation Surfaces

Organized by stack layer. Each entry: what the tool does, what it replaces, and which version it ships in.

### Layer 1: OpenAPI Spec → Go Server

**Tool:** `oapi-codegen` in strict-server mode

Generates typed Go interface + router from the ~40-endpoint Helling spec. We implement business logic only. No manual route registration, JSON parsing, or response encoding.

**Version:** v0.1.0-alpha | **Priority:** Must-have

### Layer 2: OpenAPI Spec → Go CLI Client

**Tool:** `oapi-codegen` in client mode

Generates typed HTTP client methods for every Helling endpoint. Each Cobra command is a ~15-line wrapper.

**Version:** v0.1.0-alpha | **Priority:** Must-have

### Layer 3: OpenAPI Spec → Frontend Hooks

**Tool:** `hey-api/openapi-ts` with react-query client

Generates TypeScript types + React Query hooks from the Helling spec. Custom fetcher wraps the shared fetch client with JWT injection and request ID propagation.

**Version:** v0.1.0-alpha | **Priority:** Must-have

### Layer 4: Pipeline Glue

**Tool:** `Makefile` targets (`generate`, `check-generated`)

`make generate` runs all three generators. `make check-generated` in CI fails if generated code doesn't match spec.

**Version:** v0.1.0-alpha | **Priority:** Must-have

### Layer 5: OpenAPI Spec Linting

**Tool:** `vacuum` (pb33f) or `@redocly/cli`

Lints openapi.yaml for errors, missing pagination params, envelope compliance, and best practices. Runs in pre-commit hook and CI.

**Version:** v0.1.0-alpha | **Priority:** Must-have

### Layer 6: Database Migrations

**Tool:** `goose` + `sqlc`

Migration files are versioned SQL managed by goose. Query files are SQL-first and generate typed data access code via sqlc. This keeps schema/query behavior explicit, reviewable, and aligned with ADR-038.

**Version:** v0.1.0-beta | **Priority:** Must-have

### Layer 7: Config Schema Generation

**Tool:** `invopop/jsonschema`

Generates JSON Schema from Go config struct. Outputs:

1. Startup validation with proper error messages
2. Settings page form generation (antd ProForm from JSON Schema)
3. Config documentation
4. IDE autocompletion for helling.yaml (VS Code YAML extension)

One struct → four outputs.

**Version:** v0.1.0-beta | **Priority:** High

### Layer 8: Go Static Analysis (Extended)

**Tools:** `nilaway` (Uber) + `exhaustive` linter

- nilaway: detects nil pointer dereferences statically
- exhaustive: catches missing enum cases in switch statements

Added to golangci-lint config.

**Version:** v0.1.0-alpha | **Priority:** High

### Layer 9: Changelog Generation

**Tool:** `git-cliff`

Generates CHANGELOG.md from conventional commits. Replaces hand-written changelog. Runs on every release tag.

**Version:** v0.1.0-alpha | **Priority:** High

### Layer 10: Dev Environment

**Tool:** `.devcontainer/devcontainer.json` or `flake.nix`

Reproducible dev environment. Includes: Go, Node, oapi-codegen, golangci-lint, Lima (for running a Helling test VM). Clone → open → everything works.

**Version:** v0.1.0-alpha | **Priority:** High

### Layer 11: Pre-commit Hooks (Extended)

**Tools:** vacuum, check-generated, SPDX header check, `go mod tidy` check, conventional commit lint

Extends existing .githooks/ beyond formatting. Catches spec drift, stale generated code, and unconventional commits before they reach CI.

**Version:** v0.1.0-alpha | **Priority:** High

### Layer 12: API Response Snapshot Testing

**Tool:** `cupaloy` or `go-testdeep`

Snapshot-tests Helling API response envelopes. Catches format regressions (missing `{data}` wrapper, wrong error format) automatically.

**Version:** v0.1.0-beta | **Priority:** Medium

### Layer 13: Event Schema Registry

**Tool:** `tygo` (Go → TypeScript type generator)

Generates TypeScript interfaces from Go event structs. Shared schema between Go event emission, SSE stream, webhook payloads, and frontend event handlers. No manual type synchronization.

**Version:** v0.2.0 | **Priority:** High

### Layer 14: Embedded API Documentation

**Tool:** `Scalar` or `Redoc`

Renders interactive API docs from the embedded OpenAPI spec. Dashboard routes:

- `/api/docs` → Scalar (interactive, try-it-out)
- `/api/reference` → Redoc (clean reference)

Also links to Incus and Podman API docs for proxied endpoints.

**Version:** v0.2.0 | **Priority:** High

### Layer 15: Cloud-Init Template Library

Templates in `/var/lib/helling/templates/cloud-init/` (Ubuntu, Debian, Fedora, Alpine). Validated at build time against cloud-init schema. Used in VM create wizard.

**Version:** v0.1.0-beta | **Priority:** Medium

### Layer 16: SPICE Browser Console

**Tool:** `spice-html5` class browser client

VM VGA console in the browser (ADR-010). Dynamic import to avoid loading console libraries on non-console pages.

**Version:** v0.1.0-beta | **Priority:** Must-have

### Layer 17: Prometheus Metrics

**Tool:** `prometheus/client_golang` with `promauto`

Structured metric definitions with consistent naming. Helling-specific metrics (API latency, auth events, proxy throughput) + proxied Incus `/1.0/metrics`.

**Version:** v0.3.0 | **Priority:** Must-have

### Layer 18: Grafana Dashboard Generation

**Tool:** `grafonnet` (Jsonnet library)

Generates Grafana dashboard JSON from metric definitions. Ships as a downloadable JSON in the Helling docs or as a one-click import in the dashboard settings.

**Version:** v0.3.0 | **Priority:** Medium

### Layer 19: Mock Server (Optional)

**Tool:** `Prism` (Stoplight)

Starts a mock server from the Helling OpenAPI spec. Useful for frontend development without running hellingd. Not critical for a solo dev.

**Version:** any | **Priority:** Nice-to-have

### Layer 20: CLI Man Pages + Markdown Docs

**Tool:** Cobra `doc.GenManTree()` + `doc.GenMarkdownTree()`

Auto-generates man pages (`helling(1)`) and CLI reference markdown from the Cobra command tree. Man pages installed via .deb package.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 21: Shell Completions in .deb

**Tool:** Cobra built-in + nfpm postinst

Cobra generates bash/zsh/fish completions. The .deb postinst script installs them to `/etc/bash_completion.d/`.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 22: Debian Packaging

**Tool:** `nfpm`

Generates .deb packages from a YAML config. Contents: binaries, systemd units, config files, man pages, shell completions, AppArmor profiles. postinst: create helling user, enable services, install completions.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 23: ISO Building

**Tool:** `live-build` (Debian) or `mkosi` (systemd)

Builds a bootable ISO from: Debian 13 base + Incus packages (Zabbly repo) + Podman + Helling .deb packages + preseed answers. `make iso` produces `helling-VERSION-amd64.iso`.

The preseed automates the 3 setup questions (hostname, disk, admin password) and the rest is unattended.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 24: APT Repository

**Tool:** `reprepro` (ADR-045)

Serves .deb packages for `helling system upgrade`. GPG-signed packages. Hosted on GitHub Pages, S3, or self-hosted. The upgrade command checks this repo.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 25: Release Pipeline

**Tool:** `GoReleaser`

Produces binaries (amd64 + arm64), checksums, .deb packages (via nfpm), changelogs (via git-cliff). Triggered by git tags. Integrates with Cosign + Syft.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 26: Artifact Signing

**Tool:** `Cosign` + SLSA provenance

Signs all release artifacts: binaries, .deb packages. SLSA provenance generated for supply chain verification.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 27: SBOM

**Tool:** `Syft`

Generates Software Bill of Materials (CycloneDX + SPDX) from Go binaries and web dependencies. Attached to every release.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 28: License Compliance

**Tools:** `go-licenses` + `license-checker`

Scans Go and npm dependencies for AGPL-3.0 compatibility. CI gate: block incompatible licenses.

**Version:** v1.0.0 | **Priority:** Must-have

### Layer 29: API Fuzzing

**Tool:** `Schemathesis`

Property-based fuzzing from the Helling OpenAPI spec. Runs in CI against a Lima VM with Helling installed. Catches contract violations, 500s, and edge cases.

**Version:** v0.8.0 | **Priority:** High

### Layer 30: System Validation

**Tool:** `goss` + `packer`

packer builds a VM from the ISO. goss validates the running system: services running, ports listening, sockets present, packages installed. Catches "builds but doesn't boot" bugs.

```yaml
# goss.yaml
service:
  hellingd:
    enabled: true
    running: true
  caddy:
    enabled: true
    running: true
  incusd:
    enabled: true
    running: true
file:
  /run/podman/podman.sock:
    exists: true
port:
  tcp:8443:
    listening: true
  tcp:8006:
    listening: true
```

**Version:** v0.8.0 | **Priority:** High

---

## Summary by Version

<!-- markdownlint-disable MD060 -->

| Version      | Automation added                                                                                                                              |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| v0.1.0-alpha | oapi-codegen (server + client), hey-api/openapi-ts, Makefile pipeline, vacuum, nilaway/exhaustive, git-cliff, dev container, pre-commit hooks |
| v0.1.0-beta  | goose/sqlc migration-query workflow, config JSON Schema, snapshot testing, SPICE browser console, cloud-init templates                        |
| v0.2.0       | tygo event types, embedded API docs (Scalar/Redoc)                                                                                            |
| v0.3.0       | Prometheus metrics, Grafana dashboard generation                                                                                              |
| v0.8.0       | Schemathesis fuzzing, goss system validation                                                                                                  |
| v1.0.0       | nfpm .deb, ISO builder, APT repo, GoReleaser, Cosign, SLSA, SBOM, license checks, man pages, completions                                      |

<!-- markdownlint-enable MD060 -->
