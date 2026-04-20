# Security Standards

<!-- markdownlint-disable MD040 -->

Security requirements for Helling at every layer: application, infrastructure, supply chain, operations, and incident response.

---

## 1. Application Security

### Authentication

```yaml
Passwords:
  - User password verification: delegated to PAM
  - Helling-managed secret hashing: argon2id (NOT SHA-256, NOT MD5)
  - Minimum length: 8 characters
  - No maximum length enforced by Helling auth layer
  - No composition rules (no "must have uppercase + number + symbol")
  - Check against HaveIBeenPwned top 100K passwords (offline list)
  - Rate limit: 5 attempts per 15 minutes per IP + per username

JWT:
  - Algorithm: EdDSA (Ed25519). NEVER HS256.
  - Access token expiry: 15 minutes (short-lived)
  - Refresh token expiry: 7 days (stored server-side, revocable)
  - Include: user ID, username, roles, issued_at, expiry, jti (unique ID)
  - Rotate signing key: document procedure, support key rollover via JWKS
  - Store access token: memory only (not localStorage)
  - Store refresh token: httpOnly, Secure, SameSite=Strict cookie

API Tokens:
  - Format: helling_<random 40 chars> (identifiable prefix)
  - Hash: SHA-256 before storage (token itself never stored)
  - Scopes: read, write, admin (granular per resource type)
  - Expiry: configurable, default 90 days, maximum 365 days
  - Revocation: immediate, stored in revocation list checked on every request
  - Display: show only at creation time, never again

2FA:
  - TOTP: RFC 6238, 6-digit, 30-second step, SHA-1 (compatibility)
  - WebAuthn: Level 2, discoverable credentials, user verification preferred
  - Recovery codes: 10 codes, 16 chars each, argon2id hashed, single-use
  - 2FA lockout: 5 failed attempts → require recovery code
  - Recovery: documented process requiring identity verification
```

### Proxy Auth Model (ADR-014)

```text
The proxy validates JWT before forwarding. For Incus calls,
the proxy presents the authenticated user's dedicated TLS client certificate. Incus
trust restrictions enforce scope. Podman requests are forwarded over the Podman
Unix socket, accessible only by hellingd (running as root).

Auth flow for proxied requests:
  1. Client sends request with JWT (Authorization header or cookie)
  2. hellingd middleware validates JWT, extracts user + roles
  3. Middleware loads user Incus TLS client certificate
  4. Request forwarded to Incus HTTPS API using per-user mTLS identity, or to Podman via Unix socket
  5. Audit middleware logs the action (user, resource, method, timestamp)

Auth flow for Helling-specific endpoints:
  1. Same JWT validation as above
  2. Handler checks role-based permissions via authorization middleware
  3. Handler calls service layer
  4. Audit middleware logs the action
```

### Session Security

```text
  - Session timeout: 30 minutes inactivity (configurable)
  - Concurrent sessions: unlimited (but viewable + individually revocable)
  - Session binding: tie to IP + User-Agent (warn on change, don't block)
  - Logout: invalidate access token, delete refresh token, clear cookies
  - Force logout: admin can terminate any user's sessions
```

### Request Security

```yaml
- CORS: restrict to dashboard origin only
- CSRF: SameSite=Strict cookies + custom X-CSRF-Token header
- X-Helling-Token is deprecated and must not be used in new code
- Request size: 10MB max body (configurable), 8KB max headers
- Request timeout: 30s default, 300s for uploads/backups
- Request ID: X-Request-ID generated per request, logged + returned
- TLS: minimum TLS 1.2, prefer TLS 1.3. No SSL 3.0, TLS 1.0, TLS 1.1.
- Cipher suites: AEAD only (AES-GCM, ChaCha20-Poly1305). No CBC.
- HSTS: max-age=63072000, includeSubDomains
```

### Data Security

```text
At rest:
  - SQLite: file permissions 0600 (owner read/write only)
  - Secrets in DB: encrypted with age (`filippo.io/age`), key material external to DB
  - Backup encryption: optional passphrase-protected age recipients (scrypt)
  - ZFS encryption: supported at pool level (Incus manages)
  - Audit logs: append-only, immutable once written

In transit:
  - All API traffic: TLS (self-signed minimum, ACME recommended)
  - Incus API: HTTPS with mutual TLS using per-user client certificates
  - Podman socket: Unix socket (no network exposure)
  - Cluster communication: TLS mutual authentication (Incus handles)
  - Backup transfer: TLS to remote targets
  - Webhook delivery: HTTPS only, with HMAC-SHA256 signature

Sanitization:
  - Logs: never contain passwords, tokens, keys, certificates, PII
  - Errors: never expose internal paths, database errors, stack traces to clients
  - API responses: never include fields the user doesn't have permission to see
  - Support bundles: passwords and tokens auto-redacted
```

---

## 2. Infrastructure Security

### Host Hardening

```yaml
Debian 13 base:
  - Automatic security updates: unattended-upgrades for security pocket
  - SSH: key-only authentication, no password login, no root login
  - SSH: non-standard port optional, fail2ban configured
  - Firewall: nftables default-deny inbound, allow only 8006 (dashboard) + SSH
  - Unnecessary services disabled
  - Kernel: sysctl hardening (net.ipv4.conf.all.rp_filter=1, etc.)
  - File permissions: umask 027 for services
  - /tmp: noexec, nosuid, nodev mount options
  - Audit: auditd for system-level auditing (complementary to hellingd audit)
```

### systemd Hardening (hellingd)

```toml
[Service]
# Process isolation
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Capabilities
CapabilityBoundingSet=CAP_DAC_OVERRIDE
AmbientCapabilities=

# Filesystem
ReadWritePaths=/var/lib/helling /var/log/helling /etc/helling /run/helling
ReadOnlyPaths=/usr/bin

# Network
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
IPAddressDeny=any
IPAddressAllow=localhost
IPAddressAllow=10.0.0.0/8
IPAddressAllow=172.16.0.0/12
IPAddressAllow=192.168.0.0/16

# System calls
SystemCallFilter=@system-service @network-io @file-system
SystemCallArchitectures=native
MemoryDenyWriteExecute=true

# Resource limits
LimitNOFILE=65535
LimitNPROC=4096
TasksMax=4096

# Watchdog
WatchdogSec=30
Restart=on-failure
RestartSec=5
```

### Container Security Defaults

```text
Podman containers created via Helling:
  - Run as non-root by default (rootless Podman)
  - No --privileged unless explicitly enabled per container
  - Read-only root filesystem encouraged
  - No new privileges (--security-opt=no-new-privileges)
  - Drop all capabilities, add only needed ones
  - Resource limits enforced (CPU, memory, PIDs)
  - Seccomp profile: runtime/default
  - AppArmor profile: runtime/default

Incus containers:
  - Unprivileged by default (security.privileged=false)
  - AppArmor profile enforced
  - Seccomp filtering enabled
  - UID/GID mapping for user namespace isolation
```

---

## 3. Supply Chain Security

### Build Pipeline

```text
SLSA Level 3 target:
  - Source: GitHub with branch protection, signed commits encouraged
  - Build: GitHub Actions (ephemeral runners, no self-hosted)
  - Provenance: SLSA provenance attestation via slsa-github-generator
  - Signing: Cosign keyless signing (Sigstore Fulcio + Rekor)
  - SBOM: CycloneDX + SPDX generated by Syft, attached to release
  - Verification: cosign verify on all released artifacts

Dependency security:
  - govulncheck: zero known vulnerabilities at release time
  - osv-scanner: additional vulnerability database coverage
  - Dependabot: automated patch PRs (patch only, manual review for minor/major)
  - License scan: no AGPL-incompatible dependencies
  - Pin all dependencies to exact versions (go.sum verification)
  - Pin all CI actions to SHA hashes (not version tags)

Container images:
  - Multi-stage builds (builder separate from runtime)
  - Minimal runtime image (debian-slim, no dev tools)
  - Non-root user in runtime
  - Grype scan: zero Critical, zero High at release time
  - Trivy: additional scanning layer
  - Cosign signature on all pushed images
  - SBOM embedded as OCI annotation
  - No secrets baked into any layer

Note: Container images are used for CI and optional try-it mode only.
Production Helling deploys as a .deb package on bare metal (ISO install).
```

### Release Signing

```text
Every release artifact is signed:
  - Go binaries: Cosign keyless signature + SLSA provenance
  - Container images: Cosign keyless signature
  - .deb packages: GPG signed with Helling release key
  - Checksums: SHA-256 for all artifacts
  - Verification:
    cosign verify ghcr.io/bizarre-industries/helling:v0.1.0
    cosign verify-blob --signature helling.sig helling-linux-amd64
```

---

## 4. Vulnerability Management

### Scanning Pipeline

```text
Every push:
  - govulncheck (Go vulnerability database)
  - gitleaks (secret detection in code)
  - golangci-lint with gosec (static security analysis)
  - CodeQL (code scanning)

Push to main:
  - Grype (container image vulnerability scan)
  - Dependency review and lockfile policy checks

Weekly:
  - OpenSSF Scorecard (project security posture)
  - Full dependency audit

Release gate:
  - govulncheck: no unfixed high/critical findings
  - Grype: no unfixed high/critical findings in release image
  - gitleaks: no active secrets

Removed from baseline (ADR-042 consolidation):
  - Semgrep
  - Bearer
  - osv-scanner
  - Snyk Container
```

### Vulnerability Response

```text
SECURITY.md defines:
  - Report to: security@bizarre.industries
  - PGP key for encrypted reports
  - Response timeline:
    Acknowledge: within 48 hours
    Assessment: within 7 days
    Fix (Critical): within 14 days
    Fix (High): within 30 days
    Fix (Medium): within 90 days
    Fix (Low): next release
  - Disclosure: coordinated, 90-day disclosure deadline
  - Credit: reporter credited in advisory (if desired)

CVE process:
  - Request CVE from GitHub Security Advisories
  - Publish advisory with affected versions, fixed version, mitigation
  - Release patched version
  - Notify via: GitHub advisory, release notes, security mailing list
```

---

## 5. Secrets Management

### In Helling Codebase

```text
  - No secrets in source code (gitleaks enforced)
  - No secrets in environment variables baked into container images
  - CI secrets: GitHub Secrets only, never in workflow files
  - Configuration secrets: helling.yaml (file permissions 0600) or env vars
  - Database secrets (user passwords, API tokens): encrypted in SQLite with age recipients
  - Identity key: configured via `secrets.identity_path` (or equivalent env override), file mode 0400
```

### For Users

```text
  - Secrets in containers: Podman secrets API (not environment variables for sensitive data)
  - Secrets in K8s: Kubernetes Secrets (base64, not encrypted by default)
  - Recommend: encrypted etcd for K8s secrets, or external secrets operator
  - Managed databases: connection strings shown once, stored encrypted
  - API tokens: shown once at creation, stored hashed
```

---

## 6. Incident Response

### Preparation

```text
  - Runbooks for common incidents documented in docs/runbooks/
  - Contact list for security team in SECURITY.md
  - Backup verification running (backup-verification feature)
  - Monitoring alerts configured for security events
  - Audit logging active for forensic analysis
```

### Detection

```text
Helling detects and alerts on:
  - Failed login attempts (>5 from same IP)
  - Successful login from new IP/location
  - Admin actions on other users' resources
  - API token usage from unexpected IP
  - Privilege escalation attempts
  - Configuration changes outside maintenance windows
  - Unexpected service restarts
  - Disk/network anomalies
```

### Response

```text
  1. Detect: alert triggers via warning engine or external monitoring
  2. Contain: revoke compromised tokens, isolate affected resources
  3. Investigate: audit log analysis, system log review
  4. Remediate: patch vulnerability, rotate credentials, restore from backup
  5. Report: post-mortem documented in /incidents page
  6. Improve: update runbooks, add detection rules, harden configuration
```

---

## 7. Network Security

### TLS Configuration

```go
tlsConfig := &tls.Config{
    MinVersion:               tls.VersionTLS12,
    PreferServerCipherSuites: false, // Let client choose (TLS 1.3)
    CurvePreferences: []tls.CurveID{
        tls.X25519,    // Fastest, most secure
        tls.CurveP256, // NIST standard
    },
    CipherSuites: []uint16{
        // TLS 1.3 (always preferred, configured automatically)
        // TLS 1.2 fallback:
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
        tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
    },
}
```

### API Security Middleware Stack

```text
Request flow through middleware (order matters):

1. Rate Limiter         → Block excessive requests
2. Request ID           → Generate/propagate X-Request-ID
3. Security Headers     → CSP, HSTS, X-Frame-Options, etc.
4. CORS                 → Restrict origins
5. Request Logger       → Log method, path, source IP, request ID
6. Body Limit           → Reject oversized requests
7. Authentication       → Validate JWT/API token
8. Authorization        → Check user permissions for resource
9. Audit Logger         → Record the authenticated action
10. Router              → Proxy to Incus/Podman socket OR Helling handler
11. Response Logger     → Log status, duration, response size
```

---

## 8. Compliance Readiness

### What Helling enables (not implements itself)

```text
SOC 2:
  - Audit logging → hellingd audit log
  - Access control → RBAC with PAM/LDAP/OIDC
  - Encryption → TLS + at-rest encryption
  - Monitoring → Prometheus + alerts
  - Backup → scheduled + verified

GDPR/Privacy:
  - Data export → config export, backup export
  - Data deletion → instance delete, user delete, audit log retention
  - Access logging → audit log shows who accessed what
  - Encryption → TLS + configurable at-rest

CIS Benchmarks:
  - CIS Debian 13: host hardening checklist
  - CIS Podman: container security defaults
  - CIS Kubernetes: kube-bench in cluster setup

Helling doesn't certify compliance. It provides the tools (audit logs, RBAC,
encryption, monitoring) that compliance requires. Users are responsible for
their specific compliance requirements.
```
