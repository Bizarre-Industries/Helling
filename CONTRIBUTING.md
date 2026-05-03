# Contributing to Helling

Thanks for your interest. This is a single-maintainer project right now, so contributions are welcome but the bar for merging is high.

## Before you write code

1. Read [docs/spec/architecture.md](docs/spec/architecture.md). If your change disagrees with the spec, the spec wins. Open an issue to discuss the spec change first.
2. Read [docs/standards/coding.md](docs/standards/coding.md) and [docs/standards/security.md](docs/standards/security.md).
3. For non-trivial changes, open an issue first describing what and why. Drive-by 500-line PRs without prior discussion will be closed.

## Setup

```bash
git clone https://github.com/Bizarre-Industries/helling.git
cd helling
make dev-setup
make generate
make check
```

If `make check` doesn't pass on a clean checkout, that's a bug — open an issue.

## Workflow

1. Branch from `main`. Branch name format: `<kind>/<short-slug>`, e.g. `feat/instance-snapshots`, `fix/login-rate-limit-bypass`.
2. Make focused commits. One logical change per commit. Conventional Commits format:
   - `feat:` new functionality
   - `fix:` bug fix
   - `docs:` docs only
   - `refactor:` no behavior change
   - `test:` test changes only
   - `ci:` CI/build changes
   - `chore:` housekeeping
   - `build:` build system / dependencies
3. Run `make check` and `make check-generated` before pushing.
4. Open a PR. Fill out the template. Link the issue.
5. CI must pass. Review must approve. Then it merges.

## API changes

The OpenAPI spec is the contract.

1. Edit `api/openapi.yaml`.
2. Run `make generate`. Review the generated diff.
3. Implement the new behavior in `apps/hellingd/internal/...`.
4. Add tests.
5. Update `docs/spec/architecture.md` if the change affects the architecture.
6. Commit the spec change, generated code, and implementation as separate commits where possible.

Breaking changes don't ship under `/v1`. They get a new `/v2` block in the spec, with `/v1` kept working until removed in a major release.

## Tests

- Unit tests for every service-layer function.
- No skipping tests. If a test is broken, fix it or delete it with justification.
- Race detector is mandatory.
- Integration tests go behind `//go:build integration` and run on demand. They can use a real Incus.

## Generated code

Don't edit it. Don't commit "small fixups" to generated files. Re-run `make generate` and let the tool produce them.

## Style

`gofumpt` + `goimports` + `golangci-lint` are the source of truth. CI runs them. Don't argue with the tools — change the config or change your code.

## Licensing

By submitting a PR, you agree your contribution is licensed under AGPL-3.0, the same as the rest of the project. No CLA. We follow the [Developer Certificate of Origin](https://developercertificate.org/) — sign off your commits with `git commit -s`.

## Documentation

- User-facing docs: `docs/install.md`, `docs/operations.md` (TBD)
- Internal specs: `docs/spec/`
- Standards: `docs/standards/`
- Per-feature design notes: `docs/design/<feature>.md`

If you change behavior, update the matching doc in the same PR. Out-of-date docs are worse than missing docs.

## Questions

Open a GitHub Discussion (when enabled) or an issue with the `question` label. Don't email for general questions — keep the conversation public so others benefit.

Security-sensitive issues: see [SECURITY.md](SECURITY.md).
