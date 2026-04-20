# CI Pipeline Specification

<!-- markdownlint-disable MD029 -->

Normative specification of Helling's continuous-integration pipeline. Defines which workflows exist, when they run, what they gate, and what signals they produce.

Implementation lives in `.github/workflows/`. Any divergence between this document and the actual workflow files is a bug in one or the other — open an issue.

## Scope

In scope:

- GitHub Actions workflows that run on push, pull_request, and scheduled events
- Required status checks that gate merge to `main`
- Artifact publication and release pipelines

Out of scope:

- Local developer workflow (see `docs/spec/local-dev.md`)
- Pre-commit hook behavior (see `docs/spec/pre-commit.md`)
- Security scanner configuration details (see ADR-042)

## Principles

1. **CI mirrors local.** Every check run in CI must be runnable locally via `task check`. No CI-only gates. No machine-only passes.
2. **Fail fast.** Fast, cheap checks (lint, format) run first and in parallel. Expensive checks (test, codegen drift, CodeQL) run only after cheap ones pass.
3. **Pin everything.** All action references are pinned by SHA per ADR-026. No `@v1` floating tags.
4. **Read-only by default.** Workflows request minimum permissions. Write permissions are granted only to specific jobs that publish artifacts.

## Workflows

### `quality.yml` — the merge gate

Triggered on every push to a PR branch and every push to `main`. This is the blocker for merges.

Runs in parallel:

| Job                 | Checks                                                                              | Fail condition                                                     |
| ------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `openapi`           | `vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml`      | Score < 100/100, or any rule firing at `info` severity or above    |
| `openapi-generated` | `task check:openapi:generated`                                                      | Committed `api/openapi.yaml` differs from what Huma would emit now |
| `markdown`          | markdownlint-cli2, prettier --check                                                 | Any lint or format violation                                       |
| `yaml`              | yamllint -s                                                                         | Any warning or error                                               |
| `shell`             | shellcheck -S style, shfmt -d                                                       | Any violation                                                      |
| `go`                | build + vet + golangci-lint + govulncheck + go.mod tidy + test with race + coverage | Any fail, or coverage below per-package floors                     |
| `frontend`          | biome check, tsc --noEmit, generated-code drift, bun test                           | Any fail                                                           |
| `sql`               | sqlfluff, sqlc drift, goose round-trip                                              | Any fail                                                           |
| `secrets`           | gitleaks detect                                                                     | Any finding                                                        |
| `spelling`          | typos                                                                               | Any typo                                                           |
| `links`             | lychee --offline                                                                    | Any broken internal link                                           |
| `parity`            | `scripts/check-parity.sh`                                                           | Any operation without CLI + WebUI counterpart or exception         |

Total wall-clock target: under 3 minutes for a typical PR. The Go test job dominates; everything else is under 30 seconds.

Concurrency: cancel in-progress runs on the same ref to save action minutes when pushing rapidly.

### `codeql.yml` — SAST

Triggered on push to `main`, every PR, and weekly on Mondays at 03:00 UTC (scheduled full-depth analysis).

Analyzes Go and TypeScript with the `security-extended` and `security-and-quality` query packs per ADR-042.

Fail condition: any alert at `error` severity. Alerts at `warning` severity do not block merge but must be triaged within 7 days.

### `security.yml` — Grype + OpenSSF Scorecard

Triggered weekly (Mondays 04:00 UTC) and on push to release branches.

Jobs:

- **Grype scan** of built artifacts (`dist/`) and Syft-generated SBOMs. Fails on HIGH or CRITICAL findings.
- **OpenSSF Scorecard** — records project hygiene score. Fails if score drops below 7.0.

### `release.yml` — artifact publication

Triggered on tag push matching `v*.*.*`.

Pipeline:

1. Rebuild all binaries with `-trimpath` and reproducible flags.
2. Generate SBOM with Syft.
3. Scan SBOM with Grype; fail on HIGH.
4. Sign binaries and SBOM with cosign.
5. Build .deb package.
6. Publish to GitHub Releases + static APT repo per ADR-025.

Release details in `docs/standards/release.md` (once it exists).

## Required status checks for `main` branch protection

These are the gates that must pass before a PR can merge:

- `openapi`
- `openapi-generated`
- `markdown`
- `yaml`
- `shell`
- `go`
- `frontend`
- `sql`
- `secrets`
- `spelling`
- `links`
- `parity`
- `CodeQL`

Branch protection also requires:

- 1 reviewer approval (self-approval on solo-maintainer phase; collaborators added later)
- Linear history (no merge commits)
- Signed commits
- Conversations resolved

## Caching policy

CI jobs should cache aggressively to stay under action-minute budget:

- Go module cache (`$GOPATH/pkg/mod`) keyed on `go.sum`
- Go build cache (`~/.cache/go-build`) keyed on `go.sum` + branch
- Bun lockfile cache (`web/node_modules`) keyed on `web/bun.lock`
- golangci-lint cache keyed on `.golangci.yaml` + `go.sum`

Cache hit should reduce full `task check` to under 90 seconds.

## Failure handling

When a workflow fails:

- **Lint / format fails:** fix locally via `task fmt` and push. Do not add suppressions without reason.
- **Test fails:** fix the test or the code. Flakes go into `docs/standards/testing.md` flake register (once that doc exists).
- **OpenAPI drift:** run `task gen:openapi` locally and commit the result.
- **Frontend codegen drift:** run `(cd web && bun run gen:api)` and commit.
- **Parity gap:** add a CLI command, WebUI route, or an exception entry in `docs/roadmap/phase0-parity-exceptions.yaml`.
- **CodeQL finding:** triage in the Security tab. Fix or suppress with justification (filed as waiver per QA standards §13).
- **Grype finding:** update the vulnerable dependency. If no patched version exists, file a waiver with expiry.

## Action-minute budget

Helling is solo-maintained on the free GitHub Actions tier (2000 minutes/month on private; unlimited on public). To stay within budget:

- `quality.yml` runs only on pushes that change tracked files (not on tag-only pushes).
- Docker-based workflows avoided; native runners preferred.
- Matrix builds limited to one Go version (1.26) and one OS (ubuntu-24.04) until a compatibility question demands wider coverage.
- `pre-push` git hook mirrors `task check` locally, so developers catch failures before pushing.

If budget exceeds 60% sustained usage for two consecutive months, revisit caching and matrix shape.

## Related documents

- `docs/spec/local-dev.md` — how developers run the same checks locally
- `docs/spec/pre-commit.md` — what runs before commit and push
- `docs/standards/quality-assurance.md` — normative quality gates this pipeline enforces
- `docs/decisions/026-sha-pin-github-actions.md` — action pinning policy
- `docs/decisions/042-security-scanning.md` — security scanner stack
