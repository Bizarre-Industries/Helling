# AGENTS.md

## Project Overview

Helling is a monorepo for a Debian-first virtualization platform with:

- Go backend daemon: hellingd
- Go CLI: helling
- Go TLS/static proxy: helling-proxy
- React + Vite Web UI in web
- Shared API contract in api/openapi.yaml

Architecture direction and constraints are documented in:

- docs/spec/\*
- docs/design/\*
- docs/standards/\*
- docs/roadmap/\*

If implementation behavior appears to conflict with docs, treat the docs as the source of truth and align code to docs.

## Monorepo Layout

- apps/hellingd: main backend service
- apps/helling-cli: CLI client
- apps/helling-proxy: TLS reverse proxy and static web serving
- web: frontend (React 19, TypeScript, Vite, Ant Design)
- api/openapi.yaml: API contract used for code generation
- deploy: Dockerfile and service packaging artifacts

Go workspace is declared in go.work and currently includes all three Go apps.

## Required Toolchain

- Go 1.26
- Bun (for frontend and Orval generation)
- oapi-codegen (Go API code generation)
- golangci-lint, gofumpt, goimports

Recommended bootstrap:

```bash
make dev-setup
```

This installs required tools, frontend deps, and git hooks.

## Setup Commands

Clone and bootstrap:

```bash
git clone https://github.com/Bizarre-Industries/helling.git
cd helling
make dev-setup
```

Manual frontend setup (if needed):

```bash
cd web
bun install
```

## Code Generation (Mandatory)

The OpenAPI file is the contract. Generated code must stay in sync.

Generate all artifacts:

```bash
make generate
```

Verify generated artifacts are committed and clean:

```bash
make check-generated
```

Generated paths include:

- apps/hellingd/api/\*.gen.go
- apps/helling-cli/internal/client/\*.gen.go
- web/src/api/generated/

Do not hand-edit generated files.

## Development Workflow

Typical loop:

```bash
make generate
make fmt
make lint
make test
```

Full CI-like local gate:

```bash
make check
```

Build all Go binaries:

```bash
make build
```

Run frontend dev server:

```bash
make web-dev
```

Build frontend:

```bash
make web-build
```

Build container image:

```bash
make docker
```

Clean outputs:

```bash
make clean
```

## Testing Instructions

Run all Go tests:

```bash
make test
```

This executes:

- go test -tags devauth ./apps/hellingd/... -race -count=1
- go test ./apps/helling-cli/... -race -count=1
- go test ./apps/helling-proxy/... -race -count=1

Run package-specific tests:

```bash
go test ./apps/hellingd/...
go test ./apps/helling-cli/...
go test ./apps/helling-proxy/...
```

Integration/E2E coverage in CI is defined in .github/workflows/integration.yml and exercises key API flows against a built Docker image.

When changing behavior, add or update tests close to the changed code.

## Linting and Formatting

Format all code:

```bash
make fmt
```

Check formatting without modifying files:

```bash
make fmt-check
```

Run lint:

```bash
make lint
```

Note: Current lint target focuses on apps/hellingd via golangci-lint configuration.

## Code Style and Architecture Rules

Follow standards in docs/standards/coding.md and docs/standards/security.md.

Important rules:

- Keep Incus/Podman resource operations thin and proxy-oriented where intended by architecture docs.
- Prefer service-layer access patterns over ad hoc database access.
- Use structured logging patterns.
- Validate input at API boundaries.
- Keep errors contextual and non-leaky.

Frontend rules:

- Use generated API clients/hooks from OpenAPI artifacts.
- Prefer existing component and page patterns under web/src.
- Keep TypeScript strictness and lint cleanliness.

## Security Checks

Fast security checks:

```bash
make security-fast
```

Full security checks:

```bash
make security
```

Security workflow references:

- .github/workflows/security.yml
- SECURITY.md

Never commit secrets. gitleaks is part of local and CI posture.

## Build and Release

CI pipeline definitions:

- .github/workflows/ci.yml
- .github/workflows/integration.yml
- .github/workflows/release.yml

Release flow is tag-driven (v\* tags), runs verification jobs, then publishes binaries/images.

Before opening PRs or tagging releases, ensure:

```bash
make check
make check-generated
```

## Pull Request and Commit Guidelines

Follow CONTRIBUTING.md.

- Use conventional commits:
  - feat:, fix:, docs:, test:, refactor:, ci:, chore:, build:
- Run formatting, lint, and tests before pushing.
- Keep generated artifacts updated when API contract changes.

## Agent-Specific Guidance

When working as an automated coding agent in this repo:

1. Read relevant docs/spec/design/standards/roadmap files before changing behavior.
2. If behavior is not documented, avoid guessing and flag uncertainty in your summary.
3. For API changes, update api/openapi.yaml first, then regenerate via make generate.
4. Prefer minimal diffs and avoid unrelated refactors.
5. Validate with focused commands first, then run broader checks.

Suggested pre-merge command set:

```bash
make generate
make check-generated
make fmt-check
make lint
make test
```

## Troubleshooting

Common issues:

- Generated code drift:
  - Run make generate, then re-run make check-generated.
- Tooling missing locally:
  - Run make dev-setup.
- Frontend dependency issues:
  - Run bun install in web.
- CGO-related backend build/test issues:
  - Ensure local toolchain supports CGO for sqlite/pam dependent paths.

## Additional References

- README.md
- CONTRIBUTING.md
- lessons.md
- docs/spec/architecture.md
- docs/standards/coding.md
- docs/standards/security.md
