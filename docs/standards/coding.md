# Coding Standards

Last reviewed: 2026-05-02

These rules apply to all Go code in this repository. Frontend rules are at the bottom.

## Layout

- `apps/<binary>/cmd/<binary>/main.go` — entrypoint only. Wiring, no logic.
- `apps/<binary>/internal/...` — unexported, application-specific packages.
- `apps/<binary>/api/` — generated OpenAPI types and server stubs. Hand-edits forbidden.
- Shared code across binaries goes in `libs/helling-<name>/` with its own `go.mod` (post-v0.1).

## Module discipline

- Module path: `github.com/Bizarre-Industries/helling/apps/<binary>` for now.
- Each binary is a separate Go module.
- `go.work` ties them together for local development.
- Pin `go 1.26.0` and `toolchain go1.26.2` (or current minor) explicitly.

## Architectural patterns

### Service layer

API handlers do **not** call the store, the Incus client, or any other infrastructure directly. They go through a service interface.

```go
// internal/server/handlers.go
func (s *Server) ListInstances(w http.ResponseWriter, r *http.Request, params api.ListInstancesParams) {
    instances, err := s.svc.ListInstances(r.Context(), userFromCtx(r.Context()), params)
    // ...
}

// internal/service/instances.go
type InstanceService interface {
    ListInstances(ctx context.Context, u *User, params api.ListInstancesParams) ([]api.Instance, error)
    // ...
}
```

The service interface exists so handlers are testable without spinning up Incus or SQLite.

### Storage access

- Always go through the `store` package. Never `db.Exec` outside it.
- Every store function takes `context.Context` first.
- Use `database/sql` directly with prepared statements. No ORM in v0.1.
- Read-only paths use `sql.DB.QueryContext`. Write paths that need atomicity use `sql.DB.BeginTx`.

### Incus access

- Always through `internal/incus`. Never call `lxc/incus` packages from handlers.
- The Incus client is a single shared instance. Do not reconnect per request.
- Wrap Incus operations in our own `Operation` type. Don't surface Incus's IDs or error shapes to API consumers.

### Error handling

- Wrap with `fmt.Errorf("doing X: %w", err)`. The verb form, with context, lower case, no period.
- Sentinel errors live in the package they belong to: `var ErrNotFound = errors.New("not found")`.
- API handlers translate domain errors to HTTP via a single `respondError(w, err)` helper.
- Never leak internal error text to API responses. Map to a stable `code` and a sanitized `message`.

```go
// good
if err != nil {
    return fmt.Errorf("loading instance %q: %w", name, err)
}

// bad — loses cause
if err != nil {
    return errors.New("could not load instance")
}
```

### Logging

- `log/slog` only. No `log`, no `logrus`, no `zap`.
- JSON handler in production, text in dev. Selected by `HELLING_LOG_FORMAT` env var.
- Mandatory fields on every request log: `request_id`, `method`, `path`, `status`, `duration_ms`.
- User-identifiable fields go on authenticated requests only: `user_id`, `username`. Never log passwords, session tokens, or password hashes.
- Use the request-scoped logger from context: `slog.FromContext(ctx)`. Don't use `slog.Default()` inside request handlers.

### Validation

- Input validation happens at the API boundary. Public JSON handlers enforce the constraints declared in `api/openapi.yaml`; generated types alone are not enough.
- Domain-level invariants (e.g. "instance name unique per project") live in the service layer, not the handler.

### Concurrency

- Goroutines that outlive a single request must be tracked. The daemon has a single `errgroup.Group` rooted in `main`; all background workers live there.
- A handler that spawns a goroutine for fire-and-forget work passes a derived context bound to the daemon's lifecycle, never the request context (which is cancelled on response).
- No `time.Sleep` in production code paths. Use `time.NewTicker` or `context.WithDeadline`.

### Configuration

- `config.Config` struct loaded once at startup. Fields are exported, JSON-tagged for YAML loading via `yaml.v3`.
- Config sources, in order of precedence: CLI flags → env vars (`HELLING_*` prefix) → `/etc/helling/helling.yaml` → defaults.
- No global state. Pass `Config` (or relevant subset) into the constructors that need it.

## Style

### Naming

- Exported function names follow stdlib conventions: `ListInstances`, not `list_instances` or `listInstances`.
- Acronyms are uppercase consistently: `HTTPClient`, `URLParser`, `UserID`. Both first and trailing.
- Avoid stuttering: `instance.Instance` is fine, `instance.InstanceList` is not — call it `instance.List`.
- Test functions use `Test_<func>_<case>`: `TestLogin_BadPassword`, `TestLogin_RateLimited`.

### Comments

- Every exported identifier has a doc comment starting with its name: `// ListInstances returns ...`.
- Comments explain _why_, not _what_. The code shows _what_.
- TODO comments include an owner and a date: `// TODO(suhail, 2026-05-02): ...`.

### Imports

`goimports` ordering, with a third group for our own packages:

```go
import (
    "context"
    "fmt"

    "github.com/go-chi/chi/v5"
    "github.com/lxc/incus/v6/client"

    "github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)
```

Configure with `goimports -local github.com/Bizarre-Industries/helling`.

### Files

- One concept per file. `instances.go`, `sessions.go`, `users.go` — not a 2,000-line `handlers.go`.
- Tests next to the file under test: `instances_test.go`. Black-box tests use `package <name>_test`.

## Testing

- Every public service method has a test. No exceptions.
- Use the standard library `testing` package. No `testify` unless we hit a real ergonomics wall on assertions.
- Table-driven tests where the cases share structure. Each case has a `name`.
- Race detector always on: `-race -count=1`. Ban `-count > 1` in CI (it hides flakes by averaging them out).
- Integration tests go behind a build tag: `//go:build integration`. They can spin up Incus or use a fake.
- Fakes for the Incus client live in `internal/incus/fake`. Real client in production, fake in tests. No mocks-as-frameworks.

## Generated code

- Do not edit. Ever.
- Re-run `make generate` after every spec change. CI enforces this via `make check-generated`.
- Generated files have a `// Code generated by ...; DO NOT EDIT.` line. golangci-lint skips them.

## Frontend

- TypeScript strict mode (`"strict": true` in tsconfig).
- Generated API client and React Query hooks come from `web/src/api/generated/`. Same forbidden-edit rule.
- Components live in `web/src/components/`. Pages in `web/src/pages/`.
- antd v6 is the component library. No mixing in another UI kit.
- No global state libraries (Redux, Zustand) in v0.1. React Query handles server state; component state stays local.
- Form handling via antd's `Form`, validation rules co-located with the form definition.
- Style with antd's theme tokens and CSS modules. No Tailwind in v0.1. No `styled-components` either.

## What this doc doesn't say

If a rule isn't here, default to:

1. What the Go standard library does.
2. What `gofumpt` enforces.
3. What `golangci-lint` flags.

If you find yourself fighting one of those three, raise it as a discussion before writing the workaround.
