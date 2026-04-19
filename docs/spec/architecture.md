# Helling Architecture

Helling v0.1 is a proxy-first Debian platform for Incus and Podman, with a focused set of Helling-owned control-plane endpoints.

## System Overview

- Browser UI terminates at Caddy over HTTPS.
- Caddy serves static web assets and forwards `/api/*` to `hellingd`.
- `hellingd` enforces authn/authz and either:
  - handles Helling-specific endpoints (`/api/v1/*`), or
  - proxies upstream APIs for Incus (`/api/incus/*`) and Podman (`/api/podman/*`).

## Transport Model

- Helling API (`/api/v1/*`): HTTP between proxy and daemon over local Unix socket.
- Incus proxy (`/api/incus/*`): forwarded to local Incus HTTPS API using per-user mTLS identity (ADR-024).
- Podman proxy (`/api/podman/*`): forwarded to local Podman Unix socket.
- Incus Unix socket is reserved for host administrator CLI use and is not used as the delegated-user authorization path.

## System Diagram

```
┌───────────────────────────────────────────────────────────────┐
│                         Web Browser                           │
│                     React dashboard UI                         │
└──────────────────────────────┬────────────────────────────────┘
                               │ HTTPS
┌──────────────────────────────▼────────────────────────────────┐
│                           Caddy                                │
│              TLS termination + static asset serving            │
│                 /api/* forwarding to hellingd                  │
└──────────────────────────────┬────────────────────────────────┘
                               │ Unix socket
┌──────────────────────────────▼────────────────────────────────┐
│                          hellingd                              │
│  Auth middleware (PAM+JWT), RBAC, audit, proxy dispatch        │
│                                                                │
│  Helling API handlers (approximately 40 endpoints):            │
│    auth, users, schedules, webhooks, kubernetes, system,       │
│    host firewall, audit, infra                                  │
│                                                                │
│  Upstream proxy surfaces:                                      │
│    /api/incus/*  -> Incus HTTPS API (per-user mTLS cert)       │
│    /api/podman/* -> /run/podman/podman.sock                    │
└──────────────────────────────┬────────────────────────────────┘
                               │
            ┌──────────────────┴──────────────────┐
            │                                     │
┌───────────▼───────────┐             ┌───────────▼───────────┐
│        Incus          │             │         Podman         │
│ VMs, system containers│             │ app containers, pods   │
└───────────────────────┘             └───────────────────────┘
```

## Auth and Authorization

- Authentication is PAM + JWT.
- User credential verification is delegated to PAM.
- JWT signing uses Ed25519.
- Incus authorization boundary is enforced with per-user client certificates presented to Incus.
- Incus trust restrictions (`restricted=true`, project scope/limits) provide resource boundaries.

## Helling-Owned State (SQLite)

SQLite stores Helling control-plane state only. Runtime state for VMs, containers, storage, and networks remains in Incus/Podman.

Current v0.1 schema tables are defined in `docs/spec/sqlite-schema.md` and include:

- `users`
- `sessions`
- `api_tokens`
- `totp_secrets`
- `recovery_codes`
- `incus_user_certs`
- `auth_events`

## Canonical Dependency Baseline (v0.1)

Helling keeps backend dependencies intentionally small and aligned to proxy-first architecture.

Core backend dependencies (approximately 10-12 with build-profile variance):

- `net/http` ServeMux for routing
- `golang-jwt/jwt` for JWT handling
- `msteinert/pam` for PAM integration
- `pquerna/otp` for TOTP
- `gopkg.in/yaml.v3` for config loading
- `database/sql` + generated `sqlc` query layer for state persistence
- `goose` for SQL migrations
- `filippo.io/age` for secret encryption
- `bmc-toolbox/bmclib` for deferred BMC integration paths
- minimal observability/system libraries as required by v0.1 handlers

Incus and Podman APIs are not linked via heavy SDK dependency surfaces for proxied operations.

## Request Flow

### Proxied upstream request

1. Request arrives at Caddy.
2. Forward to `hellingd`.
3. `hellingd` validates JWT/API token and resolves user context.
4. For Incus requests, `hellingd` loads user client cert and forwards with mTLS identity.
5. For Podman requests, `hellingd` forwards to Podman Unix socket.
6. Audit trail emitted for allow/deny and mutations.

### Helling-native request

1. Request arrives at Caddy.
2. Forward to `hellingd`.
3. `hellingd` validates auth and authorization.
4. Helling handler executes domain logic.
5. Response returned in Helling contract format.

## Scope Notes

- MicroVM API routes are deferred from v0.1 (ADR-006).
- Kubernetes provisioning in v0.1 uses k3s via cloud-init on Incus VMs.
