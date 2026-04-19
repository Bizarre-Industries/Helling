# Compliance Standard

This document defines compliance expectations by delivery tier.

## Tier 1 (v0.1 Ship Blockers)

These items block v0.1 release readiness:

- OpenAPI contract exists for Helling-specific API surface
- Semantic versioning discipline for releases
- Conventional commits for change traceability
- SPDX license clarity for shipped artifacts
- AGPL obligations documented and respected
- OCI-compatible container workflows through upstream runtimes
- Filesystem hierarchy alignment for install paths and runtime state
- SECURITY.md present with private disclosure workflow

## Tier 2 (v0.5 Targets)

These items are targeted for v0.5 hardening milestones:

- OpenSSF Best Practices badge at passing level
- SLSA level 1 provenance baseline
- SBOM generation and release attachment
- Routine vulnerability scanning with documented remediation policy
- Expanded identity and access controls with enterprise auth features

## Tier 3 (Post-v1 Aspirations)

These are strategic quality goals and do not block v0.1/v0.5:

- SLSA level 2/3 hardened provenance
- Formal Kubernetes conformance validation flow
- WCAG AA accessibility verification program
- OpenTelemetry adoption for traces/metrics correlation
- CloudEvents event contract standardization
- Full OWASP API Top 10 verification program

## Rule

When a requirement appears in multiple places, this tier map decides release blocking priority.
