# Coding Standards

<!-- markdownlint-disable MD040 -->

Rules for writing code in Helling. Not preferences. Standards. Violations are bugs.

---

## 1. Go Backend

### File Structure (per handler/service)

```
internal/
  incus/
    incus.go          # Client wrapper, connection management
    instances.go      # Instance CRUD operations
    instances_test.go # Tests adjacent to code
    storage.go        # Storage operations
    storage_test.go
    errors.go         # Domain-specific error types
```

No `utils/`, `helpers/`, `common/`, `misc/`. If a function doesn't have a home, the package structure is wrong.

### Error Handling

```go
// RULE: Every error wraps with context. No naked returns.
// BAD:
return err

// GOOD:
return fmt.Errorf("create instance %q: %w", name, err)

// RULE: Errors at API boundary are translated to structured responses.
// Internal errors NEVER leak to the client.
// BAD:
http.Error(w, err.Error(), 500)

// GOOD:
writeError(w, http.StatusBadGateway, &APIError{
    Error:   "Failed to create instance",
    Code:    "INCUS_ERROR",
    Action:  "Check that incusd is running: systemctl status incusd",
    DocLink: "https://bizarre.industries/docs/troubleshooting#incus",
})

// RULE: Use sentinel errors for expected conditions. Wrap for unexpected.
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrDependencyExists = errors.New("resource has dependents")
    ErrQuotaExceeded = errors.New("quota exceeded")
)

// RULE: Check errors immediately. No error accumulation patterns.
// RULE: Use errors.Is() and errors.As() for comparison, not == or type assertion.

// RULE: Errors MUST NOT contain secrets, credentials, tokens, or PII.
// Redact before wrapping:
return fmt.Errorf("auth failed for user %q from %s", username, sourceIP)
// NEVER: return fmt.Errorf("auth failed: password %q invalid", password)
```

### Shell-Out Conventions (ADR-018)

```
For host operations (nft, smartctl, systemctl, lvs, zpool), shell out to CLI tools:
- Always use exec.CommandContext with a timeout
- Parse JSON output where available (nft --json, smartctl --json, lvs --reportformat json)
- Wrap errors with context: fmt.Errorf("firewall.ListRules: %w", err)
- Tools are guaranteed present (shipped in ISO)
```

### systemd Timer Conventions (ADR-017)

```
Scheduled operations write .timer + .service unit files to /etc/systemd/system/:
- Timer names: helling-<type>-<resource>.timer
- Service names: helling-<type>-<resource>.service
- Always call systemctl daemon-reload after writing units
- Use Persistent=true to catch up on missed runs
- Use RandomizedDelaySec to prevent thundering herd
```

### Logging

```go
// RULE: All logging via slog. No fmt.Println, no log.Println.
// RULE: Structured fields, not string formatting.

// BAD:
log.Printf("Instance %s created by %s", name, user)

// GOOD:
slog.Info("instance created",
    "instance", name,
    "type", instanceType,
    "user", user,
    "source_ip", sourceIP,
    "duration_ms", elapsed.Milliseconds(),
)

// RULE: Log levels have meaning:
//   Debug: internal state, only for development
//   Info:  normal operations (instance created, backup completed)
//   Warn:  recoverable issues (retry, degraded mode, approaching limit)
//   Error: failures that need attention (API error, connection lost)

// RULE: Never log at Error for expected conditions (404 is not an error).
// RULE: Include request_id in all request-scoped logs.
// RULE: Never log passwords, tokens, keys, certificates, or PII.
// RULE: Sensitive fields use a Redactor:
slog.Info("config loaded", "db_password", slog.String("REDACTED"))
```

### Context

```go
// RULE: context.Context is always the first parameter.
func (s *InstanceService) Create(ctx context.Context, req CreateInstanceRequest) (*Instance, error)

// RULE: Pass context through entire call chain. Never store in struct.
// RULE: Use context for cancellation, deadlines, request-scoped values.
// RULE: Set timeouts on all external calls:
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

// RULE: Check context cancellation before expensive operations:
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}
```

### Concurrency

```go
// RULE: Use errgroup for parallel operations with error handling.
g, ctx := errgroup.WithContext(ctx)
for _, instance := range instances {
    inst := instance
    g.Go(func() error {
        return s.StartInstance(ctx, inst.Name)
    })
}
if err := g.Wait(); err != nil {
    return fmt.Errorf("bulk start: %w", err)
}

// RULE: Never start goroutines without a termination path.
// RULE: Use channels for communication, mutexes for state protection.
// RULE: Run tests with -race flag. Zero race conditions allowed.
// RULE: Use sync.Once for one-time initialization, not manual flags.
```

### Input Validation

```go
// RULE: Validate ALL input at the API handler layer.
// RULE: Whitelist valid values, don't blacklist bad ones.
// RULE: Validate before any processing or database access.

func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
    var req CreateInstanceRequest
    if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
        writeError(w, 400, "INVALID_JSON", "Request body is not valid JSON")
        return
    }

    // Validate name: alphanumeric + hyphens, 1-63 chars
    if !isValidName(req.Name) {
        writeError(w, 400, "INVALID_NAME", "Name must be 1-63 alphanumeric characters or hyphens")
        return
    }

    // Validate enum values against whitelist
    if !slices.Contains(validInstanceTypes, req.Type) {
        writeError(w, 400, "INVALID_TYPE", "Type must be one of: vm, container")
        return
    }

    // Validate numeric ranges
    if req.CPU < 1 || req.CPU > 128 {
        writeError(w, 400, "INVALID_CPU", "CPU must be between 1 and 128")
        return
    }
}

// RULE: Use io.LimitReader on all request bodies (prevent DoS).
// RULE: Sanitize file paths (prevent path traversal):
if strings.Contains(path, "..") || !filepath.IsAbs(path) {
    return ErrInvalidPath
}

// RULE: Use parameterized queries for ALL database access.
// NEVER: fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", name)
// ALWAYS: query with placeholders through sqlc-generated methods/database/sql
```

### Database

```go
// RULE: All database access through sqlc-generated queries + database/sql.
// RULE: Schema changes via numbered migration files, never ad-hoc ALTER TABLE.
// RULE: Transactions for multi-step operations:
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

if err := qtx.CreateInstance(ctx, params); err != nil {
    return err
}
if err := qtx.CreateAuditEntry(ctx, auditParams); err != nil {
    return err
}
if err := tx.Commit(); err != nil {
    return err
}

// RULE: Pagination on all list endpoints. Never return unbounded results.
// RULE: Indexes on all columns used in WHERE clauses.
```

### HTTP Security Headers

```go
// RULE: Set on every response via middleware:
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "0")  // Disabled, CSP handles this
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
        w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
        w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
        next.ServeHTTP(w, r)
    })
}
```

### Handler Architecture (ADR-014)

```
Most resources are proxied to Incus/Podman sockets (ADR-014). Only Helling-specific
endpoints (approximately 40) have handler implementations. These include: auth, users, settings,
tasks, warnings, webhooks, audit, and metrics.

Proxied requests:
        - hellingd validates JWT, loads user Incus TLS identity, forwards to local Incus HTTPS API (mTLS)
        - Podman requests forward to the local Podman Unix socket
  - No handler code needed for standard Incus/Podman CRUD operations
  - Auth + audit middleware still runs on proxied requests

Helling-specific handlers:
    - Implement approximately 40 Helling-specific handlers. All Incus/Podman operations go through the proxy (ADR-014).
  - Call service layer, never access DB directly
  - Follow the patterns below for API design, validation, and error handling
```

### API Design

```go
// RULE: All endpoints follow consistent naming:
//   GET    /api/v1/{resource}          → List (paginated)
//   POST   /api/v1/{resource}          → Create
//   GET    /api/v1/{resource}/{id}     → Get one
//   PUT    /api/v1/{resource}/{id}     → Full update
//   PATCH  /api/v1/{resource}/{id}     → Partial update
//   DELETE /api/v1/{resource}/{id}     → Delete

// RULE: Consistent response envelope:
// Success: { "data": {...}, "meta": { "request_id": "...", "page": { "has_next": true, "next_cursor": "...", "limit": 50 } } }
// Error:   { "error": "msg", "code": "CODE", "action": "...", "doc_link": "..." }

// RULE: Pagination via query params: ?limit=50&cursor=opaque&sort=created_at&order=desc
// RULE: Filtering via query params: ?status=running&type=vm&tags=production
// RULE: Offset/page pagination is not supported for v0.1 API contracts.

// RULE: Rate limiting on all endpoints:
//   Auth endpoints: 5 failed attempts per 15 minutes per IP + username
//   Write endpoints: 30 req/min per user
//   Read endpoints: 120 req/min per user

// RULE: Request IDs on all requests (X-Request-ID header, generated if missing).
// RULE: Request ID propagated to all logs, responses, and downstream calls.
```

---

## 2. React Frontend

### Component Rules

```tsx
// RULE: Components in src/components/. Pages in src/pages/. Hooks in src/hooks/.
// RULE: One component per file. File name matches component name.
// RULE: No business logic in components. Use hooks or services.

// RULE: TypeScript strict mode. No `any` type.
// BAD:
const data: any = response.data;
// GOOD:
const data: Instance[] = response.data;

// RULE: All API types generated from OpenAPI spec. Don't hand-type API interfaces.
// RULE: Use React Query for ALL API calls. No raw fetch in components.
// RULE: Use antd components for ALL UI. No custom HTML for things antd provides.

// RULE: Dynamic imports for heavy components:
const VncConsole = React.lazy(() => import("./VncConsole"));
const CodeEditor = React.lazy(() => import("./CodeEditor"));

// RULE: No localStorage/sessionStorage for auth tokens. Use httpOnly cookies or
// memory-only storage with refresh token flow.

// RULE: No inline styles. Use antd theme tokens or Tailwind utilities.
// RULE: No `!important` in CSS.
```

### State Management

```tsx
// RULE: React Query for server state. useState for UI state. No Redux.
// RULE: Cache invalidation via React Query + SSE events.

// Pattern: SSE invalidates React Query cache
useEffect(() => {
  const es = new EventSource("/api/v1/events");
  es.onmessage = (e) => {
    const event = JSON.parse(e.data);
    if (event.type === "instance.state_changed") {
      queryClient.invalidateQueries({ queryKey: ["instances"] });
    }
  };
  return () => es.close();
}, []);
```

### Security (Frontend)

```tsx
// RULE: Never render user input as HTML (XSS). React escapes by default.
// RULE: Never use dangerouslySetInnerHTML except for sanitized markdown.
// RULE: Validate all form inputs client-side AND server-side.
// RULE: CSRF protection via SameSite cookies + custom header.
// RULE: No secrets in frontend code. API keys stay in backend.
```

---

## 3. Testing Standards

### Coverage Targets

```
Go backend:
  Handlers:  80% line coverage minimum
  Services:  90% line coverage minimum
  Clients:   70% line coverage minimum (external dependencies mocked)
  Overall:   80% minimum, 90% goal

React frontend:
  Components: 60% minimum (test interactions, not rendering)
  Hooks:      80% minimum
  Utils:      90% minimum
```

### Test Patterns

```go
// RULE: Table-driven tests for all functions with multiple cases.
func TestValidateName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid simple", "my-vm", false},
        {"valid with numbers", "vm-123", false},
        {"empty", "", true},
        {"too long", strings.Repeat("a", 64), true},
        {"special chars", "my_vm!", true},
        {"starts with hyphen", "-invalid", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
            }
        })
    }
}

// RULE: Use testify/assert for assertions.
// RULE: Use testify/mock or manual mocks for external dependencies.
// RULE: Tests must not depend on network, filesystem, or external services.
// RULE: Integration tests in separate _integration_test.go files with build tag.
// RULE: Fuzz tests for all input parsing functions:
func FuzzValidateName(f *testing.F) {
    f.Add("valid-name")
    f.Add("")
    f.Add("../../../etc/passwd")
    f.Fuzz(func(t *testing.T, name string) {
        _ = ValidateName(name) // Must not panic
    })
}

// RULE: Run with -race flag in CI. Zero race conditions allowed.
// RULE: Benchmark critical path functions:
func BenchmarkListInstances(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _, _ = service.ListInstances(ctx, ListOptions{})
    }
}
```

### Test Organization

```
Unit tests:     Adjacent to code (*_test.go). Run with `make test`.
Integration:    test/ directory. Require running services. Run with `make test-integration`.
E2E:            test/e2e/. Full API tests against running hellingd. Run in CI only.
Fuzz:           In *_test.go files. Run with `go test -fuzz`.
Benchmarks:     In *_test.go files. Run with `go test -bench`.
Load tests:     test/load/. k6 or hey scripts. Run manually.
```

---

## 4. Code Review Checklist

Every PR must pass all items:

```
Correctness:
  [ ] Does what the ticket/issue says
  [ ] Edge cases handled (empty input, max values, concurrent access)
  [ ] Error paths tested

Security:
  [ ] Input validated (whitelist, not blacklist)
  [ ] No secrets in code, logs, or error messages
  [ ] SQL injection impossible (parameterized queries)
  [ ] Path traversal impossible (no unsanitized file paths)
  [ ] Auth check on every new endpoint
  [ ] Rate limiting considered

Quality:
  [ ] Errors wrapped with context (%w)
  [ ] Structured logging (slog, not fmt)
  [ ] Context passed through call chain
  [ ] No goroutine leaks (termination path exists)
  [ ] Tests for new code (table-driven, edge cases)
  [ ] No TODO/FIXME without linked issue

Standards:
  [ ] SPDX header on new files
  [ ] Conventional commit message
  [ ] Signed-off-by line (DCO)
  [ ] golangci-lint clean
  [ ] make check passes
```

---

## 5. Dependency Management

```
RULE: Every new dependency requires justification in PR description.
RULE: Prefer stdlib over third-party. Justify why stdlib is insufficient.
RULE: Pin exact versions. No floating (^, ~, latest).
RULE: Run `go mod tidy` and `go mod verify` in CI.
RULE: Dependabot configured for automated patch updates.
RULE: Manual review required for minor/major updates.
RULE: License check in CI (no AGPL-incompatible licenses).
RULE: govulncheck in CI. Zero known vulnerabilities at release time.

Approved dependencies (no justification needed):
  bmc-toolbox/bmclib        — BMC management (core functionality)
  net/http (stdlib)         — HTTP routing baseline
  sqlc + database/sql       — Typed query generation + persistence
  goose                     — SQL migrations
  pquerna/otp          — TOTP 2FA
  go-webauthn/webauthn — WebAuthn
  prometheus/client    — Metrics
  golang-jwt/jwt       — JWT
  spf13/cobra          — CLI framework
  gopkg.in/yaml.v3     — Configuration
  stretchr/testify     — Test assertions

Requires justification:
  Any new dependency not in the above list.
  Especially: anything that pulls in CGO, anything >5MB, anything with <100 GitHub stars.
```
