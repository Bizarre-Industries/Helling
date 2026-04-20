# ADR-042: Security scanning stack: CodeQL + Grype + govulncheck + gitleaks

> Status: Accepted (2026-04-20)
>
> Supersedes all scanning-tool references in `docs/standards/security.md` — specifically §1 (application container-image scan), §3 (dependency security), and §4 (scanning pipeline). Any remaining mention of Trivy, Semgrep, Bearer, Snyk Container, or osv-scanner in those sections is stale and must be removed when this ADR is applied.

## Context

The prior security scanning pipeline named a mix of commercial and redundant tools:

- CodeQL: not listed, missing
- govulncheck: listed, works well for Go
- gitleaks: listed, works well for pre-commit + CI
- golangci-lint / gosec: listed, works at code level
- Grype: listed, works for artifacts + SBOMs
- Semgrep: listed, overlaps with CodeQL for SAST
- Bearer: listed, commercial license required
- Snyk Container: listed, commercial license required
- osv-scanner: listed, overlaps with govulncheck for Go coverage
- OpenSSF Scorecard: listed, independent value for project hygiene
- Trivy: referenced in `security.md` §1 as "additional scanning layer"; suspended by ADR-009 after the March 2026 Aqua supply-chain attack

Helling is a solo-developed open-source project on GitHub. Commercial scanner licenses are a non-starter. Tool overlap is a maintenance burden without additive coverage. And Trivy specifically is paused until upstream Aqua practices meet Helling's supply-chain bar (ADR-009).

## Decision

Consolidate to a minimal, non-overlapping, no-commercial-license scanning stack:

- **CodeQL** — primary SAST for Go and TypeScript, security-extended + security-and-quality query packs.
- **Grype** — vulnerability scanning of built artifacts (binaries, .deb) and Syft-generated SBOMs.
- **govulncheck** — Go module vulnerability scanner with symbol-awareness (more precise than generic package scanners for Go).
- **gitleaks** — committed secret detection.
- **OpenSSF Scorecard** — weekly project hygiene metrics.
- **golangci-lint** with **gosec** — code-level security linting inside `golangci.yaml`.

Removed tools:

- **Semgrep** — superseded by CodeQL's security-extended queries.
- **Bearer** — commercial license.
- **Snyk Container** — commercial license, superseded by Grype.
- **osv-scanner** — redundant with govulncheck for Go; GitHub Dependabot handles general dependency advisories.
- **Trivy** — suspended per ADR-009; removed from `security.md` §1 container-image pipeline.

## Gates

- Every push: CodeQL, govulncheck, gitleaks, golangci-lint.
- Release branches + weekly: Grype against built artifacts and SBOMs.
- Weekly: OpenSSF Scorecard.
- Failing severity: HIGH or CRITICAL on any scanner blocks merge.
- CodeQL: any alert at `error` severity blocks merge; alerts at `warning` severity require written triage within 7 days.

## Consequences

**Easier:**

- No commercial license dependencies.
- Non-overlapping coverage: SAST (CodeQL), artifact vulns (Grype), Go-specific vulns (govulncheck), secrets (gitleaks), project hygiene (Scorecard).
- Symbol-aware Go scanning means fewer false positives in CI.
- Single SARIF surface (CodeQL Security tab) for most findings.

**Harder:**

- CodeQL builds are slower than Semgrep; compensate with scheduled weekly full-depth runs and per-PR differential queries.
- Losing Semgrep's custom-rule ecosystem; write custom CodeQL queries when needed (Go query pack + custom queries in `.github/codeql-queries/`).
- Grype requires SBOM generation in the release pipeline (Syft is already in use, so this is additive, not new).

## References

- `docs/standards/quality-assurance.md` §9 for the normative scanning matrix.
- `docs/standards/security.md` — §1, §3, §4 all require the sweep noted at the top of this ADR.
- ADR-009 (no Trivy — reaffirmed here).
- ADR-026 SHA-pin GitHub Actions (all security tooling actions pinned by SHA).
