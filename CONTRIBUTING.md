# Contributing

Thanks for contributing to Helling.

This repository follows docs-first execution and ADR-driven architecture.
Behavior changes must align with specs/standards/ADRs in `docs/`.

## Core Rules

- Specs and ADRs are authoritative over implementation drift.
- Keep changes minimal and scoped; avoid unrelated refactors.
- Use conventional commits.
- Include validation evidence for every behavioral change.
- Keep generated artifacts in sync.

## Prerequisites

- Go 1.26
- Bun
- make
- git

Bootstrap locally:

```bash
make dev-setup
```

If using the task-based workflow:

```bash
task install
task hooks
```

## Branch and Commit Workflow

1. Create a topic branch from `main`.
2. Implement changes with tests/docs updates in the same branch.
3. Run generation + quality checks.
4. Commit with conventional commit format.
5. Open a PR with a clear scope and checklist.

## Commit Message Format

Use conventional commit prefixes:

- `feat:` new functionality
- `fix:` bug fixes
- `docs:` documentation
- `refactor:` non-functional code changes
- `test:` tests
- `chore:` maintenance
- `ci:` CI/CD configuration
- `build:` packaging/build system

Examples:

- `feat(api): add Huma login endpoint scaffold`
- `docs(standards): align security scanning with ADR-042`

## Signed-off-by (DCO)

All commits must include a Signed-off-by trailer:

```text
Signed-off-by: Your Name <you@example.com>
```

Use:

```bash
git commit -s
```

## Pull Request Requirements

Every PR must include:

- Summary of scope and intent
- Linked issue/ADR/spec references
- Validation commands run and results
- Risk notes for security/compatibility impact

Required checklist before review:

- `make generate`
- `make check-generated`
- `make fmt-check`
- `make lint`
- `make test`

If task workflow is enabled:

- `task check`

## Docs and Contract Changes

If API behavior changes:

1. Update Go/Huma contract source.
2. Regenerate `api/openapi.yaml`.
3. Regenerate dependent clients.
4. Update affected docs in `docs/spec/` and `docs/standards/`.

Do not hand-edit generated files unless explicitly documented.

## Docs / Plans / Checklists Sync

Every PR that touches source (`apps/`, `web/`, `api/openapi.yaml`, `deploy/`, `scripts/`) must also update at least one of:

- `docs/spec/`, `docs/standards/`, `docs/decisions/` (domain spec / standards / ADRs)
- `docs/roadmap/checklist.md` (release gates + per-PR ticking)
- `docs/plans/` (multi-phase plans)
- `docs/audits/` (audit findings + status snapshots)
- `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `SECURITY.md`

Or include a recognized doc-bypass marker in the commit body: `Refs: F-XX`, `Refs: ADR-NNN`, `Refs: #NN`, or `[skip-docs]` for genuine docs-irrelevant changes (formatting, test-only, CI-only, dependency bumps).

### WebUI audit findings

The WebUI v0.2 audit (`docs/audits/webui-2026-04-27.md`) catalogues 51 findings labelled `F-01`..`F-51`. Sequencing lives in `docs/plans/webui-phase-2-6.md`. Per-PR ticking surface is `docs/roadmap/checklist.md`.

When a PR closes a finding:

1. Add `Closes: F-XX` to the commit body (one F-ID per line for multiples). `Refs: F-XX` is for advisory mentions and does not force a tick.
2. Tick the matching `[ ] **F-XX**` line in `docs/roadmap/checklist.md` to `[x]` and append the commit SHA.
3. Update the per-phase status snapshot in `docs/audits/webui-2026-04-27.md` if the close moves a phase to ✅.

### Automated guardrails

- **Git pre-push** (`lefthook.yml` → `docs-sync`): runs `scripts/check-docs-sync.sh --strict` on the upstream-divergent range. Push fails if (a) commits touch source without docs and lack a bypass marker, or (b) any commit declares `Closes: F-XX` for an unticked finding.
- **Editor sanity-check hook** (project-local config; outside the repo because agent / editor caches are gitignored — see `scripts/docs-snapshot.sh`): runs the snapshot script at session start so an editor or coding-agent session sees current release-gate progress, the WebUI status snapshot, recent ADRs, and uncommitted changes before deciding tasks.
- **Editor PostToolUse hook** (project-local config, gitignored): runs `scripts/check-docs-sync.sh` in warn mode after any `git commit`, `git push`, or `gh pr create`, surfacing reminders before a PR opens.

Bypass once with `LEFTHOOK_EXCLUDE=docs-sync git push ...` if you intentionally split docs into a follow-up PR. CI does not duplicate this gate today; the goal is to catch drift at push time.

## Testing Expectations

- Add or update tests close to changed code.
- Include edge cases and error paths.
- Keep tests deterministic and isolated.
- Use race detector in Go test runs where applicable.

## Security Expectations

- Never commit secrets.
- Keep auth, envelope, and pagination contracts stable unless spec/ADR updates are included.
- Run secret scanning and vulnerability checks required by CI.

## Review Guidance

Reviewers prioritize:

- Behavioral correctness
- Security impact
- Contract compatibility
- Spec/docs parity
- Test quality

## Code of Conduct

See `CODE_OF_CONDUCT.md`.

## Security Disclosure

See `SECURITY.md` for responsible disclosure.
