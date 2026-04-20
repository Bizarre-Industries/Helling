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
