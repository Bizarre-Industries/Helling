# Standards, Specifications & Compliance

Every standard, spec, and best practice that Helling must comply with to be a legitimate cloud-native, Linux-native, open source platform. Organized by domain. Each entry: what the standard is, what it requires, and Helling's current compliance status.

---

## 1. CNCF / Cloud Native Standards

### OCI (Open Container Initiative)
```
What:     Industry standard for container images and runtimes.
Specs:    Image Spec (image format), Runtime Spec (how containers run), Distribution Spec (how images are distributed)
Requires: Container images are OCI-compliant, runtime is OCI-compliant
Helling:  ✓ Podman uses OCI images and runc/crun (OCI-compliant runtimes)
          ✓ Incus uses OCI images for application containers
Action:   None — Podman and Incus handle this natively
```

### CNI (Container Network Interface)
```
What:     Standard plugin interface for container networking
Requires: K8s clusters use CNI-compliant network plugins
Helling:  ✓ K8s clusters deploy with Flannel, Calico, or Cilium (all CNI-compliant)
Action:   Ensure cluster wizard only offers CNI-compliant options
```

### CSI (Container Storage Interface)
```
What:     Standard for exposing storage to containerized workloads
Requires: K8s storage plugins implement CSI
Helling:  ✓ K8s clusters use Longhorn or local-path (both CSI-compliant)
Action:   Document CSI drivers supported. Add CSI driver management in K8s settings.
```

### CRI (Container Runtime Interface)
```
What:     Plugin interface for K8s to use container runtimes
Requires: K8s uses CRI-compliant runtime (containerd, CRI-O)
Helling:  ✓ K3s/kubeadm use containerd (CRI-compliant)
Action:   None — handled by K8s distribution choice
```

### OpenTelemetry (OTel)
```
What:     Vendor-neutral observability framework (traces, metrics, logs)
Requires: Instrument applications with OTel SDK, export via OTLP
Helling:  ✗ NOT IMPLEMENTED
Action:   
  - Add OpenTelemetry Go SDK to hellingd
  - Instrument API handlers with traces (request ID, duration, status)
  - Export traces via OTLP to configurable endpoint (Jaeger, Tempo, etc.)
  - Structured logs should include trace_id and span_id
  - Metrics exposed via Prometheus (already planned) AND OTLP
  - Dashboard: /settings → Telemetry → OTLP endpoint configuration
Priority: Medium (v0.5+). Prometheus /metrics covers most monitoring needs.
          OTel adds distributed tracing for debugging complex operations.
```

### CloudEvents
```
What:     Specification for describing events in a common format
Requires: Events emitted by the system follow CloudEvents schema
Helling:  ✗ NOT IMPLEMENTED — webhook events use custom format
Action:   
  - Webhook payloads SHOULD follow CloudEvents spec v1.0:
    {
      "specversion": "1.0",
      "type": "dev.bizarre.helling.instance.started",
      "source": "/instances/vm-web-1",
      "id": "uuid",
      "time": "2026-04-13T14:22:00Z",
      "datacontenttype": "application/json",
      "data": { "name": "vm-web-1", "status": "Running" }
    }
  - This makes Helling events consumable by any CloudEvents-compatible system
    (Knative, Argo Events, Tekton, etc.)
Priority: Low (post-v1). Custom format works. CloudEvents adds interoperability.
```

### Kubernetes Conformance
```
What:     CNCF certification that a K8s distribution passes conformance tests
Requires: K8s clusters created by Helling pass Sonobuoy conformance suite
Helling:  ⚠ UNTESTED — clusters use K3s/kubeadm which are conformant,
          but Helling's provisioning must not break conformance
Action:   
  - Add Sonobuoy conformance test to integration CI
  - Run after cluster creation in E2E tests
  - Document which K8s versions are tested and conformant
Priority: High (v0.5). Users deploying production K8s need this assurance.
```

---

## 2. Linux / OS Standards

### FHS (Filesystem Hierarchy Standard)
```
What:     Standard directory layout for Unix-like systems
Requires: Config in /etc, variable data in /var, binaries in /usr or /opt
Helling:  ✓ /etc/helling/ (config), /var/lib/helling/ (data), /var/log/helling/ (logs)
          ✓ /opt/helling/web/ (dashboard static files)
Action:   Verify no files are placed outside FHS-compliant locations.
          Man pages in /usr/share/man/. Systemd units in /lib/systemd/system/.
```

### systemd
```
What:     Linux init system and service manager
Requires: Services have proper unit files with security hardening
Helling:  ⚠ PARTIALLY — unit files exist but not hardened
Action:
  hellingd.service:
    [Service]
    Type=notify                          # sd_notify integration
    ExecStart=/usr/bin/hellingd
    Restart=on-failure
    RestartSec=5
    WatchdogSec=30                       # systemd watchdog
    
    # Security hardening:
    ProtectSystem=strict
    ProtectHome=true
    PrivateTmp=true
    NoNewPrivileges=false                # Needs root for Incus/Podman
    ReadWritePaths=/var/lib/helling /var/log/helling /etc/helling
    CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_SYS_ADMIN
    SystemCallFilter=@system-service @network-io @file-system
    MemoryDenyWriteExecute=true
    
    # Resource limits:
    LimitNOFILE=65535
    LimitNPROC=4096
    
    # Logging:
    StandardOutput=journal
    StandardError=journal
    SyslogIdentifier=hellingd
```

### AppArmor
```
What:     Linux security module for mandatory access control
Requires: Confined profiles for system services
Helling:  ✗ NOT IMPLEMENTED — hellingd runs unconfined
Action:   
  - Create AppArmor profile: /etc/apparmor.d/usr.bin.hellingd
  - Allow: read /etc/helling/*, rw /var/lib/helling/*, rw /var/log/helling/*
  - Allow: Unix socket communication with Incus and Podman
  - Deny: everything else
  - Ship profile in .deb package, load on install
Priority: Medium (v0.5). Important for security-conscious users.
```

### Journal / Syslog (ADR-019)
```
What:     Standard system logging
Requires: Structured log output compatible with journald
Helling:  ✓ IMPLEMENTED via ADR-019 — systemd journal is the primary audit + log store
Action:
  - hellingd logs to stdout → captured by systemd journal automatically
  - Audit events written as structured journal entries with custom fields
  - Journal fields: PRIORITY, SYSLOG_IDENTIFIER=hellingd, plus custom:
    HELLING_USER, HELLING_ACTION, HELLING_RESOURCE, HELLING_REQUEST_ID
  - Query audit log: journalctl -u hellingd HELLING_ACTION=instance.create
  - Syslog forwarding: configurable RFC 5424 output to external syslog server
  - No SQLite audit tables — journal is the single source of audit truth
  - Journal is append-only and tamper-evident when configured with FSS (Forward Secure Sealing)
```

---

## 3. Security Standards

### OpenSSF Best Practices Badge
```
What:     Self-certification that project follows open source security best practices
Requires: 67 criteria across: basics, change control, reporting, quality, security, analysis
Helling:  ✗ NOT APPLIED — need to register at bestpractices.coreinfrastructure.org
Action:   Apply for Passing badge. Key requirements:
  MUST:
  - [x] FLOSS license (AGPL-3.0) ✓
  - [ ] Documentation of how to contribute (CONTRIBUTING.md)
  - [ ] Working build system (make build) ✓
  - [ ] Automated test suite (make test) ✓
  - [ ] SECURITY.md with vulnerability reporting process
  - [ ] Fix critical vulnerabilities within 60 days
  - [ ] Unique version numbering (SemVer) ✓
  - [ ] Cryptographic verification of releases (Cosign planned)
  - [ ] Static analysis tool in CI (golangci-lint + gosec) ✓
  
  SHOULD:
  - [ ] Code coverage >80%
  - [ ] Signed releases
  - [ ] Hardened development environment
Priority: High (v0.1). Apply immediately. Many criteria already met.
```

### SLSA (Supply-chain Levels for Software Artifacts)
```
What:     Framework for ensuring integrity of software artifacts
Levels:   L1 (documented build), L2 (hosted build), L3 (hardened build)
Helling:  ✗ NOT IMPLEMENTED
Action:
  SLSA L1 (v0.1):
  - [ ] Build process documented (Makefile, nfpm config)
  - [ ] Provenance generated (who built what, when, from which source)
  
  SLSA L2 (v0.5):
  - [ ] Build service is hosted (GitHub Actions)
  - [ ] Provenance generated by build service (not developer)
  - [ ] Provenance signed (Sigstore/Cosign)
  
  SLSA L3 (v1.0):
  - [ ] Build environment is hardened (ephemeral, isolated)
  - [ ] Build is non-falsifiable (tamper-proof provenance)
  - [ ] Use slsa-github-generator for SLSA L3 provenance
```

### SBOM (Software Bill of Materials)
```
What:     Machine-readable inventory of all software components
Formats:  SPDX (ISO/IEC 5962) and CycloneDX (ECMA-424)
Requires: Generate SBOM for every release artifact
Helling:  ⚠ PLANNED — Syft in release pipeline
Action:
  - [ ] Generate CycloneDX SBOM with Syft for Go binaries
  - [ ] Generate CycloneDX SBOM with Syft for container images
  - [ ] Generate SPDX SBOM as alternative format
  - [ ] Attach SBOMs to GitHub releases
  - [ ] Embed SBOM in container image labels (OCI annotation)
  - [ ] VEX (Vulnerability Exploitability eXchange) for false positive management
  - [ ] License compliance: go-licenses or scancode-toolkit in CI
  - [ ] Block dependencies with AGPL-incompatible licenses
  - [ ] Generate NOTICE file from dependency license scan
```

### OWASP API Security Top 10 (2023)
```
What:     Top 10 API security risks
Helling compliance:
  API1 Broken Object Level Auth:      [ ] Need per-resource permission checks
  API2 Broken Authentication:         [x] JWT + TOTP + WebAuthn + rate limiting
  API3 Broken Object Property Auth:   [ ] Need field-level access control
  API4 Unrestricted Resource Consume: [ ] Need request size limits, pagination limits
  API5 Broken Function Level Auth:    [ ] Need role checks on every handler
  API6 Unrestricted Access to Flows:  [ ] Need rate limiting on expensive operations
  API7 Server Side Request Forgery:   [ ] Need URL validation for webhook targets
  API8 Security Misconfiguration:     [ ] Need secure defaults, disable debug in prod
  API9 Improper Inventory Management: [x] OpenAPI spec covers all endpoints
  API10 Unsafe API Consumption:       [ ] Need input validation for all external data
```

### CIS Benchmarks
```
What:     Configuration security benchmarks for OS, containers, K8s
Helling:
  - [ ] CIS Debian 13 Benchmark: hardened OS defaults in installer
  - [ ] CIS Podman Benchmark: secure container defaults
  - [ ] CIS Kubernetes Benchmark: kube-bench in K8s cluster setup
  - [ ] Dashboard: /security → CIS Scan → run benchmark, show results, remediation
Priority: Post-v1. Important for compliance-focused users.
```

---

## 4. API Standards

### OpenAPI 3.1
```
What:     Standard for describing REST APIs
Requires: Machine-readable API description in OpenAPI format
Helling:  ✓ api/openapi.yaml exists, oapi-codegen generates types
Action:
  - [ ] Upgrade to OpenAPI 3.1 (from 3.0) for JSON Schema compatibility
  - [ ] Serve spec at GET /api/v1/openapi.yaml
  - [ ] Swagger UI at GET /api/docs
  - [ ] Redoc at GET /api/reference
  - [ ] Validate all requests against OpenAPI schema (middleware)
  - [ ] Validate all responses against OpenAPI schema (test)
  - [ ] Generate changelogs from OpenAPI diffs (oasdiff)
```

### REST Maturity (Richardson Level 3)
```
What:     REST API maturity model
Levels:   L0 (HTTP), L1 (resources), L2 (HTTP verbs), L3 (HATEOAS)
Helling:  L2 (resources + HTTP verbs). No HATEOAS.
Action:
  - L2 is sufficient for infrastructure APIs. Proxmox, VMware, DO are all L2.
  - HATEOAS adds complexity without value for CLI/dashboard consumption.
  - Decision: Stay at L2. Document API navigation in OpenAPI spec.
```

### OAuth 2.0 / OIDC
```
What:     Standard for authorization (OAuth) and identity (OIDC)
Requires: Support OIDC for SSO integration
Helling:  ✗ NOT IMPLEMENTED — PAM + JWT only
Action:
  - [ ] OIDC client support (login via external IdP)
  - [ ] OIDC discovery (/.well-known/openid-configuration on IdP)
  - [ ] Token exchange (OIDC token → Helling JWT)
  - [ ] Group claim mapping (OIDC groups → Helling roles)
  - [ ] PKCE flow for dashboard (SPA)
Priority: Medium (v0.5). Required for LDAP/AD/SSO integration.
```

### SCIM (System for Cross-domain Identity Management)
```
What:     Standard for user provisioning from IdP to application
Requires: SCIM 2.0 endpoint for user/group sync
Helling:  ✗ NOT IMPLEMENTED
Action:   Post-v1. LDAP sync covers most use cases.
```

---

## 5. Go Standards

### Project Layout
```
What:     Community conventions for Go project structure
Standard: golang-standards/project-layout (de facto, not official)
Helling:  ⚠ PARTIALLY — uses apps/ instead of cmd/, internal is correct
Action:
  Current:  apps/hellingd/internal/
  Standard: cmd/hellingd/main.go + internal/
  
  Decision: Keep current layout. apps/ works with go.work multi-module.
  The important rule: internal/ prevents external imports. This is correct.
  
  Ensure:
  - [ ] cmd/ (or apps/) contains only main.go with minimal wiring
  - [ ] internal/ contains all business logic
  - [ ] api/ contains OpenAPI spec
  - [ ] web/ contains frontend
  - [ ] docs/ contains documentation
  - [ ] No pkg/ directory (anti-pattern for internal projects)
```

### Effective Go + Code Review Comments
```
What:     Official Go team guidance on idiomatic Go
Helling must follow:
  - [ ] gofumpt formatting (stricter than gofmt)
  - [ ] goimports for import grouping (stdlib, external, internal)
  - [ ] Errors: wrap with fmt.Errorf("context: %w", err), never discard
  - [ ] Naming: MixedCaps, not snake_case. Short receiver names.
  - [ ] Interfaces: accept interfaces, return structs
  - [ ] Context: first parameter, passed through call chain
  - [ ] Goroutines: always have a termination path, use errgroup
  - [ ] Tests: table-driven, in *_test.go files, use testify for assertions
  - [ ] golangci-lint with gosec, govet, errcheck, staticcheck enabled
```

### Go Module Versioning
```
What:     How Go modules handle versions
Requires: SemVer, go.mod version constraints
Helling:
  - [ ] All three modules (hellingd, helling-cli, helling-proxy) share go.work
  - [ ] Dependencies pinned to specific versions (not floating)
  - [ ] go mod tidy on every PR
  - [ ] go mod verify in CI
  - [ ] Dependabot for automated dependency updates
```

---

## 6. Software Engineering Standards

### Semantic Versioning (SemVer 2.0.0)
```
What:     MAJOR.MINOR.PATCH versioning standard
Helling:  ✓ Planned: v0.1.0 → v0.5.0 → v1.0.0
Rules:
  - Pre-v1: API can change without MAJOR bump (0.x.y)
  - Post-v1: breaking API changes require MAJOR bump
  - PATCH: bug fixes only
  - MINOR: new features, backward compatible
  - Pre-release: v0.1.0-alpha, v0.1.0-beta, v0.1.0-rc.1
```

### Conventional Commits
```
What:     Standard for commit message format
Format:   type(scope): description
Helling:  ✓ Planned with pre-commit hook enforcement
Types:    feat, fix, docs, test, refactor, ci, chore, perf, build
Breaking: feat!: or BREAKING CHANGE: footer
Enables:  Automated changelog (git-cliff), SemVer determination
```

### Keep a Changelog
```
What:     Standard for changelog format
Requires: CHANGELOG.md with Added, Changed, Deprecated, Removed, Fixed, Security sections
Helling:  ✗ NOT IMPLEMENTED
Action:   Generate from conventional commits using git-cliff
```

### 12-Factor App
```
What:     Methodology for building cloud-native applications
Helling compliance:
  I.    Codebase:        ✓ One repo, multiple deploys
  II.   Dependencies:    ✓ Go modules, explicit deps
  III.  Config:          ⚠ helling.yaml + env vars (need full env var support)
  IV.   Backing services: ✓ Incus, Podman, SQLite are attachable resources
  V.    Build/release/run: ⚠ Need strict separation (GoReleaser handles this)
  VI.   Processes:       ✓ hellingd is stateless (state in SQLite + Incus)
  VII.  Port binding:    ✓ Self-contained HTTP server
  VIII. Concurrency:     ✓ goroutines for concurrent operations
  IX.   Disposability:   ⚠ Need graceful shutdown (SIGTERM handling)
  X.    Dev/prod parity: ✓ Single mode (Incus + Podman on bare metal)
  XI.   Logs:            ⚠ Need structured JSON to stdout (not files)
  XII.  Admin processes: ✓ CLI for admin tasks
```

### DCO (Developer Certificate of Origin)
```
What:     Lightweight alternative to CLA for open source contributions
Requires: Signed-off-by line in every commit
Helling:  ✓ AGPL-3.0 + DCO chosen (stated in CLAUDE.md)
Action:
  - [ ] Document DCO requirement in CONTRIBUTING.md
  - [ ] Add DCO check in CI (GitHub Action: dco-check)
  - [ ] Reject PRs without Signed-off-by
```

---

## 7. Observability Standards

### Prometheus Exposition Format
```
What:     Standard text format for metrics endpoints
Requires: /metrics endpoint serving metrics in Prometheus format
Helling:  ✓ Planned with prometheus/client_golang
Action:
  - [ ] Implement /metrics endpoint
  - [ ] Follow naming conventions: helling_<subsystem>_<metric>_<unit>
  - [ ] Include TYPE and HELP for every metric
  - [ ] Include helling_build_info{version, goversion}
```

### Structured Logging
```
What:     JSON-formatted log output with standard fields
Standard: No single standard, but common fields from ECS (Elastic Common Schema)
Helling:
  - [ ] All logs as JSON via slog
  - [ ] Standard fields: timestamp, level, message, logger, caller
  - [ ] Request fields: request_id, method, path, status, duration, user, source_ip
  - [ ] Error fields: error, stack_trace
  - [ ] Context fields: instance_name, operation, task_id
```

### Syslog RFC 5424
```
What:     Standard for system log message format
Requires: Structured syslog messages when forwarding to external
Helling:
  - [ ] Syslog forwarding in RFC 5424 format (not legacy RFC 3164)
  - [ ] Facility: local0 (configurable)
  - [ ] Severity mapping: slog levels → syslog severity
  - [ ] Structured data: include Helling-specific metadata
```

---

## 8. Accessibility

### WCAG 2.1 AA
```
What:     Web Content Accessibility Guidelines
Requires: 50 success criteria for accessible web content
Helling dashboard:
  - [ ] Perceivable: text alternatives, captions, color contrast ≥4.5:1
  - [ ] Operable: keyboard navigable, no time limits, skip navigation
  - [ ] Understandable: readable, predictable, input assistance
  - [ ] Robust: compatible with assistive technologies
  
  antd has built-in ARIA support. Key actions:
  - [ ] Test with screen reader (NVDA, VoiceOver)
  - [ ] Test keyboard-only navigation through all pages
  - [ ] Run axe-core or Lighthouse accessibility audit
  - [ ] Ensure all status communicated by icon+color+text (not color alone)
  - [ ] Focus management on modal open/close
  - [ ] Alt text on all informational images/icons
Priority: Medium. Compliance improves usability for everyone.
```

---

## 9. Licensing & Legal

### SPDX License Headers
```
What:     Machine-readable license identifiers in source files
Standard: REUSE specification (reuse.software)
Requires: Every file has SPDX header or is covered by .reuse/dep5
Helling:
  - [ ] Every .go file: // SPDX-License-Identifier: AGPL-3.0-or-later
  - [ ] Every .tsx/.ts file: // SPDX-License-Identifier: AGPL-3.0-or-later
  - [ ] Every .yaml/.md file: covered by .reuse/dep5
  - [ ] Run `reuse lint` in CI
  - [ ] REUSE badge in README
```

### License Compatibility
```
Helling is AGPL-3.0-or-later. Dependencies must be compatible:
  ✓ Compatible: MIT, BSD-2, BSD-3, Apache-2.0, ISC, MPL-2.0, LGPL-2.1+, LGPL-3.0+
  ✗ Incompatible: GPL-2.0-only (without "or later"), SSPL, BSL, proprietary
  
  - [ ] License check in CI: scancode-toolkit or go-licenses
  - [ ] Block PRs that add incompatible-licensed dependencies
  - [ ] Document all dependency licenses in NOTICE file
```

---

## 10. Release Engineering

### GoReleaser Configuration
```
What:     Standard Go release automation tool
Helling:
  - [ ] .goreleaser.yml with linux/amd64 + linux/arm64 targets
  - [ ] Checksums (SHA-256) for all artifacts
  - [ ] Cosign signing of all binaries and container images
  - [ ] SLSA provenance attestation
  - [ ] SBOM (CycloneDX) attached to release
  - [ ] Changelog from conventional commits (git-cliff or goreleaser built-in)
  - [ ] .deb package generation via nfpm (primary distribution format)
  - [ ] Container image for CI/try-it mode only (not primary deployment)
```

### Container Image Standards
```
What:     Best practices for OCI container images
Helling container image (CI + optional try-it mode, not primary deployment):
  - [ ] Multi-stage build (builder → runtime)
  - [ ] Non-root user in runtime image
  - [ ] Minimal base image (debian-slim or distroless)
  - [ ] OCI annotations: org.opencontainers.image.* labels
  - [ ] Health check instruction (HEALTHCHECK CMD)
  - [ ] No secrets in image layers
  - [ ] .containerignore excludes unnecessary files
  - [ ] Pinned base image digests (not :latest)
  - [ ] SBOM embedded as OCI annotation
  - [ ] Cosign signature on pushed images
```

---

## Compliance Summary

| Standard | Status | Priority | Target |
|----------|--------|----------|--------|
| OCI (container images) | ✓ Compliant | - | Done |
| CNI/CSI/CRI (K8s) | ✓ Compliant | - | Done |
| OpenTelemetry | ✗ Missing | Medium | v0.5 |
| CloudEvents | ✗ Missing | Low | Post-v1 |
| K8s Conformance | ⚠ Untested | High | v0.5 |
| FHS | ✓ Compliant | - | Done |
| systemd hardening | ⚠ Partial | High | v0.1 |
| AppArmor profile | ✗ Missing | Medium | v0.5 |
| OpenSSF Badge | ✗ Not applied | High | v0.1 |
| SLSA L1 | ✗ Missing | High | v0.1 |
| SLSA L2 | ✗ Missing | High | v0.5 |
| SLSA L3 | ✗ Missing | Medium | v1.0 |
| SBOM (CycloneDX+SPDX) | ⚠ Planned | High | v0.1 |
| OWASP API Top 10 | ⚠ Partial | High | v0.5 |
| CIS Benchmarks | ✗ Missing | Low | Post-v1 |
| OpenAPI 3.1 | ⚠ 3.0 exists | Medium | v0.1 |
| OAuth 2.0 / OIDC | ✗ Missing | Medium | v0.5 |
| SemVer | ✓ Planned | - | Done |
| Conventional Commits | ✓ Planned | - | Done |
| 12-Factor App | ⚠ Mostly | Medium | v0.5 |
| DCO | ✓ Planned | - | Done |
| Prometheus format | ⚠ Planned | High | v0.1 |
| Structured logging | ⚠ Partial | High | v0.1 |
| WCAG 2.1 AA | ✗ Untested | Medium | v0.5 |
| SPDX/REUSE | ⚠ Partial | High | v0.1 |
| License compatibility | ⚠ Unchecked | High | v0.1 |
| GoReleaser + Cosign | ⚠ Planned | High | v0.1 |
| Container image standards | ⚠ Partial | High | v0.1 |
