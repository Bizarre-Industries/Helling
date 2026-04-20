# Local Development Workflow Specification

<!-- markdownlint-disable MD029 -->

Normative workflow for local Helling development. Defines the tools, directory structure, daily commands, and quality gates that run on a developer's machine.

Implementation lives in `Taskfile.yaml`, `Makefile`, `lefthook.yml`, and `scripts/*`. Any divergence between this document and those files is a bug.

## Scope

In scope:

- Tooling install and first-time setup
- Daily command surface (`task check`, `task fmt`, `task gen`, etc.)
- IDE configuration recommendations
- Local database and service bootstrap
- Troubleshooting common failures

Out of scope:

- CI pipeline behavior (see `docs/spec/ci.md`)
- Git hook behavior (see `docs/spec/pre-commit.md`)
- Lima VM setup (see `docs/standards/development-environment.md`)

## Supported host platforms

- Debian 13 (native — no VM required)
- Ubuntu 24.04+ (native)
- macOS with Lima (per ADR-034)
- Linux distributions with `apt`-equivalent package managers (manual install)

Windows is not a supported development host.

## First-time setup

Order of operations:

### 1. Clone

```bash
git clone git@github.com:Bizarre-Industries/Helling.git
cd Helling
```

### 2. Install Go 1.26+

```bash
# Debian/Ubuntu
curl -fsSL https://go.dev/dl/go1.26.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
```

### 3. Install tooling

```text
bash scripts/install-tools.sh
```

This installs, in order:

- Go tools: `task`, `lefthook`, `golangci-lint`, `govulncheck`, `gofumpt`, `goimports`, `shfmt`, `gitleaks`, `oapi-codegen`, `sqlc`, `goose`, `vacuum`
- System linters (via apt/brew): `shellcheck`, `yamllint`
- Rust tools: `typos`, `lychee`
- Python tools: `sqlfluff`
- Node tools: `markdownlint-cli2`, `prettier`
- Frontend: `bun` + `web/` dependencies
- Security: `grype`

The script is idempotent. Re-run any time a new tool is added to `task check`.

### 4. Install git hooks

```bash
task hooks
```

This runs `lefthook install` which writes `.git/hooks/{pre-commit,pre-push,commit-msg}` dispatchers.

### 5. Verify

```bash
task check
```

Expected: all checks pass (on a clean checkout). If something fails on the first run, it's either a tooling install issue (fix via `scripts/install-tools.sh`) or a repo drift issue (file an issue).

### 6. Lima (macOS / non-Linux only)

Per ADR-034. See `docs/standards/development-environment.md`.

## Daily command surface

| Command      | What it does                                 | When to run                               |
| ------------ | -------------------------------------------- | ----------------------------------------- |
| `task`       | List all tasks                               | When you forget the command               |
| `task check` | Run every quality gate                       | Before every `git push`                   |
| `task fmt`   | Auto-format everything fixable               | After making changes, before staging      |
| `task gen`   | Regenerate openapi.yaml + downstream clients | After changing Huma operation definitions |
| `task test`  | Run Go + frontend tests                      | When iterating on implementation          |
| `task build` | Build all binaries into `bin/`               | Before manual smoke testing               |
| `task clean` | Remove build artifacts and coverage          | Before a fresh run                        |

Sub-tasks for targeted runs:

| Command                  | What it does                                           |
| ------------------------ | ------------------------------------------------------ |
| `task check:go`          | Go-only gates (build, vet, lint, vuln, test, coverage) |
| `task check:openapi`     | OpenAPI lint + score ≥100 gate                         |
| `task check:frontend`    | Frontend-only gates                                    |
| `task openapi:report`    | Show vacuum category breakdown                         |
| `task openapi:dashboard` | Interactive vacuum TUI                                 |

Makefile equivalents exist for the ADR-043 mental model:

- `make generate` → `task gen`
- `make check-generated` → `task check:openapi:generated`
- `make check` → `task check`

## Editor / IDE configuration

The repo includes `.editorconfig` with project-wide whitespace rules (tabs for Go, spaces for everything else). Ensure your editor respects it.

Recommended editor extensions:

| Editor           | Extensions                                                                                                                                                            |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| VS Code          | `golang.go`, `biomejs.biome`, `editorconfig.editorconfig`, `davidanson.vscode-markdownlint`, `tamasfe.even-better-toml`, `redhat.vscode-yaml`, `timonwong.shellcheck` |
| Neovim           | `gopls`, `biome-lsp` via `nvim-lspconfig`; `null-ls` or `conform.nvim` for formatting                                                                                 |
| JetBrains GoLand | Built-in support for most; add Biome plugin, Markdown plugin                                                                                                          |

VS Code workspace settings (not committed — per-developer preference):

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.formatTool": "custom",
  "go.alternateTools": { "customFormatter": "gofumpt" },
  "editor.formatOnSave": true,
  "[typescript]": { "editor.defaultFormatter": "biomejs.biome" },
  "[typescriptreact]": { "editor.defaultFormatter": "biomejs.biome" }
}
```

## Directory conventions

```text
Helling/
├── apps/
│   ├── hellingd/               # main daemon
│   │   ├── cmd/hellingd/       # entry point
│   │   └── internal/
│   │       └── api/            # Huma operation registrations (per-tag files)
│   ├── helling-cli/            # CLI binary
│   └── helling-proxy/          # edge proxy daemon (ADR-027)
├── api/
│   ├── openapi.yaml            # GENERATED by Huma — do not hand-edit
│   └── .vacuum.yaml            # OpenAPI linting rules
├── db/
│   ├── schema.sql
│   ├── migrations/             # goose migrations
│   └── queries/                # sqlc input
├── internal/
│   ├── db/queries/             # sqlc output (generated)
│   └── ...
├── tools/
│   └── openapi-dump/           # spec-from-Huma generator
├── web/                        # React + antd WebUI
│   ├── src/
│   │   └── api/generated/      # orval/hey-api output (generated)
│   └── biome.json
├── scripts/
│   ├── install-tools.sh
│   ├── check-coverage.sh
│   └── check-parity.sh
└── docs/                       # (existing)
```

## Local services

Helling integrates with Incus, Podman, and a local SQLite database. For development:

| Service | Local bringup                                                | Port / Socket                                              |
| ------- | ------------------------------------------------------------ | ---------------------------------------------------------- |
| Incus   | `sudo incus admin init` (once), `sudo systemctl start incus` | `127.0.0.1:8443` (HTTPS loopback, ADR-036)                 |
| Podman  | `systemctl --user start podman.socket`                       | `/run/user/$UID/podman/podman.sock`                        |
| SQLite  | Auto-created by goose                                        | `/var/lib/helling/helling.db` (prod) or `./dev.db` (local) |
| Caddy   | Skip for dev; `hellingd` listens directly                    | N/A in dev                                                 |

Bootstrap a local database:

```bash
export GOOSE_DRIVER=sqlite3
export GOOSE_DBSTRING=./dev.db
goose -dir db/migrations up
```

Teardown:

```bash
rm dev.db
```

## Common failures and fixes

### `task check` fails on `openapi` job

```yaml
FAIL: openapi score 33/100 below minimum 100/100
```

Means the committed `api/openapi.yaml` doesn't match the Helling vacuum ruleset. If post-Huma, run `task gen:openapi` to regenerate; the score will be 100 by construction if your Go struct tags are correctly populated.

### `task check` fails on `openapi-generated` job

```yaml
FAIL: committed api/openapi.yaml differs from Huma-generated output
```

Someone hand-edited `api/openapi.yaml` (forbidden), or a Go type change wasn't followed by regeneration. Fix:

```bash
task gen:openapi
git add api/openapi.yaml
git commit --amend --no-edit
```

### `task check` fails on `frontend-generated` job

Same class of problem for the frontend client:

```bash
cd web
bun run gen:api
cd ..
git add web/src/api/generated
git commit --amend --no-edit
```

### `task check` fails on `go` → `coverage`

```text
✗ internal/handlers — 74.2% below floor 80%
```

Add tests or refactor. Coverage floors are enforced per-package. See `docs/standards/quality-assurance.md` §5.2.

### `task check` fails on `secrets`

```yaml
gitleaks: finding in <file>
```

Do not commit. If it's a false positive (e.g., a test fixture), update `.gitleaksignore` with justification. Never bypass.

### `task check` is slow (>3 min)

Check cache hits. Go module cache should be populated after the first run. If consistently slow, see `docs/spec/ci.md` caching policy.

## What `task check` does NOT catch

Quality gates check static properties of the code and docs. They do not verify:

- Runtime behavior of handlers (unit tests do)
- Integration with real Incus/Podman (manual smoke testing does)
- WebUI rendering (Playwright tests, if added, would)
- Performance characteristics (benchmark runs do)

Always run a manual smoke test before submitting a PR for anything beyond docs changes.

## Related documents

- `docs/spec/ci.md` — what this workflow mirrors in CI
- `docs/spec/pre-commit.md` — what runs automatically on commit and push
- `docs/standards/development-environment.md` — Lima bootstrap
- `docs/standards/quality-assurance.md` — normative quality gates
- `CONTRIBUTING.md` — PR workflow, DCO, commit conventions
