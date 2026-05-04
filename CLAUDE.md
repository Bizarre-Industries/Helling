# CLAUDE.md

> Operating manual for AI coding agents working in this repo.
> Last reviewed: 2026-05-02.

## Project overview

Helling is a self-hosted, single-node, Debian-first system container and VM management platform built on top of Incus and Podman.

The repository contains:

- **Go backend daemon:** `apps/hellingd/` — listens on a Unix socket, talks to Incus.
- **Go CLI:** `apps/helling-cli/` — interactive client over the same socket.
- **Go TLS/static proxy:** `apps/helling-proxy/` — terminates TLS, serves the web bundle, forwards API.
- **React + Vite Web UI:** `web/` — React 19 + TypeScript + antd v6.
- **Shared API contract:** `api/openapi.yaml` — single source of truth for the API.

Architecture, design constraints, standards, and roadmap live in:

- `docs/spec/` — source-of-truth specs
- `docs/design/` — design notes per feature
- `docs/standards/` — coding and security standards
- `docs/roadmap/` — versioned roadmaps

If implementation behavior conflicts with the docs, **the docs are the source of truth**. Update the code to match, or update the docs in the same PR.

## Monorepo layout

```text
apps/hellingd        backend daemon
apps/helling-cli     CLI client
apps/helling-proxy   TLS reverse proxy + static web serving
web/                 frontend (React 19, TypeScript, Vite, antd 6)
api/openapi.yaml     API contract used for code generation
deploy/              Dockerfile and packaging artifacts
```

The Go workspace is declared in `go.work` and includes the three Go apps. Each app is a separate module.

## Required toolchain

- Go 1.26 (toolchain 1.26.2 pinned)
- Bun (frontend tooling and Orval generation)
- `oapi-codegen` v2 (managed as a Go tool dependency in `apps/hellingd/go.mod`)
- `golangci-lint`, `gofumpt`, `goimports`

Bootstrap with:

```bash
make dev-setup
```

This installs Go tools, frontend dependencies (when `web/` exists), and the pre-commit git hook.

## Setup commands

```bash
git clone https://github.com/Bizarre-Industries/helling.git
cd helling
make dev-setup
```

Manual frontend setup (when `web/` is scaffolded):

```bash
cd web && bun install
```

## Code generation (mandatory)

`api/openapi.yaml` is the contract. Generated code must stay in sync with it.

```bash
make generate          # regenerate Go server, Go client, and TS client
make check-generated   # fail if working tree drifts from spec
```

Generated paths:

- `apps/hellingd/api/*.gen.go` — server stubs and models
- `apps/helling-cli/internal/client/*.gen.go` — Go client (when scaffolded)
- `web/src/api/generated/` — TypeScript client and React Query hooks

**Do not hand-edit generated files.** Behavior changes go through the spec.

## Development workflow

Typical loop:

```bash
make generate
make fmt
make lint
make test
```

Full CI-equivalent local gate:

```bash
make check
```

Build all Go binaries to `./bin/`:

```bash
make build
```

Frontend dev server:

```bash
make web-dev
```

Frontend production build:

```bash
make web-build
```

Container image:

```bash
make docker
```

Clean outputs:

```bash
make clean
```

## Testing

Run all Go tests:

```bash
make test
```

This runs each module with `-race -count=1`. The `hellingd` test target uses build tag `devauth` (gates an in-memory auth backend used in tests).

Package-specific:

```bash
go test ./apps/hellingd/...
go test ./apps/helling-cli/...
go test ./apps/helling-proxy/...
```

Integration / E2E tests live behind the `integration` build tag and run on demand or via dedicated CI workflows. They may require a real Incus instance.

When changing behavior, add or update tests near the changed code.

## Linting and formatting

```bash
make fmt        # format in place
make fmt-check  # verify formatting (CI gate)
make lint       # golangci-lint across all Go modules
```

## Code style and architecture rules

Authoritative reference: `docs/standards/coding.md` and `docs/standards/security.md`. Highlights:

- Handlers stay thin; business logic lives in service interfaces.
- Storage access goes through `internal/store` only.
- Incus interaction goes through `internal/incus` only.
- Errors wrap with `fmt.Errorf("doing X: %w", err)`.
- Logging is `log/slog` only — JSON in production, text in dev.
- No CGO unless explicitly justified. The default SQLite driver is `modernc.org/sqlite`.

Frontend rules:

- Use generated API client and React Query hooks from `web/src/api/generated/`.
- antd v6 is the only UI component library.
- TypeScript strict mode mandatory.

## Security checks

```bash
make security-fast   # gitleaks + govulncheck
make security        # adds gosec via golangci-lint
```

Workflows:

- `.github/workflows/ci.yml` — formatting, lint, generate-drift, tests, govulncheck
- `SECURITY.md` — disclosure policy

Never commit secrets. The pre-commit hook from `make dev-setup` runs `make fmt-check lint`.

## Build and release

CI workflow definitions live in `.github/workflows/`. Release flow is tag-driven (`v*` tags) and runs verification before publishing binaries and images.

Before opening a PR or tagging a release:

```bash
make check
make check-generated
```

## Pull requests and commits

See `CONTRIBUTING.md`.

- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `ci:`, `chore:`, `build:`.
- Format, lint, and test before pushing.
- Keep generated artifacts in sync when the API contract changes.
- Sign commits per DCO (`git commit -s`).

## Agent-specific guidance

When working as an automated coding agent in this repo:

1. Read the relevant docs under `docs/spec/`, `docs/design/`, `docs/standards/`, and `docs/roadmap/` before changing behavior.
2. If behavior is undefined, **don't guess**. Flag the gap in your summary and ask.
3. For API changes: update `api/openapi.yaml` first, then run `make generate`. Only then implement.
4. Prefer minimal diffs. Avoid unrelated refactors in the same PR.
5. Validate locally with focused commands (`go test ./<changed-pkg>/...`) before running `make check`.
6. If a doc reference is broken, fix the doc in the same PR.

Suggested pre-merge command set:

```bash
make generate
make check-generated
make fmt-check
make lint
make test
```

## Agent tooling (Claude Code)

This repo ships project-shared agent tooling under `.claude/`. Per-contributor
overrides go in `.claude/settings.local.json` (gitignored).

### Slash commands (`.claude/commands/`)

- `/check` — `make check && make check-generated` (full pre-merge gate)
- `/regen` — `make generate` then verify drift
- `/ship [remote] [refspec]` — fmt + check + check-generated + security-fast, then push
- `/snapshot` — print docs/plans/checklist snapshot (also fires on SessionStart)
- `/replan-mark <tag>` — advance `.claude/.last-shipped-tag` after a next-version plan is rewritten

### Skills (`.claude/skills/`)

Auto-loaded at session start. Mix of project-specific and ECC-installed:

- **`openapi-workflow`** (project) — spec-first contract; never hand-edit
  generated code
- `golang-patterns`, `golang-testing`, `bun`, `frontend-design`,
  `accessibility`, `seo` (ECC) — language and surface guidance
- `auto-skill`, `skill-discovery` (ECC) — meta tooling for adding more skills

The ECC autoskills tool installs to `.agents/skills/` (gitignored). To deploy
into Claude Code's load path:

```bash
cp -RL .agents/skills/. .claude/skills/
```

### Hooks (`.claude/hooks/`)

Wired in `.claude/settings.json`:

- **`replan-on-tag.sh`** — SessionStart. Detects new `v*` tags pushed since the
  last marker; scaffolds `docs/plans/v<NEXT>-plan.md` and tells the agent to
  rewrite + execute it.
- **`never-guess.sh`** — PreToolUse on `Bash` and `WebFetch`. Injects a
  verification-reminder additionalContext blob when the proposed action invokes
  an external tool whose CLI surface may have drifted (incus, podman, gh,
  golangci-lint, oapi-codegen, openapi-ts, vacuum, lefthook, bun, task, cosign,
  dch, dput, helling binaries) or matches a high-risk shell pattern (`rm -rf`,
  `--force`, `git reset --hard`).
- **`mark-replanned.sh <tag>`** — manual. Run after rewriting the next-version
  plan and landing the first execution-step commit. Use `/replan-mark <tag>`.

Also wired:

- `scripts/docs-snapshot.sh` (SessionStart + UserPromptSubmit on plan/decision
  prompts) — emits the release-gate snapshot, recent ADRs, working-tree
  dirtiness.
- `scripts/claude-hooks/post-bash-version-shipped.sh` (PostToolUse on Bash) —
  detects `git push --tags` of a `v*` tag and triggers `task plan:next-version`.

### Council subagents (`.claude/agents/`)

Spawned in parallel for changes matching the trigger list (new ADR, new
external dep, breaking API/schema, deletion >100 LOC, edits under
`apps/hellingd/internal/auth/`, `api/openapi.yaml`, signing config, or
`.github/workflows/release.yml`):

- `council-architect`, `council-security`, `council-perf-skeptic`,
  `council-devil-advocate`, `council-ux-critic` (UI changes only)
- `self-critique` — fires before commits that don't trigger the full council
- `mechanical` — Haiku-backed, for renames/format/regex-replace ops

See `.claude/agents/README.md` for deliberation flow and trigger details.

## Troubleshooting

**Generated code drift:**

```bash
make generate && make check-generated
```

**Tooling missing locally:**

```bash
make dev-setup
```

**Frontend dependency issues:**

```bash
cd web && bun install
```

**SQLite issues:**
The default driver is pure-Go `modernc.org/sqlite`, no CGO required. If you see CGO-related errors, you've imported `github.com/mattn/go-sqlite3` somewhere — remove it.

## References

- `README.md` — user-facing project intro
- `CONTRIBUTING.md` — contribution workflow
- `lessons.md` — lessons learned, append-only
- `docs/spec/architecture.md` — canonical architecture
- `docs/standards/coding.md` — code style and architecture rules
- `docs/standards/security.md` — security model and constraints
- `docs/roadmap/v0.1.md` — current milestone
