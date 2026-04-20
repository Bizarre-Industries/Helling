# Pre-commit / Pre-push Hook Specification

<!-- markdownlint-disable MD029 -->

Normative specification of Helling's git hooks. Defines what runs on each git event, in what order, how to skip, and what the auto-fix behavior guarantees.

Implementation lives in `lefthook.yml` at the repo root. Divergence between this document and that file is a bug.

## Why hooks

Two purposes, served by two different hook stages:

1. **Pre-commit** — cheap, staged-files-only tripwires. Block obvious mistakes (committed secrets, unformatted files) before they enter the git log. Runs in ~5-10 seconds.
2. **Pre-push** — full CI-equivalent gate. Catch everything that CI would fail on before burning action minutes. Runs in ~60-120 seconds.

A third hook enforces commit message conventions.

## Toolchain

Hooks are managed by **lefthook** — a single Go binary that reads `lefthook.yml`. No Python/Ruby runtime required. Installation happens via `task hooks`:

```bash
task hooks   # runs `lefthook install`
```

This writes `.git/hooks/pre-commit`, `.git/hooks/pre-push`, and `.git/hooks/commit-msg` as lefthook dispatchers. `.git/` is not committed; re-run `task hooks` after cloning or after `.git/hooks/` is wiped.

## Pre-commit

Triggered on `git commit`. Parallel execution. Scoped to staged files only.

### Hooks run

| Hook           | Scope                                      | Behavior on failure       | Auto-fixes? |
| -------------- | ------------------------------------------ | ------------------------- | ----------- |
| `gitleaks`     | staged files                               | Abort commit              | No          |
| `gofmt`        | `*.go`                                     | Format file, re-stage     | Yes         |
| `goimports`    | `*.go`                                     | Reorder imports, re-stage | Yes         |
| `markdownlint` | `*.md`                                     | Abort commit              | No          |
| `prettier-md`  | `*.md`                                     | Format file, re-stage     | Yes         |
| `yamllint`     | `*.{yaml,yml}` (except `api/openapi.yaml`) | Abort commit              | No          |
| `shellcheck`   | `*.{sh,bash}`                              | Abort commit              | No          |
| `shfmt`        | `*.{sh,bash}`                              | Format file, re-stage     | Yes         |
| `vacuum`       | `api/openapi.yaml` only                    | Abort commit              | No          |
| `typos`        | `*.md`                                     | Abort commit              | No          |

### Auto-fix contract

Hooks marked "Yes" under Auto-fixes rewrite the file in place and `git add` it back. The net effect: the commit succeeds with formatted content, the developer does not need to re-run the command. If you stage intentionally-unformatted content, those hooks will silently fix it; verify with `git diff --cached` before pushing if that matters.

Hooks marked "No" never rewrite files. They only report.

### Scoping

All pre-commit hooks operate on `{staged_files}` — only the files included in the current commit. A commit touching one `.go` file does not trigger markdown or yaml checks. This keeps pre-commit under 10 seconds in the typical case.

### Skipping

Skip all hooks once:

```text
LEFTHOOK=0 git commit ...
```

Skip specific hooks:

```text
LEFTHOOK_EXCLUDE=gitleaks,typos git commit ...
```

Skip via flag (discouraged):

```bash
git commit --no-verify
```

Skipping is always possible. The backstop is CI, which runs the equivalent gates unconditionally.

## Pre-push

Triggered on `git push`. Sequential execution (for readable failure output).

### What runs

```bash
task check
```

That's it. The pre-push hook is a single-command shell that invokes the full local gate suite:

- OpenAPI lint (score ≥100)
- Markdown / YAML / shell lint
- Go build, vet, lint, vuln, tidy, race-tests, coverage
- Frontend lint + type-check + generated-code drift + tests
- SQL lint + sqlc drift + goose round-trip
- Secrets, spelling, links, parity

Expected runtime: 60-120 seconds on a warm cache. First run after tool installs takes longer.

### Skipping

```bash
git push --no-verify
```

You will get flagged by CI. Use only for branches explicitly intended as in-progress snapshots (e.g., work-in-progress PR drafts that you know will fail and are asking for feedback on).

### Rationale for pre-push vs pre-commit

The division is deliberate. Pre-commit stays fast so it doesn't interrupt flow on every checkpoint commit. Pre-push catches the full gate before network I/O and CI runners are consumed.

If a hook belongs to neither stage well, it belongs in pre-push. Default to correctness over speed at the push boundary.

## Commit-msg

Triggered after the commit message is written, before the commit is created.

### Conventional Commits enforcement

The subject line must match:

```text
^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert|spec|adr)(\(.+\))?!?: .{1,}
```

Where:

- **type** is one of: `feat`, `fix`, `chore`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `revert`, `spec`, `adr`
- **scope** is optional, in parens, e.g. `(auth)`, `(handlers)`, `(webui)`
- **`!`** marks a breaking change
- **subject** is at least one character, by convention under 72 chars

Valid examples:

```text
feat(auth): add TOTP verification endpoint
fix: handle nil response in webhook delivery
docs(spec): document rate limit semantics
refactor!: rename UserCreate → userCreate for camelCase consistency
spec(api): add /events SSE endpoint
adr: accept ADR-044 hey-api for frontend codegen
```

Invalid examples (will be rejected):

```text
updated auth code            ← no type
feat added auth              ← missing colon
Fix(auth): better errors     ← type must be lowercase
```

### Length warnings

Subject lines longer than 72 characters print a warning but do not abort. Long subjects are discouraged but not forbidden; sometimes a specific commit genuinely needs 80 characters.

### Rationale for `spec` and `adr` types

Conventional Commits standardizes a type list that doesn't name documentation types distinctly. Helling adds two:

- `spec(...)` — changes to `docs/spec/*.md` files
- `adr(...)` — changes to `docs/decisions/*.md` files

These categories show up frequently enough in a docs-heavy project to warrant their own namespace. Changelog-generators that expect strict Conventional Commits should be configured to map these as equivalent to `docs`.

## What the hooks do NOT do

- Do not run integration tests (too slow for hook context).
- Do not run Grype or CodeQL (too slow; belong in CI).
- Do not enforce coverage floors at commit time (only at push time, via `task check`).
- Do not enforce API-CLI-WebUI parity at commit time (only at push time).
- Do not enforce sign-off / DCO. That's repo policy enforced by CI and review, not hooks.

## Troubleshooting

### Hook didn't run

Re-install: `task hooks`. Lefthook may have been wiped by `git clone --recursive` or `.git/hooks/` corruption.

### Hook ran but I don't see the auto-fixed file in my commit

The hook auto-fixed and re-staged the file. Check `git diff HEAD~1` after the commit to verify what actually landed. If the commit didn't include the fix, the hook failed silently — check `LEFTHOOK_VERBOSE=1 git commit ...`.

### Pre-commit is slow (>30 seconds)

One hook is out of scope. Run `LEFTHOOK_VERBOSE=1 git commit ...` and find the slow one. Likely culprits: goimports on a very large staged change, or vacuum on `api/openapi.yaml` that needs caching.

### Pre-push is slow (>3 minutes)

That's `task check`. See `docs/spec/local-dev.md` "Common failures" section. Typical cause: cold Go build cache.

### I want to add a new hook

Edit `lefthook.yml`. The pattern:

```yaml
pre-commit:
  commands:
    my-new-hook:
      glob: "*.ext" # which files
      run: my-linter {staged_files}
      tags: category fast # grouping labels
```

Any new hook in pre-commit must complete in under 5 seconds for the typical case. If it's slower, put it in pre-push.

## Related documents

- `docs/spec/ci.md` — CI workflows that run the same gates unconditionally
- `docs/spec/local-dev.md` — daily workflow and command surface
- `docs/standards/quality-assurance.md` — quality gates these hooks enforce locally
- `CONTRIBUTING.md` — commit message policy, DCO, review workflow
