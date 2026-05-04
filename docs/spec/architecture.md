# Helling Architecture (v0.1)

> Status: **draft, source-of-truth**. Code that disagrees with this doc is wrong, not the doc.
> Maintainer: @binGhzal · Last reviewed: 2026-05-02

## 1. What Helling is

A self-hosted, single-node, Debian-first system container and VM management platform built on top of Incus and Podman, with a unified REST API and a web dashboard.

Out of scope for v0.1:

- Multi-node clustering
- Orchestrating across hosts
- Built-in image registry
- Tenant isolation beyond OS-level Incus projects
- Anything Kubernetes-shaped

This is a deliberate scope cut from the earlier "300+ endpoint platform" framing. The v0.1 surface is what proves the architecture. Everything else is post-v0.1.

## 2. Components

```text
┌────────────────────────────────────────────────────────────┐
│                       Browser / CLI                         │
└──────────────────────┬─────────────────────────────────────┘
                       │ HTTPS
                       ▼
┌────────────────────────────────────────────────────────────┐
│  Caddy edge service (TLS + static web + reverse proxy)      │
└──────────────────────┬─────────────────────────────────────┘
                       │ Unix socket (/run/helling/api.sock)
                       ▼
┌────────────────────────────────────────────────────────────┐
│                       hellingd                              │
│                                                             │
│  ┌────────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │  Router    │→ │ Service  │→ │  Store   │  │  Incus   │  │
│  │  (chi)     │  │  Layer   │  │ (SQLite) │  │  Client  │  │
│  └────────────┘  └────┬─────┘  └──────────┘  └────┬─────┘  │
│                       │                            │        │
└───────────────────────┼────────────────────────────┼────────┘
                        │                            │
                        ▼                            ▼
                ┌──────────────┐           ┌──────────────────┐
                │  helling.db  │           │  /var/lib/incus  │
                │  (SQLite)    │           │  (unix socket)   │
                └──────────────┘           └──────────────────┘
```

### 2.1 hellingd

The backend daemon. Owns: HTTP routing, request validation, business logic, persistence, Incus interaction. Listens on a Unix socket only. Never exposed directly to the network.

### 2.2 Caddy edge service

TLS terminator and static asset server. Reads the API socket from `hellingd` and proxies to it. Serves the React bundle from disk. Runs as an unprivileged user. Listens on `:8006` in v0.1.

The reason this is a separate process: `hellingd` requires membership in the `incus` group. Running TLS handling in that same process expands the attack surface unnecessarily. Splitting it lets the public-facing process stay unprivileged.

### 2.3 helling-cli

Local CLI client. Talks to `hellingd` over the same Unix socket using the same OpenAPI contract. v0.1 surface: enough subcommands to drive the v0.1 endpoints (login, container list/create/start/stop/delete).

### 2.4 web

React 19 + TypeScript + Vite. SPA. Calls the API through Caddy. Dashboard shell + container list + container detail in v0.1. Generated TypeScript client from `api/openapi.yaml` via `@hey-api/openapi-ts`.

## 3. Request lifecycle (v0.1)

A typical authenticated request:

1. Browser → Caddy on `:8006` over TLS.
2. Caddy strips TLS and forwards to `hellingd` over `/run/helling/api.sock`.
3. `hellingd`'s chi router matches the path.
4. Middleware chain: request ID → structured logger → recoverer → timeout → auth.
5. Auth middleware accepts either the session cookie (HTTP-only, `Secure`, `SameSite=Lax`) or a bearer token. Session cookies are looked up by hashed token in `sessions`; API tokens are looked up by hashed token in `api_tokens`; access JWTs are verified with the persisted Ed25519 signing key. 401 if invalid/expired.
6. Handler invokes the service layer.
7. Service layer reads/writes the SQLite store and/or calls the Incus client over its own Unix socket.
8. Response shaped per OpenAPI spec, JSON-encoded, returned.

Async Incus operations (instance create, start, stop) return a Helling operation ID immediately. The service polls the Incus operation in a background goroutine, mirrors state into the `operations` table, and the client polls `GET /v1/operations/{id}`.

## 4. Data model (v0.1)

SQLite, single file at `$HELLING_STATE_DIR/helling.db` (default `/var/lib/helling`).

```sql
CREATE TABLE users (
    id           INTEGER PRIMARY KEY,
    username     TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,    -- argon2id, encoded
    created_at   INTEGER NOT NULL,  -- unix epoch
    is_admin     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,  -- 32-byte random, base64url
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE TABLE operations (
    id           TEXT PRIMARY KEY,  -- uuid v7
    user_id      INTEGER NOT NULL REFERENCES users(id),
    kind         TEXT NOT NULL,     -- e.g. "instance.create"
    target       TEXT NOT NULL,     -- e.g. instance name
    incus_op_id  TEXT,              -- mirror of Incus operation id when applicable
    status       TEXT NOT NULL,     -- pending|running|success|failure|cancelled
    error        TEXT,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    metadata     BLOB                -- json
);
CREATE INDEX idx_operations_user ON operations(user_id);
CREATE INDEX idx_operations_status ON operations(status);

CREATE TABLE api_tokens (
    id           TEXT PRIMARY KEY,  -- uuid v7
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    scopes       TEXT NOT NULL,     -- read|write|admin
    created_at   INTEGER NOT NULL,
    expires_at   INTEGER,
    last_used_at INTEGER
);

CREATE TABLE totp_secrets (
    user_id    INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret     TEXT NOT NULL,
    enabled    INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE totp_recovery_codes (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,        -- argon2id
    used      INTEGER NOT NULL DEFAULT 0
);
```

Migrations: numbered SQL files in `apps/hellingd/internal/store/migrations/`, applied at boot via `golang-migrate` or hand-rolled (decision in §10).

We do **not** persist instance state in SQLite. Incus is the source of truth for instance state. Helling persists only its own state: users, sessions, audit operations.

## 5. Incus integration

`hellingd` talks to Incus through a narrow internal client built on Go's standard `net/http` transport over the Incus restricted user socket at `/var/lib/incus/user.socket`. The direct Incus module dependency is intentionally avoided until the upstream vulnerability set is clear for our import graph.

Service account: the `hellingd` systemd unit runs as the `helling` user, added to the `incus` group. `incus-admin` is escalation, not default; if a future feature needs that authority it must move into a separate privileged helper rather than promoting `hellingd`.

Operations against Incus are async. The pattern:

```go
op, err := client.CreateInstance(req)
if err != nil { return err }
// op is *Operation. Don't block. Persist op.URL() and poll, or use op.AddHandler.
```

Helling owns the operation lifecycle visible to its API consumers. It does not surface raw Incus operation IDs.

## 6. Authentication and authorization

### v0.1

- **Local users only.** Stored in `users` table. Passwords hashed with argon2id (`golang.org/x/crypto/argon2`), parameters: `t=3, m=64MiB, p=4, saltLen=16, keyLen=32`. These match current OWASP guidance for argon2id. Tune up if benchmarks allow.
- **Session + JWT auth.** On `POST /v1/auth/login`, generate a 32-byte random session ID (`crypto/rand`), store hash in `sessions` (so DB compromise doesn't grant active sessions), set as HTTP-only `Secure` `SameSite=Lax` cookie, 7-day expiry, sliding window via `last_seen_at`. API and CLI flows also receive short-lived Ed25519-signed access JWTs per ADR-031.
- **Persistent JWT signing key.** The Ed25519 seed is loaded from the configured key path or generated on first boot with `0600` permissions.
- **TOTP MFA.** Users can enroll TOTP. Once enabled, password login returns a pre-session MFA challenge; no session or JWT is issued until a valid TOTP code or one-time recovery code is presented. Recovery codes are hashed with argon2id.
- **API tokens.** Users can create scoped API tokens (`read`, `write`, `admin`). Only token hashes are stored. Expired tokens and insufficient scopes are rejected by middleware.
- **No PAM.** Removed from earlier scope. Reasons: pulls CGO into the build, ties auth to local OS users (which is a poor fit for a multi-user web tool), expands attack surface, and offers nothing v0.1 actually needs.
- **Two roles only:** `user` and `admin`. Admin checks are explicit at the route/handler boundary. User management, config/upgrade, audit, notifications, schedules, webhooks, firewall, BMC, Kubernetes, and raw native proxy surfaces are admin-only in v0.1. Fine-grained RBAC matrix is post-v0.1.
- **Deferred privileged endpoints.** Post-v0.1 surfaces mounted for UI parity return explicit `501 Not Implemented`; they must not persist secrets, create placeholder resources, or return fake queued/staged success.
- **Raw proxy boundary.** Non-admin raw Incus/Podman proxy requests are rejected in v0.1. Non-admin Incus proxy access must wait for ADR-024 per-user mTLS over loopback HTTPS; raw Unix-socket forwarding is never used for non-admin requests.

### Future (post-v0.1, non-binding)

- OIDC SSO via `coreos/go-oidc` for SaaS-style deployments.
- Mutual TLS for inter-service auth as an alternative to cookies/JWTs.
- Fine-grained permissions tied to Incus projects.

## 7. API contract

`api/openapi.yaml` is the single source of truth. OpenAPI 3.1 (downgraded to 3.0 in the codegen step where required by oapi-codegen).

Generation:

- Server: `oapi-codegen` with `chi-server` output, generated into `apps/hellingd/api/server.gen.go`.
- Models: same package, `apps/hellingd/api/types.gen.go`.
- Go client: `oapi-codegen` with `client` output, into `apps/helling-cli/internal/client/client.gen.go`.
- TS client: `@hey-api/openapi-ts` into `web/src/api/generated/`. React Query hooks output enabled.

Generated files are committed. `make check-generated` confirms the working tree is clean after `make generate`. CI fails on drift.

We do not hand-edit generated code, ever. Behavior changes go through the OpenAPI spec.

## 8. Logging and observability

- **Structured logging:** `log/slog` (stdlib) with JSON handler in production, text in dev.
- **Request log fields:** `request_id`, `user_id` (when authenticated), `method`, `path`, `status`, `duration_ms`, `bytes_out`.
- **No metrics in v0.1.** Prometheus `/metrics` is post-v0.1.
- **No tracing in v0.1.** OpenTelemetry hooks reserved at the middleware boundary for later.
- **Audit log:** every state-changing API call appends a row to `operations` (or a separate `audit` table — TBD in implementation).

## 9. Deployment topology (v0.1)

Single host, Debian 13 (or current Debian stable):

```text
systemd
  ├── hellingd.service (hellingd, runs as user `helling`)
  └── caddy.service    (Caddy, member of group `helling-proxy`)
```

State directories:

- `/var/lib/helling` — SQLite + state, owned by `helling:helling`
- `/run/helling/api.sock` — Unix socket, group `helling-proxy`, mode 0660
- `/etc/helling/helling.yaml` — config, root:helling 0640
- `/var/log/helling/` — logs (or rely on journald)

Container images are not shipped in v0.1. Helling installs through the Debian-first ISO/systemd path from ADR-021.

## 10. Open questions (must resolve before v0.1 ships)

1. **SQLite driver**: pure-Go `modernc.org/sqlite` (no CGO) vs CGO `mattn/go-sqlite3`. Default: pure-Go. Re-evaluate if benchmarks show meaningful regression for our workload (mostly small writes).
2. **Migration tool**: `golang-migrate/migrate` library, hand-rolled `embed.FS` runner, or `pressly/goose`. Default: hand-rolled (it's ~150 LOC and removes a dep).
3. **OpenAPI spec format**: 3.0 vs 3.1. oapi-codegen still has limited 3.1 support — start with 3.0.x for the contract, revisit when oapi-codegen catches up.
4. **TLS cert sourcing for Caddy**: file paths only in v0.1 (`tls.cert`, `tls.key`), or Caddy-managed ACME for public hosts? Default: file paths only in the ISO profile.
5. **Web auth store**: cookie session (this doc) vs bearer token in `localStorage`. Default: cookie. localStorage tokens have known XSS-exfil risk.

## 11. Non-goals and explicit rejections

- **No PAM integration.** Decided in §6.
- **No CGO unless we lose this argument on SQLite.** §10.1.
- **No Kubernetes orchestration.** It's a different product. Helling is for people who don't want K8s.
- **No "API gateway" abstraction layer.** chi + middleware is enough.
- **No GraphQL.** REST + JSON. The spec drives the contract.
- **No microservices in v0.1.** Two Helling Go binaries (`hellingd`, `helling-cli`) plus packaged Caddy is the floor, not a starting point.
- **No event bus, message queue, or pub/sub.** State changes are synchronous DB writes; long-running work is goroutines + the `operations` table.

## 12. Glossary

- **Instance** — Incus term for a container or VM. We use it the same way.
- **Operation** — an async unit of work. Both Incus and Helling have these; the doc distinguishes when context is ambiguous.
- **Project** — Incus's tenant boundary. Not a Helling concept yet; mapped 1:1 to "default" in v0.1.
