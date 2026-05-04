# Quality Assurance Standards

<!-- markdownlint-disable MD040 MD024 -->

Normative quality gates for every Helling artifact: OpenAPI contract, Go code, TypeScript code, markdown, YAML, shell, SQL, container images, dependencies.

Principle: **every artifact must pass its quality gate on every push. No gate is optional. No exceptions without a documented waiver.**

---

## 1. OpenAPI Contract Gate

### 1.1 Ruleset

Linter: `vacuum` (<https://quobix.com/vacuum>).
Ruleset: Helling custom ruleset at `api/.vacuum.yaml` (extends vacuum `recommended` baseline).
Version: vacuum ≥ 0.26.0 pinned in CI.

### 1.2 Score Policy

```text
Target score:      100 / 100
Minimum to merge:  100 / 100
CI gate:           fails if overallScore < 100
                   fails if categoryStatistics[*].score < 100
                   fails if any rule firing in 'error' severity
```

No waiver policy. If a rule genuinely doesn't apply, disable it in `api/.vacuum.yaml` with a comment explaining why. Exceptions live in the ruleset, not in PRs.

### 1.3 Custom rules Helling adds

Beyond vacuum's recommended ruleset, Helling enforces:

| Rule ID                              | What it checks                                                                                                  |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| `helling-property-snake-case`        | Property names must be snake_case (overrides camelCase default).                                                |
| `helling-ulid-id-path-param`         | Path parameters named `id` must use `pattern: '^[0-9A-HJKMNP-TV-Z]{26}$'`.                                      |
| `helling-required-example`           | Every request/response media type must have `example` or `examples`.                                            |
| `helling-required-description`       | Every `requestBody`, `parameter`, `schema`, and schema `property` must have `description`.                      |
| `helling-sensitive-writeonly`        | Properties named `password`, `secret`, `token`, `encrypted_*` must declare `writeOnly: true`.                   |
| `helling-error-envelope-on-mutation` | Every `POST`/`PUT`/`DELETE`/`PATCH` operation must declare 400, 401, 429 responses referencing `ErrorEnvelope`. |
| `helling-pagination-on-list`         | Any GET returning an array must include `PageLimit` and `PageCursor` parameters.                                |
| `helling-no-single-ref-combinator`   | Forbid `allOf: [$ref: X]` wrapping a single ref. Inline instead.                                                |
| `helling-operation-id-camelcase`     | `operationId` must match `^[a-z][a-zA-Z0-9]*$`.                                                                 |
| `helling-tag-declared`               | Every `tags: [X]` on an operation must reference a `tags[]` entry at the root.                                  |

See `api/.vacuum.yaml` for the normative ruleset.

### 1.4 CI command

```bash
vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml
```

Exit code non-zero fails the job.

### 1.5 Gate semantics after ADR-043 (2026-04-20)

With Huma as the contract source for Helling-owned endpoints, the generated OpenAPI artifact should auto-score 100/100 by construction for structural/doc coverage categories.

The OpenAPI gate now validates two classes of failure:

1. Struct-tag/doc-comment drift from intended API semantics.
2. Custom design-intent rules that protect Helling wire contracts.

Practical implication:

- Legacy hygiene rules such as `helling-required-description` become largely redundant because descriptions are enforced at the Go type/field layer before generation.
- Intent-oriented rules remain valuable regression detectors and must stay enabled.

Rules that remain high-value include:

- `writeOnly` enforcement for sensitive fields.
- ULID pattern enforcement for identifier fields and path params.
- Error envelope consistency for mutation and auth/error paths.
- Pagination contract enforcement for list endpoints.

Any gate failure is treated as a code/type contract regression, not a manual YAML formatting task.

### 1.6 Concrete before/after: `POST /api/v1/auth/login`

**Before (current):**

```yaml
/api/v1/auth/login:
  post:
    tags: [Auth]
    operationId: authLogin
    summary: PAM authenticate and issue JWT pair
    security: []
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/AuthLoginRequest"
    responses:
      "200":
        description: Login successful
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/EnvelopeAuthLoginResponse"
      "202":
        description: MFA challenge required
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/EnvelopeMfaChallengeResponse"
      "401":
        $ref: "#/components/responses/AuthError"
      "429":
        $ref: "#/components/responses/RateLimitError"
```

**After (100/100):**

```yaml
/api/v1/auth/login:
  post:
    tags: [Auth]
    operationId: authLogin
    summary: PAM authenticate and issue JWT pair
    description: |
      Authenticates a user via the host PAM stack and issues a short-lived
      access token plus a refresh cookie. If TOTP is enrolled for the user,
      the response is `202 Accepted` with an `mfa_token`; complete the flow
      via `POST /api/v1/auth/mfa/complete`.

      Rate limited to 5 attempts per 15 minutes per IP and per username.
    security: []
    requestBody:
      required: true
      description: PAM credentials, optionally with a TOTP code for single-shot login.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/AuthLoginRequest"
          examples:
            basic:
              summary: Password-only login
              value:
                username: alice
                password: correct-horse-battery-staple
            with_totp:
              summary: Password + inline TOTP (no MFA challenge round-trip)
              value:
                username: alice
                password: correct-horse-battery-staple
                totp_code: "492018"
    responses:
      "200":
        description: Login successful. Access token returned; refresh token set as `helling_refresh` cookie.
        headers:
          Set-Cookie:
            description: httpOnly, Secure, SameSite=Strict refresh cookie.
            schema:
              type: string
              example: >-
                helling_refresh=eyJ...; HttpOnly; Secure; SameSite=Strict;
                Path=/api/v1/auth; Max-Age=604800
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/EnvelopeAuthLoginResponse"
            example:
              data:
                access_token: eyJhbGciOiJFZERTQSIsImtpZCI6ImsxIn0...
                token_type: Bearer
                expires_in: 900
              meta:
                request_id: req_01JZABC0123456789ABCDEF
      "202":
        description: MFA challenge required. Complete via `/auth/mfa/complete`.
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/EnvelopeMfaChallengeResponse"
            example:
              data:
                mfa_required: true
                mfa_token: mfa_01JZABC...
              meta:
                request_id: req_01JZABC...
      "400":
        $ref: "#/components/responses/ValidationError"
      "401":
        $ref: "#/components/responses/AuthError"
      "429":
        $ref: "#/components/responses/RateLimitError"
```

Apply this shape to every operation. One weekend of disciplined work.

### 1.7 SchemaObject hardening

Every field in a request/response schema must carry constraints aligned with `docs/spec/validation.md`. Examples:

```yaml
User:
  type: object
  description: A Helling-managed user account, backed by a PAM identity.
  required: [id, username, role, status, created_at]
  properties:
    id:
      type: string
      description: ULID identifier.
      pattern: "^[0-9A-HJKMNP-TV-Z]{26}$"
      example: "01JZABC0123456789ABCDEF"
    username:
      type: string
      description: POSIX-compatible username. Matches `^[a-z_][a-z0-9_-]{0,31}$`.
      pattern: "^[a-z_][a-z0-9_-]{0,31}$"
      minLength: 1
      maxLength: 32
      example: alice
    role:
      type: string
      description: Fixed Helling role (ADR-032).
      enum: [admin, user, auditor]
      example: admin
    status:
      type: string
      description: Account activity state.
      enum: [active, disabled]
      example: active
    created_at:
      type: string
      description: Unix-epoch-aligned ISO-8601 timestamp in UTC.
      format: date-time
      example: "2026-04-20T10:00:00Z"

AuthLoginRequest:
  type: object
  description: PAM login credentials.
  required: [username, password]
  properties:
    username:
      type: string
      pattern: "^[a-z_][a-z0-9_-]{0,31}$"
      minLength: 1
      maxLength: 32
    password:
      type: string
      format: password
      minLength: 8
      maxLength: 1024
      writeOnly: true
    totp_code:
      type: string
      description: Optional TOTP code for single-shot MFA.
      pattern: "^[0-9]{6}$"
      minLength: 6
      maxLength: 6
```

---

## 2. Markdown Gate

Linter: `markdownlint-cli2` with `.markdownlint.yaml` at repo root.

```yaml
# .markdownlint.yaml baseline
default: true
MD013: false # line length (prose-wrap is noisy)
MD033: false # inline HTML allowed (needed for <!-- markdownlint-disable -->)
MD041: false # first-line-h1 (some files start with frontmatter)
MD040: true # fenced code blocks must have a language
MD024:
  siblings_only: true # heading duplicates allowed across sections
```

CI command: `markdownlint-cli2 '**/*.md' '#node_modules'`. Exit non-zero on any violation.

Additionally: `prettier --check '**/*.md'` for consistent markdown formatting (table alignment, list indentation, trailing whitespace).

Applies to: every `.md` file in the repo.

---

## 3. YAML Gate

Linter: `yamllint` with `.yamllint.yaml`.

```yaml
# .yamllint.yaml baseline
extends: default
rules:
  line-length:
    max: 200
    level: warning
  document-start: disable
  truthy:
    check-keys: false # 'on:' in GitHub Actions
  comments:
    min-spaces-from-content: 1
  indentation:
    spaces: 2
```

CI command: `yamllint -s .`. `-s` returns non-zero for warnings too (stricter than default).

Applies to: `.yaml`, `.yml` files except `api/openapi.yaml` (vacuum handles that).

---

## 4. Shell Script Gate

Linter: `shellcheck`.

```text
CI command: find . -type f \( -name '*.sh' -o -name '*.bash' \) \
  -not -path './node_modules/*' -not -path './.git/*' \
  -print0 | xargs -0 shellcheck -S style -e SC1091
```

`-S style` enables style-level warnings (strictest level below `info`).
`-e SC1091` suppresses the "sourced file not found" false positive for install-time sourcing.

Also: `shfmt -d -i 2 -ci` enforces consistent formatting. Exit non-zero on any diff.

Applies to: every shell script, including those embedded in `postinst`/`postrm` of the Debian package (lint those after extraction).

---

## 5. Go Backend Gate

Formatter: `gofmt` (stdlib) + `goimports`.
Linter: `golangci-lint` pinned to a specific version in `.golangci.yaml`.

```yaml
# .golangci.yaml baseline for Helling
run:
  timeout: 5m
  go: "1.26"

linters:
  enable:
    - asasalint # as and variadic mismatches
    - asciicheck # non-ASCII in idents
    - bidichk # bidirectional unicode tricks
    - bodyclose # http.Response.Body closed
    - contextcheck # ctx passed consistently
    - copyloopvar # Go 1.22+ loop variable semantics
    - dupl # duplicate code blocks
    - durationcheck # multiplication of time.Duration values
    - errcheck # unchecked errors
    - errname # error types named correctly
    - errorlint # errors.Is / errors.As usage
    - exhaustive # switch/map completeness for enums
    - forbidigo # forbid specific function calls
    - gci # import ordering
    - gocheckcompilerdirectives
    - gochecksumtype # sum-type completeness
    - goconst # repeated strings → const
    - gocritic # meta-linter, broad checks
    - gocyclo # cyclomatic complexity
    - gofmt
    - gofumpt # stricter gofmt
    - goimports
    - gosec # security checks (SQL injection, path traversal, file perms)
    - gosimple
    - govet
    - ineffassign
    - misspell
    - musttag # struct tags on types going through JSON/YAML
    - nakedret # named returns without explicit return
    - nilerr # nil err returned but not nil value
    - nilnil # nil, nil returns
    - noctx # http.Get without context
    - nolintlint # //nolint directives must have explanation
    - perfsprint # Sprintf → faster alternatives
    - prealloc # slice prealloc hints
    - predeclared # shadowing builtins
    - reassign # reassigning package-level vars
    - revive # replacement for deprecated golint
    - rowserrcheck # sql.Rows.Err() called
    - sloglint # slog key-value usage consistency
    - sqlclosecheck # sql.Rows closed
    - staticcheck
    - testifylint # testify assertion correctness
    - thelper # t.Helper() in helpers
    - tparallel # t.Parallel() correctness
    - unconvert
    - unparam
    - unused
    - usestdlibvars # prefer stdlib consts
    - wastedassign
    - whitespace

linters-settings:
  gocyclo:
    min-complexity: 15
  dupl:
    threshold: 150
  gosec:
    excludes: [] # no exclusions
  forbidigo:
    forbid:
      - p: '^fmt\.Print.*$'
        msg: "use slog for logging, not fmt"
      - p: '^log\.Print.*$'
        msg: "use slog for logging, not log package"
      - p: '^panic\('
        msg: "handle errors, don't panic"
      - p: '^os\.Exit\('
        msg: "return errors from main, don't exit from deep code"

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - dupl
        - gosec
        - gocyclo
  max-issues-per-linter: 0
  max-same-issues: 0
  new: false
```

### 5.1 Additional Go gates

| Check           | Command                        | Gate                       |
| --------------- | ------------------------------ | -------------------------- |
| `go vet`        | `go vet ./...`                 | exit 0                     |
| `go build`      | `go build ./...`               | exit 0                     |
| `go test -race` | `go test -race ./...`          | exit 0                     |
| `govulncheck`   | `govulncheck ./...`            | exit 0, zero HIGH+CRITICAL |
| `gofmt -l`      | `test -z "$(gofmt -l .)"`      | zero diff                  |
| `go mod tidy`   | verify diff after run is empty | zero diff                  |

### 5.2 Coverage gates

```yaml
Handlers: 80% line coverage minimum once handler/service/client package
  boundaries exist.
Services: 90% line coverage minimum once service packages exist.
Clients: 70% line coverage minimum once external client packages are mocked.
Overall: 80% target, with module-specific v0.1 ratchets enforced locally
  until the backend has enough executable surface for the target to be honest.
```

The local `task check:go` gate runs module-aware coverage profiles. v0.1 ratchets
`apps/hellingd` at 20% overall and `apps/helling-cli` at 80% overall; raise these
floors as tests land. CI uses `go test -coverprofile=cover.out ./...` plus
`go tool cover -func=cover.out` parsed by a small script that fails if any package
under `internal/handlers/`, `internal/services/`, `internal/clients/` is below its
threshold.

---

## 6. TypeScript / React Gate

Formatter: `biome` (replaces prettier + eslint for TS).
Target linter: `biome lint` with `biome.json` at `web/`.

```json
{
  "$schema": "https://biomejs.dev/schemas/1.9.0/schema.json",
  "organizeImports": { "enabled": true },
  "linter": {
    "enabled": true,
    "rules": {
      "recommended": true,
      "correctness": {
        "noUnusedVariables": "error",
        "useExhaustiveDependencies": "error"
      },
      "suspicious": { "noExplicitAny": "error", "noConsole": "warn" },
      "style": { "useConst": "error", "useTemplate": "error" },
      "a11y": { "recommended": true, "useButtonType": "error" },
      "security": { "noDangerouslySetInnerHtml": "error" }
    }
  },
  "formatter": {
    "enabled": true,
    "indentStyle": "space",
    "indentWidth": 2,
    "lineWidth": 100
  }
}
```

Additional gates:

- `tsc --noEmit` must pass.
- `hey-api/openapi-ts` generation must be idempotent: `bun run gen:api && git diff --exit-code web/src/api/generated`.
- `bun run test` must pass (Vitest).
- `bun run build` must pass.

The active v0.1 frontend merge gate is typecheck, generated-code drift, Vitest,
and production build. `bun run check` remains the cleanup target tracked by
WebUI audit F-08/F-09; promote it back into `task check:frontend` after the
legacy JSX and accessibility backlog is closed.

### 6.1 Strict TypeScript config

`tsconfig.json` must set:

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "exactOptionalPropertyTypes": true
  }
}
```

Most strict settings are on by default in `strict`, but `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes` must be explicit.

---

## 7. SQL / Schema Gate

Linter: `sqlfluff` with dialect `sqlite`.

```bash
# .sqlfluff baseline
[sqlfluff]
dialect = sqlite
templater = raw
max_line_length = 120

[sqlfluff:rules:aliasing.table]
aliasing = explicit

[sqlfluff:rules:capitalisation.keywords]
capitalisation_policy = upper
```

CI command: `sqlfluff lint db/schema.sql db/migrations/ db/queries/`.

Additional gate: `sqlc generate` must be idempotent. `sqlc generate && git diff --exit-code internal/db/queries/` must pass.

`goose` migrations must verify round-trip: CI spins up an empty SQLite, runs `goose up`, then `goose down`, then `goose up` again, expecting no errors. (Even if down migrations are best-effort for prod, they must execute cleanly in test.)

---

## 8. Dockerfile / Container Image Gate

Not used for Helling v0.1 production (ADR-021). Applies only to CI images and optional try-it images.

Linter: `hadolint`. Gate: zero errors at severity `info` or above.
Scanner: `grype` (see §9).

---

## 9. Security Scanning Gate (replaces security.md §4)

### 9.1 Current state (to be removed)

security.md §4 currently references:

- govulncheck (keep)
- gitleaks (keep)
- golangci-lint + gosec (keep, moved into §5)
- Grype (keep)
- Semgrep (remove — redundant with CodeQL)
- Bearer (remove — commercial license)
- osv-scanner (remove — redundant with govulncheck + Dependabot)
- Snyk Container (remove — commercial license)
- OpenSSF Scorecard (keep, weekly schedule unchanged)

### 9.2 New scanning matrix

| Tool              | Covers                                         | Frequency                                        | Gate                               |
| ----------------- | ---------------------------------------------- | ------------------------------------------------ | ---------------------------------- |
| CodeQL            | SAST for Go + JS/TS; security-extended queries | Every push                                       | Zero HIGH/CRITICAL alerts          |
| Grype             | Vulnerabilities in built artifacts and SBOMs   | Every push on release branches; weekly otherwise | Zero HIGH/CRITICAL in final images |
| govulncheck       | Go module vulnerabilities (symbol-aware)       | Every push                                       | Zero HIGH/CRITICAL                 |
| gitleaks          | Committed secrets                              | Every push + pre-commit hook                     | Zero findings                      |
| OpenSSF Scorecard | Project hygiene metrics                        | Weekly                                           | Score ≥ 7.0                        |

### 9.3 CodeQL workflow

Analyze Go and JavaScript. Enable security-extended + security-and-quality query packs.

```yaml
# .github/workflows/codeql.yml (pinned action SHAs per ADR-026)
name: CodeQL
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: "0 3 * * 1" # weekly Monday 03:00 UTC

permissions:
  actions: read
  contents: read
  security-events: write

jobs:
  analyze:
    runs-on: ubuntu-24.04
    strategy:
      fail-fast: false
      matrix:
        language: [go, javascript-typescript]
    steps:
      - uses: actions/checkout@<sha-pinned>
      - uses: github/codeql-action/init@<sha-pinned>
        with:
          languages: ${{ matrix.language }}
          queries: security-extended,security-and-quality
      - uses: github/codeql-action/autobuild@<sha-pinned>
      - uses: github/codeql-action/analyze@<sha-pinned>
        with:
          category: "/language:${{ matrix.language }}"
```

### 9.4 Grype workflow

Scan built artifacts (binaries, .deb, SBOMs). Replace Snyk Container entirely.

```yaml
# In release pipeline after Syft SBOM generation:
- name: Generate SBOM
  run: syft packages dir:dist/ -o cyclonedx-json > dist/sbom.cdx.json
- name: Scan SBOM
  run: grype sbom:dist/sbom.cdx.json --fail-on high
```

### 9.5 govulncheck workflow

Symbol-aware Go vuln scanner. Much better signal than osv-scanner for Go. Runs on every push.

```yaml
- uses: golang/govulncheck-action@<sha-pinned>
  with:
    go-version-input: "1.26"
    check-latest: true
```

### 9.6 gitleaks workflow

```yaml
- uses: gitleaks/gitleaks-action@<sha-pinned>
```

Plus local pre-commit via `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: <sha-pinned>
    hooks:
      - id: gitleaks
```

### 9.7 What this means for security.md

security.md §3 (Supply Chain Security) keeps its structure but replaces the `Scanning Pipeline` block:

```text
Every push:
  - CodeQL (security-extended + security-and-quality)
  - govulncheck
  - gitleaks
  - golangci-lint (with gosec)

Push to main / release branches:
  - Grype against built artifacts and SBOMs
  - All per-push scanners re-run on merge commit

Weekly:
  - OpenSSF Scorecard
```

Remove all references to Semgrep, Bearer, Snyk, osv-scanner. Update `coding.md` line 437 dependency list accordingly.

---

## 10. Documentation Gate

### 10.1 Link checker

`lychee` with `lychee.toml`.

```toml
# lychee.toml
exclude = [
  "https://bizarre.industries/docs/errors/.*",   # docs not yet deployed
  "https://localhost.*",
]
max_concurrency = 8
timeout = 20
```

CI: `lychee --offline docs/ --no-progress` on every push (external checks weekly).

### 10.2 Spelling

`typos` (Rust-based, fast).

```toml
# typos.toml
[default]
extend-ignore-re = [
  "hellingd", "hellingprox", "helling_.*",
  "incusd", "incus-.*",
  "\\$ref",
]
```

CI: `typos` must exit zero.

### 10.3 API-CLI-WebUI parity

`phase0-parity-matrix.md` enforcement is currently manual. Add a CI script that compares:

- Every `operationId` in `api/openapi.yaml`
- Every command in `docs/spec/cli.md` (parsed from the fenced code blocks)
- Every route in `docs/spec/webui-spec.md` (parsed from the `/foo` headings)

Fail if an operation lacks either a CLI command or a WebUI route, unless listed in `docs/roadmap/phase0-parity-exceptions.yaml`:

```yaml
# docs/roadmap/phase0-parity-exceptions.yaml
exceptions:
  - operation_id: eventsSse
    missing: cli
    reason: SSE stream follow command deferred to v0.2
    owner: suhail
    target: v0.2.0
  - operation_id: healthGet
    missing: cli
    reason: Not exposed to end users (infra-only)
    owner: suhail
    target: never
```

Script at `scripts/check-parity.sh`. Runs in CI. Fails on unjustified drift.

---

## 11. Quality CI Workflow

Single workflow runs all gates in parallel. File: `.github/workflows/quality.yml`.

```yaml
name: Quality
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read
  pull-requests: read
  security-events: write

jobs:
  openapi:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - name: Install vacuum
        run: curl -fsSL https://quobix.com/scripts/install_vacuum.sh | sh
      - name: Lint OpenAPI (must score 100)
        run: |
          vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml
          score=$(vacuum report --stdout api/openapi.yaml | jq '.statistics.overallScore')
          echo "Score: $score"
          if [ "$score" != "100" ]; then
            echo "FAIL: openapi.yaml score $score/100, required 100"
            exit 1
          fi

  markdown:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - uses: DavidAnson/markdownlint-cli2-action@<sha-pinned>
        with: { globs: "**/*.md" }
      - run: npx prettier --check '**/*.md'

  yaml:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - run: pip install yamllint && yamllint -s .

  shell:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - run: sudo apt install -y shellcheck shfmt
      - run: shellcheck -S style $(find . -name '*.sh' -not -path './.git/*')
      - run: shfmt -d -i 2 -ci $(find . -name '*.sh' -not -path './.git/*')

  go:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - uses: actions/setup-go@<sha-pinned>
        with: { go-version: "1.26" }
      - run: go build ./...
      - run: go vet ./...
      - run: go test -race -coverprofile=cover.out ./...
      - run: bash scripts/check-coverage.sh cover.out
      - uses: golangci/golangci-lint-action@<sha-pinned>
        with: { version: latest }
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...
      - run: |
          go mod tidy
          git diff --exit-code go.mod go.sum

  frontend:
    runs-on: ubuntu-24.04
    defaults: { run: { working-directory: web } }
    steps:
      - uses: actions/checkout@<sha-pinned>
      - uses: oven-sh/setup-bun@<sha-pinned>
      - run: bun install --frozen-lockfile
      - run: bun run biome check .
      - run: bun run tsc --noEmit
      - run: bun run gen:api && git diff --exit-code src/api/generated
      - run: bun test --coverage

  sql:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - run: pip install sqlfluff
      - run: sqlfluff lint db/schema.sql db/migrations/ db/queries/
      - run: |
          go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
          sqlc generate
          git diff --exit-code internal/db/queries

  secrets:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
        with: { fetch-depth: 0 }
      - uses: gitleaks/gitleaks-action@<sha-pinned>

  docs:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha-pinned>
      - uses: lycheeverse/lychee-action@<sha-pinned>
        with: { args: "--offline docs/" }
      - uses: crate-ci/typos@<sha-pinned>
      - run: bash scripts/check-parity.sh
```

CodeQL and Grype are separate workflows (`.github/workflows/codeql.yml`, `.github/workflows/security.yml`) because they have different cadence and permission profiles.

Branch protection (for main):

- Require status checks: `openapi`, `markdown`, `yaml`, `shell`, `go`, `frontend`, `sql`, `secrets`, `docs`, `CodeQL`.
- Require all conversations resolved.
- Require linear history (no merge commits).
- Require signed commits (already per security.md).
- Require 1 review (self-review on solo until collaborators join).

---

## 12. Pre-commit Local Gate

`.pre-commit-config.yaml` runs a fast subset locally:

```yaml
repos:
  - repo: local
    hooks:
      - id: vacuum
        name: vacuum lint openapi
        entry: vacuum lint --ruleset api/.vacuum.yaml --fail-severity info api/openapi.yaml
        language: system
        files: ^api/openapi\.yaml$
      - id: golangci-lint
        name: golangci-lint
        entry: golangci-lint run --new-from-rev=HEAD~1
        language: system
        files: \.go$
      - id: gofmt
        name: gofmt
        entry: gofmt -l
        language: system
        files: \.go$
      - id: shellcheck
        name: shellcheck
        entry: shellcheck -S style
        language: system
        files: \.(sh|bash)$
  - repo: https://github.com/gitleaks/gitleaks
    rev: <sha-pinned>
    hooks:
      - id: gitleaks
  - repo: https://github.com/crate-ci/typos
    rev: <sha-pinned>
    hooks:
      - id: typos
```

Mandatory install for contributors. Documented in CONTRIBUTING.md.

---

## 13. Waiver Policy

No rule may be silenced in-place (no `//nolint:gosec`, no `<!-- markdownlint-disable -->` on single lines) without an inline comment explaining why. Example:

```go
//nolint:gosec // G304: path is validated against /var/lib/helling prefix upstream
f, err := os.Open(path)
```

Global rule disables must live in the tool's config file with a comment. `api/.vacuum.yaml` can disable individual rules globally; inline suppression is not a vacuum feature anyway.

For failing-in-CI exceptions, create a waiver file at `docs/waivers/<YYYY-MM-DD>-<slug>.md` describing:

- What is waived.
- Why.
- When it will be removed.
- Owner.

Waiver file presence is tracked; waivers older than 90 days fail CI on the main branch (a scheduled job checks `git log` on waivers).

---

## 14. Quality Score Dashboard

Every CI run writes quality scores to `docs/quality/latest.json`:

```json
{
  "run_id": "17823458",
  "commit": "201a2c7",
  "timestamp": "2026-04-20T22:00:00Z",
  "openapi_score": 100,
  "openapi_warnings": 0,
  "go_coverage": {
    "handlers": 83,
    "services": 91,
    "clients": 74,
    "overall": 82
  },
  "frontend_coverage": { "components": 65, "hooks": 83, "utils": 92 },
  "security": {
    "codeql_high": 0,
    "grype_high": 0,
    "govulncheck_high": 0,
    "gitleaks": 0
  },
  "parity_gaps": 0
}
```

GitHub Pages renders this at <https://bizarre-industries.github.io/Helling/quality>.

Regression → CI fails.
Improvement → CI succeeds, value recorded.
30-day trend graph auto-generated.

---

## 15. Enforcement Timeline

v0.1.0 release gates (enforced now, in this order):

| Week        | Gate activated                         |
| ----------- | -------------------------------------- |
| This week   | vacuum score ≥ 100 on api/openapi.yaml |
| This week   | markdownlint + prettier on all .md     |
| This week   | yamllint strict on all yaml            |
| This week   | shellcheck + shfmt                     |
| Next        | gitleaks + govulncheck + CodeQL        |
| Next        | golangci-lint with full linter set     |
| Before v0.1 | Coverage gates                         |
| Before v0.1 | Parity matrix automated check          |
| Before v0.1 | Grype on built artifacts               |
| Before v0.2 | Waiver expiry enforcement              |

Before this doc merges, `api/openapi.yaml` must already pass at 100/100. Otherwise the doc is aspirational, not normative.

---

## 16. Waiver Examples (current state)

None approved. The 33/100 openapi score is a pre-standards baseline, not a waiver. Must be fixed before this standard merges.

---

## 17. Related Documents

- `docs/standards/coding.md` — language-level rules (updates needed per §5).
- `docs/standards/security.md` — security posture (updates needed per §9).
- `docs/standards/compliance.md` — release tier gates (this doc adds to Tier 1).
- `docs/standards/infrastructure.md` — ops and deployment (no change).
- `docs/roadmap/phase0-parity-matrix.md` — source of parity exceptions.
- `api/.vacuum.yaml` — OpenAPI ruleset.
- `.github/workflows/quality.yml` — CI gate implementation.
- `ADR-042` — Security scanning consolidation (see adrs directory).
