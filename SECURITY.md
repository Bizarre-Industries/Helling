# Security Policy

## Supported Versions

Helling is in pre-alpha development. No versions are currently supported for
production use. Security advisories will begin with the first tagged release
(`v0.1.0-alpha`).

## Reporting a Vulnerability

**Do not report security issues as public GitHub issues.**

Preferred channel — GitHub Private Vulnerability Reporting:

- <https://github.com/Bizarre-Industries/Helling/security/advisories/new>

Alternative channel — email:

- <mailto:security@bizarre.industries>

When reporting, please include:

- Affected component (hellingd / helling-cli / helling-proxy / helling-agent / web / other)
- Affected version or commit SHA
- Reproduction steps
- Impact assessment
- Suggested mitigation, if available

Encrypted communication is available on request. Reports may be submitted in
English or Arabic.

## Disclosure Process

- **Initial triage acknowledgement target:** 72 hours
- **Status update cadence:** every 7 days until resolution
- **Fix timeline:** depends on severity per the table below
- **Coordinated disclosure:** preferred; embargo negotiated on a case-by-case basis

### Severity targets

| Severity | CVSS v3.1  | Target fix timeline |
| -------- | ---------- | ------------------- |
| Critical | 9.0 – 10.0 | 7 days              |
| High     | 7.0 – 8.9  | 30 days             |
| Medium   | 4.0 – 6.9  | 60 days             |
| Low      | 0.1 – 3.9  | 90 days             |

After a fix is released, a GitHub Security Advisory will be published with a
CVE (if applicable) and credit to the reporter, unless the reporter requests
anonymity.

## Scope

**In scope:**

- Authentication and authorization flaws in hellingd, helling-cli,
  helling-proxy, helling-agent, or the WebUI
- Privilege escalation paths via the API, CLI, or installer
- Sensitive data exposure (tokens, credentials, PII, session material)
- Supply-chain integrity issues in release artifacts, container images, or CI
  build pipelines
- Cryptographic misuse (argon2id parameters, JWT signing, TOTP verification,
  age-encrypted secrets handling)
- Sandbox escape from Incus or Podman workloads managed by Helling
- Issues in the `api/openapi.yaml` contract that enable impersonation, replay,
  or unauthorized access

**Out of scope:**

- Social engineering attacks against maintainers
- Vulnerabilities in unsupported forks or third-party distributions
- Vulnerabilities in upstream dependencies already covered by their own
  security policies (report those to the relevant project; we will coordinate)
- Missing security hardening unrelated to a concrete exploitation path
- Denial of service via resource exhaustion on self-hosted deployments without
  authentication bypass

## Security-related references

- ADR-026 — SHA-pinned GitHub Actions
- ADR-030 — argon2id password hashing
- ADR-039 — age-encrypted secret storage
- ADR-042 — security scanning strategy
- `docs/spec/threat-model.md` — repository threat model
- `docs/standards/quality-assurance.md` — security gates in CI

## Hall of Fame

Reporters who submit valid, in-scope reports will be listed here (with consent):

- _(none yet)_

## Maintainer

- [Suhail (@binGhzal)](https://github.com/binGhzal)
