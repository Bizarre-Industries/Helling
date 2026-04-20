# Infrastructure & Operations Standards

How Helling is deployed, monitored, backed up, and operated. Based on SRE principles (Google), DORA metrics (Accelerate research), and production infrastructure best practices.

---

## 1. Deployment Standards

### Incus Transport Listener Policy

```text
RULE: Delegated-user Incus operations must use the local HTTPS listener with mTLS identity.

Requirements:
  - Set `core.https_address` to `127.0.0.1:8443` on the host.
  - `/api/incus/*` requests from hellingd must present the caller's per-user TLS certificate.
  - Incus Unix socket is reserved for host administrator CLI operations only.
  - Do not use query-parameter project scoping as an authorization boundary.

Verify:
  - `incus config get core.https_address` returns `127.0.0.1:8443`
  - Delegated user calls are denied when trust restrictions do not allow the action.
```

### Incus Minimum Version

```text
RULE: Helling requires Incus >= 6.14.0.

Why:
  - CVE-2025-52889 (DoS via DHCP lease exhaustion on bridge networks)
  - CVE-2025-52890 (firewall rule bypass on bridge networks with ACLs, CVSS 8.1)
  - CVE-2025-4115 (local privilege escalation via custom storage volumes)

All three affect Incus 6.12 and 6.13 and are patched in 6.14+.

Enforcement:
  - ISO build manifest pins `incus >= 6.14.0` via the Zabbly apt repo
  - hellingd boot-time version check disables the Incus proxy and logs a
    HIGH-severity structured journal entry if below minimum
  - See docs/spec/platform.md §11 for the authoritative version matrix
```

### Build Reproducibility

```yaml
RULE: Any commit on main produces identical artifacts regardless of who builds it.

Requirements:
  - Go: CGO_ENABLED=1, pinned Go version in go.mod, go.sum verified
  - React: bun.lockb committed, deterministic build output
  - .deb: nfpm config pinned, reproducible package builds
  - ISO: live-build/mkosi config versioned, reproducible image builds
  - CI: pinned action SHAs, pinned tool versions
  - SBOM: artifact composition recorded for every build

Verify: `make build` on fresh clone produces byte-identical binaries
(modulo timestamps, which GoReleaser strips with -trimpath -ldflags=-s -w)
```

### Release Process

```text
1. Feature freeze: merge window closes
2. Release branch: cut from main
3. Version bump: update version constant, changelog
4. CI: full pipeline (build, test, lint, security scan, integration test)
5. Artifacts: GoReleaser produces binaries, .deb packages (via nfpm)
6. Signing: Cosign signs all artifacts, SLSA provenance generated
7. SBOM: Syft generates CycloneDX + SPDX, attached to release
8. Publish: GitHub Release, APT repo update (via reprepro per ADR-045), optional ghcr.io push
9. Announce: changelog, blog post, notification channels
10. Monitor: watch error rates, support channels for 48 hours

Rollback: if critical issue found within 48 hours, yank release and publish patch.

Tooling:
  - goose + sqlc: SQL-first migrations and typed query generation
  - nfpm: .deb package generation (hellingd, helling-cli, edge service config)
  - reprepro: APT repository management (hosts .deb packages for apt-get upgrade, see ADR-045)
  - live-build / mkosi: ISO image building (bootable installer)
  - GoReleaser: orchestrates build + package + sign + publish
```

### Upgrade Safety

```bash
RULE: Upgrades must be safe, reversible, and non-destructive.
RULE: Primary upgrade path is APT-based: apt-get update && apt-get upgrade helling

Pre-upgrade:
  - Automatic backup of hellingd SQLite database
  - dpkg stores previous package version for rollback
  - Verify package signature (GPG-signed .deb via reprepro repo per ADR-045)

During upgrade (handled by .deb postinst script):
  - systemctl stop caddy (dashboard offline)
  - systemctl stop hellingd (API offline)
  - dpkg replaces binaries
  - systemctl start hellingd (auto-applies database migrations via goose)
  - Health check (GET /health?detail=true via curl in postinst)
  - If health check fails: postinst exits non-zero, dpkg marks package as failed
  - systemctl start caddy

Post-upgrade:
  - Dashboard shows "What's New" banner
  - Verify: all services healthy, all instances accessible
  - Monitor error rates for 24 hours

Rollback:
  apt-get install helling=<previous-version>
  OR: helling upgrade --rollback
  → Restores previous .deb package version
  → Restores previous SQLite from /var/lib/helling/backups/
  → Restarts services
  → Verifies health

RULE: Schema migrations (goose) must be forward-only but the application must handle
running against an older schema gracefully (new columns have defaults, removed
columns are ignored). This allows rollback without schema rollback.
```

---

## 2. Monitoring Standards

### Golden Signals (Google SRE)

```yaml
Monitor these four signals for hellingd:

1. Latency: time to serve API requests
   Metric: helling_api_request_duration_seconds (histogram)
   SLO: p50 < 50ms, p95 < 200ms, p99 < 1s
   Alert: p95 > 500ms for 5 minutes

2. Traffic: request rate
   Metric: helling_api_requests_total (counter)
   Normal: varies by installation
   Alert: sudden 10x increase (possible DoS)

3. Errors: rate of failed requests
   Metric: helling_api_errors_total (counter, by status code)
   SLO: error rate < 1% of total requests
   Alert: error rate > 5% for 2 minutes

4. Saturation: resource utilization approaching limits
   Metrics: helling_goroutines, helling_open_connections, helling_db_size_bytes
   Alert: goroutines > 10000, open connections > 1000, DB > 1GB
```

### USE Method (Brendan Gregg) for Infrastructure

```text
For every resource (CPU, RAM, disk, network):

Utilization: percentage of resource used
  helling_node_cpu_usage_percent
  helling_node_memory_used_bytes / helling_node_memory_total_bytes
  helling_storage_pool_used_bytes / helling_storage_pool_total_bytes

Saturation: degree of queuing / contention
  CPU load average (> core count = saturated)
  Disk queue depth (> 1 sustained = saturated)
  Memory swap usage (> 0 = RAM saturated)

Errors: error count
  Disk SMART errors
  Network interface errors / drops
  ECC memory errors (if available via BMC)
```

### RED Method for API Endpoints

```yaml
For every API endpoint:

Rate: Requests per second
  helling_api_requests_total{method, path}

Errors: Failed requests per second
  helling_api_errors_total{method, path, status}

Duration: Time per request
  helling_api_request_duration_seconds{method, path}
```

### Structured Logging Standard

```json
{
  "timestamp": "2026-04-13T14:22:01.234Z",
  "level": "info",
  "message": "instance created",
  "logger": "hellingd.incus",
  "caller": "incus/instances.go:142",
  "request_id": "req-abc123",
  "user": "admin",
  "source_ip": "192.168.1.10",
  "instance": "vm-web-1",
  "type": "virtual-machine",
  "duration_ms": 1234,
  "task_id": "task-def456"
}
```

Fields:

- `timestamp`: RFC 3339 with milliseconds
- `level`: debug, info, warn, error (lowercase)
- `message`: human-readable description (lowercase, no period)
- `logger`: hierarchical logger name (hellingd.subsystem)
- `caller`: file:line (for debug/error only)
- `request_id`: propagated through entire request lifecycle
- All other fields: snake_case, typed values (not stringified numbers)

---

## 3. Backup Standards

### 3-2-1 Rule

```text
3 copies of data
2 different storage types
1 offsite copy

Helling's implementation:
  Copy 1: Live data (Incus storage pools)
  Copy 2: Local backup (different pool or local directory)
  Copy 3: Remote backup (NFS, S3, or offsite Helling instance)
```

### Backup Schedule Default

```yaml
Instances (VMs + CTs):
  Daily: 01:00 AM, retain 7
  Weekly: Sunday 02:00 AM, retain 4
  Monthly: 1st of month 03:00 AM, retain 6

hellingd database:
  Daily: automatic, retain 7
  Before every upgrade: automatic

Compression: zstd (level 3, good ratio + speed)
Verification: weekly automatic restore test
```

### Recovery Time Objectives

```text
RTO (Recovery Time Objective):
  hellingd crash:     < 30 seconds (systemd auto-restart)
  Node failure (HA):  < 5 minutes (HA failover + VM restart)
  Full node rebuild:  < 2 hours (install + restore config + restore backups)
  Disaster recovery:  < 24 hours (new hardware + full restore)

RPO (Recovery Point Objective):
  With snapshots:     < 4 hours (snapshot schedule)
  With daily backups: < 24 hours
  With replication:   < 15 minutes
```

---

## 4. Capacity Planning

### Resource Thresholds

```text
Green (healthy):     < 70% utilization
Yellow (warning):    70-85% utilization
Orange (critical):   85-95% utilization
Red (emergency):     > 95% utilization

Apply to: CPU, RAM, storage, network bandwidth, disk IOPS

Actions:
  Warning (70%):  Dashboard indicator, no notification
  Critical (85%): Dashboard banner + notification to configured channels
  Emergency (95%): Dashboard banner + urgent notification + capacity forecast
```

### Growth Tracking

```bash
hellingd tracks 30-day trends for:
  - Total CPU allocation across all instances
  - Total RAM allocation
  - Storage usage per pool
  - Instance count
  - Container count
  - Network bandwidth

Forecast: linear regression on 30-day data → "days until threshold"
Display: /dashboard → Capacity widget with trend arrows and forecast
Report: Monthly capacity report (email, PDF)
```

---

## 5. Change Management

### Change Categories

```text
Standard (no approval needed):
  - Start/stop/restart instances
  - Create snapshots
  - View logs/metrics
  - User self-service operations

Normal (audit logged, no approval):
  - Create/delete instances
  - Modify instance config
  - Create/modify firewall rules
  - Add/remove users
  - Deploy templates/stacks

Critical (confirmation required):
  - Delete storage pools
  - Delete cluster nodes
  - Apply OS updates
  - Upgrade Helling version
  - Modify cluster configuration
  - Wipe disks

Emergency (double confirmation + audit alert):
  - Delete instances with protection enabled
  - Modify admin user accounts
  - Disable 2FA for a user
  - Export/import full configuration
```

### Maintenance Windows

```text
Scheduled maintenance:
  - Define recurring windows in /settings
  - Notifications sent 24h before
  - Dashboard shows banner during window
  - Alerts suppressed for expected disruptions
  - Auto-operations (updates, restarts) only run during windows

Unscheduled maintenance:
  - Requires explicit override confirmation
  - Extra audit logging
  - Immediate notification to all configured channels
```

---

## 6. SRE Practices

### SLOs (Service Level Objectives)

```yaml
Helling platform SLOs:

API Availability:
  Target: 99.9% (8.7 hours downtime per year)
  Measurement: successful responses / total responses (excluding planned maintenance)

API Latency:
  Target: p95 < 200ms for reads, p95 < 2s for writes
  Measurement: helling_api_request_duration_seconds histogram

Backup Reliability:
  Target: 99% of scheduled backups complete successfully
  Measurement: successful backups / scheduled backups per month

Dashboard Availability:
  Target: 99.5% (43 hours downtime per year)
  Measurement: health check success rate from external monitor

These are internal targets, not contractual SLAs.
Tracking enables data-driven improvement decisions.
```

### Error Budgets

```text
99.9% availability = 0.1% error budget = 8.7 hours/year

When error budget is consumed:
  - Freeze new features
  - Focus on reliability improvements
  - Increase testing coverage
  - Review and fix top error sources

When error budget is healthy:
  - Ship features faster
  - Accept more risk on non-critical changes
  - Experiment with new capabilities
```

### Incident Severity Levels

```yaml
P1 (Critical): Platform completely down, all users affected
  Response: Immediate (< 15 min), all-hands
  Examples: hellingd won't start, database corruption, security breach

P2 (Major): Significant functionality impaired
  Response: Within 1 hour
  Examples: Incus unavailable (no VM management), backup system failing

P3 (Minor): Single feature degraded, workaround exists
  Response: Within 4 hours
  Examples: Dashboard slow, one container type failing, metrics delayed

P4 (Low): Cosmetic or minor inconvenience
  Response: Next business day
  Examples: UI glitch, typo in error message, non-critical warning
```

### Post-Incident Review

```yaml
After every P1/P2 incident:

Document:
  1. Timeline: what happened, when, detected how
  2. Impact: what was affected, for how long, how many users
  3. Root cause: why it happened (5 Whys analysis)
  4. Resolution: what fixed it
  5. Prevention: what changes prevent recurrence
  6. Action items: specific tasks with owners and deadlines

Principles:
  - Blameless: focus on systems, not people
  - Thorough: dig to root cause, not symptoms
  - Actionable: every review produces concrete improvements
  - Shared: published to team (or community for open source)
```

---

## 7. DORA Metrics

Track these four metrics (from Accelerate research by Forsgren, Humble, Kim):

```yaml
Deployment Frequency:
  Target: Multiple times per week (Elite performance)
  Measurement: GitHub releases per month

Lead Time for Changes:
  Target: < 1 day from commit to production release
  Measurement: time from merge to main → release published

Change Failure Rate:
  Target: < 15%
  Measurement: releases requiring hotfix or rollback / total releases

Mean Time to Recovery (MTTR):
  Target: < 1 hour
  Measurement: time from incident detection to resolution

Track these in a dashboard (private, for project health assessment).
Not exposed to users — internal quality metric.
```

---

## 8. Documentation Standards

### Code Documentation

```text
Go:
  - Package comment on every package (// Package X provides...)
  - Exported function comment on every exported function
  - No comment on obvious functions (getters, simple wrappers)
  - godoc must render clean for all public APIs

React:
  - JSDoc on all component props interfaces
  - README.md per complex component explaining usage
```

### User Documentation

```text
Structure (docs/ or documentation site):
  Getting Started:
    - Quick start (ISO install or try-it container)
    - Installation (bare metal)
    - First VM tutorial
    - First container tutorial

  Guides:
    - Networking setup
    - Storage configuration
    - Backup and recovery
    - Kubernetes clusters
    - Clustering and HA
    - Security hardening

  Reference:
    - API documentation (generated from OpenAPI)
    - CLI reference (generated from Cobra)
    - Configuration reference (helling.yaml)
    - Architecture overview

  Troubleshooting:
    - Common issues and solutions
    - Diagnostics (`helling doctor`)
    - Log analysis
    - Recovery procedures

RULE: Every new feature ships with documentation. Code without docs is incomplete.
RULE: Every error code links to a troubleshooting page.
RULE: Screenshots are real (not mockups) and updated with each release.
```

### API Documentation

```text
OpenAPI spec (api/openapi.yaml) is the source of truth.

Requirements:
  - Every endpoint has: summary, description, request/response schemas
  - Every schema has: description, example values
  - Every error response documented with code + message + action
  - Served at /api/docs (Swagger UI) and /api/reference (Redoc)
  - Downloadable at /api/v1/openapi.yaml
  - Versioned: breaking changes → new API version
  - Changelog: generated from OpenAPI diffs between releases (oasdiff)
```
