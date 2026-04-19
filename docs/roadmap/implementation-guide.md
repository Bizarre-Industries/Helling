# Helling Implementation Guide

<!-- markdownlint-disable MD029 MD010 -->

**Version:** v4 (2026-04-15)
**Purpose:** Detailed, actionable implementation steps for Helling development
**Audience:** Developers implementing features according to the roadmap

> **Companion to:** [plan.md](./plan.md) (high-level roadmap) and [checklist.md](./checklist.md) (verification gates)

---

## Table of Contents

- [Overview](#overview)
- [Current State](#current-state)
- [Phase 1: v0.1.0-alpha](#phase-1-v010-alpha---foundation)
- [Phase 2: v0.1.0-beta](#phase-2-v010-beta---core-dashboard)
- [Phase 3: v0.2.0](#phase-3-v020---platform-core)
- [Phase 4: v0.3.0+](#phase-4-v030-and-beyond)
- [Critical Success Factors](#critical-success-factors)
- [Risk Mitigation](#risk-mitigation)

---

## Overview

### What is Helling?

Helling is a Proxmox-style hypervisor platform built on Debian 13, combining:

- **Incus** (VMs via QEMU/KVM + system containers via LXC)
- **Podman** (application containers, Docker-compatible)
- **k3s on Incus VMs** for Kubernetes provisioning (Cloud Hypervisor microVM runtime is deferred from v0.1)

All accessible through a unified React dashboard with a proxy-first architecture.

### Core Architectural Principles

1. **Proxy middleware over custom handlers** (ADR-014)
   - ~300 lines of proxy code replaces ~150 endpoint handlers
   - Incus/Podman APIs exposed at `/api/incus/*` and `/api/podman/*`
   - Native upstream response formats (no re-enveloping)

2. **Shell out over Go libraries** (ADR-018)
   - Use system tools: `nft`, `systemctl`, `journalctl`
   - Avoid dependencies: no `google/nftables`, no `go-systemd`

3. **ISO-only deployment** (ADR-021)
   - No Docker mode, no manual installation
   - Boot ISO → answer 3 questions → running system

4. **Minimal dependencies**

- Keep Go dependencies minimal and aligned to proxy-first architecture
- Everything else via proxy or system tools

---

## Current State

**As of 2026-04-15:**

✅ **Exists:**

- Basic structure: 3 Go apps (hellingd, helling-cli, helling-proxy)
- Some handlers: auth, system, schedules, webhooks, microvm
- Frontend: 20 React page components
- OpenAPI spec started (api/openapi.yaml)
- Database models in internal/auth/models.go, internal/db/

⚠️ **Needs Work:**

- Proxy middleware implementation (critical)
- Code generation pipeline (oapi-codegen + orval)
- Complete OpenAPI spec (~40 endpoints)
- Frontend connected to real APIs (currently may have mocks)
- Many v0.1.0-alpha checklist items

---

## Phase 1: v0.1.0-alpha - Foundation

**Gate:** Boot ISO → setup wizard → dashboard shows real Incus instances and Podman containers

**Priority Order:** OpenAPI Spec → Code Generation → Proxy → Auth → Frontend

### 1.1 OpenAPI Spec Completion

**File:** `api/openapi.yaml`
**Priority:** CRITICAL - Everything depends on this

**Current State:** Partial spec exists (started, ~100 lines visible)

**Action Items:**

1. **Complete all ~40 Helling endpoints:**

```yaml
# Auth endpoints (8)
/auth/setup: # POST - first admin (one-time)
/auth/login: # POST - PAM → JWT
/auth/refresh: # POST - refresh access token
/auth/logout: # POST - clear refresh cookie
/auth/mfa/complete: # POST - verify TOTP
/auth/totp/setup: # POST - enable 2FA
/auth/totp/verify: # POST - confirm TOTP setup
/auth/totp: # DELETE - disable 2FA
/auth/tokens: # GET, POST - API token CRUD
/auth/tokens/{id}: # DELETE - revoke token

# Users endpoints (5)
/users: # GET, POST
/users/{id}: # GET, PUT, DELETE

# Schedules endpoints (6)
/schedules: # GET, POST
/schedules/{id}: # GET, PUT, DELETE
/schedules/{id}/run: # POST

# Webhooks endpoints (6)
/webhooks: # GET, POST
/webhooks/{id}: # GET, PUT, DELETE
/webhooks/{id}/test: # POST

# System endpoints (5)
/system/info: # GET
/system/hardware: # GET
/system/config: # GET, PUT
/system/diagnostics: # GET
/system/upgrade: # POST

# Platform endpoints (2)
/health: # GET (public)
/events: # GET (SSE stream)
```

2. **Define complete schemas:**

```yaml
components:
  schemas:
    # Envelope responses
    DataResponse:
      type: object
      required: [data]
      properties:
        data: {}

    ListResponse:
      type: object
      required: [data, meta]
      properties:
        data:
          type: array
        meta:
          type: object
          properties:
            total: { type: integer }
            page: { type: integer }
            per_page: { type: integer }

    ErrorResponse:
      type: object
      required: [error]
      properties:
        error: { type: string }
        code: { type: string }
        action: { type: string }
        doc_link: { type: string }

    # Auth types
    LoginRequest:
      type: object
      required: [username, password]
      properties:
        username: { type: string }
        password: { type: string }
        totp_code: { type: string }

    AuthTokens:
      type: object
      properties:
        access_token: { type: string }
        refresh_token: { type: string }
        expires_in: { type: integer }

    # ... (continue for all request/response types)
```

3. **Add pagination to all list endpoints:**

```yaml
parameters:
  - name: page
    in: query
    schema: { type: integer, default: 1 }
  - name: per_page
    in: query
    schema: { type: integer, default: 50, maximum: 100 }
  - name: sort
    in: query
    schema: { type: string }
  - name: order
    in: query
    schema: { type: string, enum: [asc, desc] }
```

**Verification:**

```bash
# Lint spec
npx @redocly/cli lint api/openapi.yaml
# Should return: 0 errors

# Count endpoints
grep -c "operationId:" api/openapi.yaml
# Should return: ~40

# Validate structure
npx @redocly/cli bundle api/openapi.yaml
# Should succeed without warnings
```

### 1.2 Code Generation Pipeline

**Priority:** CRITICAL - Automates 3 surfaces (backend, CLI, frontend)

**Current State:** Makefile has `generate` target, configs may be incomplete

#### Backend Generation (oapi-codegen strict-server)

**File:** `apps/hellingd/oapi-codegen.yaml`

```yaml
package: api
generate:
  strict-server: true
  chi-server: true
  models: true
  embedded-spec: true
output: api/generated.go
```

**What it generates:**

- `StrictServerInterface` with typed method signatures
- Chi router factory function
- All request/response model structs
- OpenAPI spec embedded as constant

**Usage in code:**

```go
// Before: manual handler registration
router.Post("/api/v1/auth/login", h.Login)

// After: implement generated interface
type Handlers struct {
    Auth *auth.Service
    // ...
}

func (h *Handlers) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
    // Business logic only
    tokens, err := h.Auth.Login(req.Username, req.Password, req.TotpCode)
    return LoginResponse{Data: tokens}, err
}

// Router auto-generated from spec
router := api.NewStrictServer(handlers, middlewares)
```

#### CLI Generation (oapi-codegen client)

**File:** `apps/helling-cli/oapi-codegen.yaml`

```yaml
package: client
generate:
  client: true
  models: true
output: internal/client/generated.go
```

**Usage in CLI:**

```go
// cmd/system.go
func systemInfoCmd(cmd *cobra.Command, args []string) error {
    client, err := getClient()
    if err != nil {
        return err
    }

    // Use generated client method
    resp, err := client.GetSystemInfo(context.Background())
    if err != nil {
        return err
    }

    printJSON(resp.Data)
    return nil
}
```

#### Frontend Generation (orval)

**File:** `web/orval.config.ts`

```typescript
import { defineConfig } from "orval";

export default defineConfig({
  helling: {
    input: "../api/openapi.yaml",
    output: {
      target: "src/api/generated/helling.ts",
      client: "react-query",
      mode: "tags-split",
      override: {
        mutator: {
          path: "src/api/fetcher.ts",
          name: "customFetch",
        },
      },
    },
  },
});
```

**Custom fetcher with JWT:**

```typescript
// src/api/fetcher.ts
export const customFetch = async <T>(config: RequestConfig): Promise<T> => {
  const token = localStorage.getItem("access_token");

  const response = await fetch(config.url, {
    ...config,
    headers: {
      ...config.headers,
      ...(token && { Authorization: `Bearer ${token}` }),
    },
  });

  if (!response.ok) {
    if (response.status === 401) {
      // Try refresh
      await refreshToken();
      // Retry request
      return customFetch(config);
    }
    throw new Error(await response.text());
  }

  return response.json();
};
```

**Usage in components:**

```typescript
import { useGetSystemInfo, useLoginMutation } from '@/api/generated/helling';

function DashboardPage() {
  const { data, isLoading } = useGetSystemInfo();

  return (
    <div>
      <h1>System Info</h1>
      <pre>{JSON.stringify(data?.data, null, 2)}</pre>
    </div>
  );
}
```

#### Makefile Integration

**Update:** `Makefile`

```makefile
generate:
	@echo "==> Generating backend..."
	cd apps/hellingd && oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
	@echo "==> Generating CLI client..."
	cd apps/helling-cli && oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
	@echo "==> Generating frontend hooks..."
	cd web && bunx orval
	@echo "✓ Generation complete"

check-generated: generate
	@echo "==> Checking for stale generated code..."
	@git diff --exit-code apps/hellingd/api/*.gen.go apps/helling-cli/internal/client/*.gen.go web/src/api/generated/ || \
		(echo "ERROR: Generated code is stale. Run 'make generate' and commit." && exit 1)
	@echo "✓ Generated code is up to date"
```

**Verification:**

```bash
make generate
# Should generate:
# - apps/hellingd/api/generated.go
# - apps/helling-cli/internal/client/generated.go
# - web/src/api/generated/helling.ts

make check-generated
# Should pass (no diff)

git status
# Should show updated .gen.go files
```

### 1.3 Proxy Middleware Implementation

**Priority:** CRITICAL - Core of ADR-014

**Files:**

- `apps/hellingd/internal/proxy/proxy.go` - core proxy handler
- `apps/hellingd/internal/proxy/incus.go` - Incus-specific logic
- `apps/hellingd/internal/proxy/podman.go` - Podman-specific logic
- `apps/hellingd/internal/api/middleware.go` - JWT/RBAC/audit

#### Core Proxy Handler

**File:** `apps/hellingd/internal/proxy/proxy.go`

```go
package proxy

import (
    "net/http"
    "net/http/httputil"
    "net"
    "strings"
)

type ProxyConfig struct {
    TargetSocket string
    PathPrefix   string
    Validator    JWTValidator
  Authorizer   Authorizer
    Auditor      Auditor
}

func NewProxyHandler(cfg ProxyConfig) http.Handler {
    // Custom dialer for Unix socket
    dialer := &net.Dialer{}
    transport := &http.Transport{
        DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
            return dialer.DialContext(ctx, "unix", cfg.TargetSocket)
        },
    }

    // Create reverse proxy
    proxy := &httputil.ReverseProxy{
        Director: func(req *http.Request) {
            // Strip path prefix
            req.URL.Path = strings.TrimPrefix(req.URL.Path, cfg.PathPrefix)
            req.URL.Scheme = "http"
            req.URL.Host = "unix"

            // Remove Authorization header (don't forward to upstream)
            req.Header.Del("Authorization")
        },
        Transport: transport,
        ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
            http.Error(w, "proxy error: "+err.Error(), http.StatusBadGateway)
        },
    }

    // Wrap with middleware
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check for WebSocket upgrade
        if isWebSocket(r) {
            handleWebSocketProxy(w, r, cfg.TargetSocket)
            return
        }

        proxy.ServeHTTP(w, r)
    })

    return WithMiddleware(handler, cfg)
}

func isWebSocket(r *http.Request) bool {
    return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
           strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}
```

#### WebSocket Proxy

```go
func handleWebSocketProxy(w http.ResponseWriter, r *http.Request, targetSocket string) {
    // Upgrade client connection
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }
    clientConn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer clientConn.Close()

    // Connect to upstream Unix socket
    upstreamConn, err := dialWebSocketUnix(targetSocket, r.URL.Path)
    if err != nil {
        return
    }
    defer upstreamConn.Close()

    // Bidirectional copy
    done := make(chan struct{}, 2)

    go func() {
        io.Copy(clientConn.UnderlyingConn(), upstreamConn.UnderlyingConn())
        done <- struct{}{}
    }()

    go func() {
        io.Copy(upstreamConn.UnderlyingConn(), clientConn.UnderlyingConn())
        done <- struct{}{}
    }()

    <-done
}
```

#### Middleware Stack

**File:** `apps/hellingd/internal/api/middleware.go`

```go
func WithMiddleware(handler http.Handler, cfg proxy.ProxyConfig) http.Handler {
    return WithJWT(
    WithPolicy(
            WithAudit(
                WithAutoSnapshot(handler, cfg),
                cfg.Auditor,
            ),
      cfg.Authorizer,
        ),
        cfg.Validator,
    )
}

func WithJWT(next http.Handler, validator JWTValidator) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractToken(r)
        if token == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        claims, err := validator.Validate(token)
        if err != nil {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }

        // Inject claims into context
        ctx := context.WithValue(r.Context(), "user", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func WithPolicy(next http.Handler, authorizer Authorizer) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*UserClaims)

    // Authorize request against role + scope policy. For Incus, scope is
    // enforced by per-user Incus trust identities (restricted=true + project).
    if !authorizer.Allow(user, r.Method, r.URL.Path) {
      http.Error(w, "forbidden", http.StatusForbidden)
      return
        }

        next.ServeHTTP(w, r)
    })
}

func WithAudit(next http.Handler, auditor Auditor) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Log asynchronously (don't block)
        go auditor.Log(AuditEntry{
            User:   r.Context().Value("user").(*UserClaims).Username,
            Method: r.Method,
            Path:   r.URL.Path,
            Time:   time.Now(),
        })

        next.ServeHTTP(w, r)
    })
}
```

#### Router Setup

**File:** `apps/hellingd/internal/api/router.go`

```go
func NewRouter(h *Handlers) http.Handler {
    r := chi.NewRouter()

    // Global middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Public endpoints (no auth)
    r.Get("/api/v1/health", h.GetHealth)

    // Helling API (generated strict server)
    strictHandler := api.NewStrictServer(h, middlewares...)
    r.Mount("/api/v1", strictHandler)

    // Incus proxy
    incusProxy := proxy.NewProxyHandler(proxy.ProxyConfig{
        TargetSocket:  h.IncusSocketPath,
        PathPrefix:    "/api/incus",
        Validator:     h.Auth,
      Authorizer:    h.Authz,
        Auditor:       h.Auditor,
    })
    r.Handle("/api/incus/*", incusProxy)

    // Podman proxy
    podmanProxy := proxy.NewProxyHandler(proxy.ProxyConfig{
        TargetSocket:  h.PodmanSocketPath,
        PathPrefix:    "/api/podman",
        Validator:     h.Auth,
      Authorizer:    h.Authz,
        Auditor:       h.Auditor,
    })
    r.Handle("/api/podman/*", podmanProxy)

    return r
}
```

**Verification:**

```bash
# Start hellingd
./bin/hellingd

# Get JWT token
TOKEN=$(curl -X POST http://localhost:8006/api/v1/auth/login \
  -d '{"username":"admin","password":"test"}' | jq -r '.data.access_token')

# Test Incus proxy
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8006/api/incus/1.0/instances | jq

# Should return Incus native response format

# Test without auth
curl http://localhost:8006/api/incus/1.0/instances
# Should return 401
```

### 1.4 Auth Handlers Implementation

**Priority:** HIGH - Required for login

**Files:**

- `apps/hellingd/internal/auth/service.go`
- `apps/hellingd/internal/auth/jwt.go`
- `apps/hellingd/internal/auth/totp.go`
- `apps/hellingd/internal/auth/pam.go`

**Implementation:** Use generated `StrictServerInterface`, implement business logic only.

```go
// apps/hellingd/internal/api/handlers.go
func (h *Handlers) SetupAdmin(ctx context.Context, req api.SetupAdminRequest) (api.SetupAdminResponse, error) {
    if !h.Auth.IsSetupRequired() {
        return api.SetupAdminResponse{}, &api.Error{
            Code: "ALREADY_SETUP",
            Message: "Setup already complete",
        }
    }

    tokens, err := h.Auth.CreateFirstAdmin(req.Username, req.Password)
    if err != nil {
        return api.SetupAdminResponse{}, err
    }

    return api.SetupAdminResponse{
        Data: api.AuthTokens{
            AccessToken:  tokens.AccessToken,
            RefreshToken: tokens.RefreshToken,
            ExpiresIn:    900, // 15 min
        },
    }, nil
}

func (h *Handlers) Login(ctx context.Context, req api.LoginRequest) (api.LoginResponse, error) {
    tokens, mfaRequired, err := h.Auth.Login(req.Username, req.Password, req.TotpCode)
    if err != nil {
        return api.LoginResponse{}, err
    }

    if mfaRequired {
        return api.LoginResponse{
            MfaRequired: true,
            MfaToken: tokens.MfaToken,
        }, nil
    }

    // Set refresh token cookie
    // (middleware handles this via response modifier)

    return api.LoginResponse{
        Data: api.AuthTokens{
            AccessToken: tokens.AccessToken,
            ExpiresIn: 900,
        },
    }, nil
}
```

**Verification:**

```bash
# First setup
curl -X POST http://localhost:8006/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme123"}'

# Should return:
# {"data":{"access_token":"eyJ...","expires_in":900}}

# Login
curl -X POST http://localhost:8006/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme123"}'

# Should return JWT tokens
```

### 1.5 Frontend - Real API Integration

**Priority:** HIGH - v0.1 gate requires dashboard with real data

#### Three API Clients Setup

**File:** `web/src/api/incus.ts`

```typescript
import { apiFetch } from "@/api/fetch";

export interface IncusInstance {
  name: string;
  status: string;
  type: "container" | "virtual-machine";
  state: {
    cpu: { usage: number };
    memory: { usage: number; total: number };
    network: Record<
      string,
      {
        addresses: Array<{ family: string; address: string }>;
      }
    >;
  };
}

export interface IncusResponse<T> {
  metadata: T;
  status: string;
  status_code: number;
}

export class IncusClient {
  async listInstances(project?: string): Promise<IncusInstance[]> {
    const url = project
      ? `/api/incus/1.0/instances?recursion=2&project=${encodeURIComponent(project)}`
      : "/api/incus/1.0/instances?recursion=2";

    const data = await apiFetch<IncusResponse<IncusInstance[]>>(url);
    return data.metadata;
  }

  async startInstance(name: string, project?: string): Promise<void> {
    const url = project
      ? `/api/incus/1.0/instances/${name}/state`
      : `/api/incus/1.0/instances/${name}/state`;

    await apiFetch(url, {
      method: "PUT",
      body: JSON.stringify({ action: "start" }),
    });
  }

  async stopInstance(name: string, project?: string): Promise<void> {
    const url = project
      ? `/api/incus/1.0/instances/${name}/state`
      : `/api/incus/1.0/instances/${name}/state`;

    await apiFetch(url, {
      method: "PUT",
      body: JSON.stringify({ action: "stop", force: true }),
    });
  }

  // ... more methods
}

export const incusClient = new IncusClient();
```

**File:** `web/src/api/podman.ts`

```typescript
export interface PodmanContainer {
  Id: string;
  Names: string[];
  Image: string;
  Status: string;
  State: string;
  Created: number;
}

export class PodmanClient {
  async listContainers(): Promise<PodmanContainer[]> {
    return apiFetch<PodmanContainer[]>(
      "/api/podman/v5.0/libpod/containers/json?all=true",
    );
  }

  async startContainer(id: string): Promise<void> {
    await apiFetch(`/api/podman/v5.0/libpod/containers/${id}/start`, {
      method: "POST",
    });
  }

  async stopContainer(id: string): Promise<void> {
    await apiFetch(`/api/podman/v5.0/libpod/containers/${id}/stop`, {
      method: "POST",
    });
  }

  // ... more methods
}

export const podmanClient = new PodmanClient();
```

#### Update Dashboard Pages

**File:** `web/src/pages/DashboardPage.tsx`

```typescript
import { useGetSystemInfo } from '@/api/generated/helling';
import { incusClient } from '@/api/incus';
import { podmanClient } from '@/api/podman';
import { useQuery } from '@tanstack/react-query';

export function DashboardPage() {
  const { data: systemInfo } = useGetSystemInfo();

  const { data: instances } = useQuery({
    queryKey: ['incus', 'instances'],
    queryFn: () => incusClient.listInstances()
  });

  const { data: containers } = useQuery({
    queryKey: ['podman', 'containers'],
    queryFn: () => podmanClient.listContainers()
  });

  return (
    <div>
      <Row gutter={16}>
        <Col span={6}>
          <Card title="System">
            <Statistic
              title="Uptime"
              value={systemInfo?.data.uptime}
              suffix="seconds"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card title="Instances">
            <Statistic
              title="Running"
              value={instances?.filter(i => i.status === 'Running').length || 0}
            />
            <Statistic
              title="Total"
              value={instances?.length || 0}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card title="Containers">
            <Statistic
              title="Running"
              value={containers?.filter(c => c.State === 'running').length || 0}
            />
            <Statistic
              title="Total"
              value={containers?.length || 0}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
```

**File:** `web/src/pages/InstancesPage.tsx`

```typescript
export function InstancesPage() {
  const { data: instances, isLoading } = useQuery({
    queryKey: ['incus', 'instances'],
    queryFn: () => incusClient.listInstances()
  });

  const startMutation = useMutation({
    mutationFn: (name: string) => incusClient.startInstance(name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['incus', 'instances'] })
  });

  const columns = [
    { title: 'Name', dataIndex: 'name' },
    {
      title: 'Status',
      dataIndex: 'status',
      render: (status: string) => (
        <Badge status={status === 'Running' ? 'success' : 'default'} text={status} />
      )
    },
    { title: 'Type', dataIndex: 'type' },
    {
      title: 'IPv4',
      render: (_, record: IncusInstance) => {
        const eth0 = record.state?.network?.eth0;
        const ipv4 = eth0?.addresses?.find(a => a.family === 'inet');
        return ipv4?.address || '-';
      }
    },
    {
      title: 'Actions',
      render: (_, record: IncusInstance) => (
        <Space>
          <Button
            size="small"
            onClick={() => startMutation.mutate(record.name)}
            disabled={record.status === 'Running'}
          >
            Start
          </Button>
          <Button
            size="small"
            danger
            onClick={() => stopMutation.mutate(record.name)}
            disabled={record.status !== 'Running'}
          >
            Stop
          </Button>
        </Space>
      )
    }
  ];

  return (
    <Table
      loading={isLoading}
      dataSource={instances || []}
      columns={columns}
      rowKey="name"
    />
  );
}
```

**Verification:**

```bash
cd web
bun dev

# Open http://localhost:5173
# Login with admin credentials
# Dashboard should show:
# - Real system stats from /api/v1/system/info
# - Real instance count from /api/incus/1.0/instances
# - Real container count from /api/podman/.../containers/json

# Navigate to Instances page
# Should show list from Incus
# Start/stop buttons should work
```

### 1.6 Code Cleanup

**Priority:** MEDIUM - Keeps codebase aligned with ADRs

**Enforce ISO-only deployment posture (ADR-021):**

```bash
# Validate Docker artifacts are absent (do not re-introduce them)
test ! -f deploy/Dockerfile
test ! -f deploy/docker-compose.yml
test ! -f deploy/entrypoint.sh

# Review docs for stale references and update manually where needed
rg -n "Docker mode|docker-compose|deploy/Dockerfile" docs/
```

**Remove unused dependencies (ADR-018, ADR-017, ADR-011):**

```bash
cd apps/hellingd

# Remove from go.mod:
# - google/nftables (use nft CLI)
# - go-co-op/gocron (use systemd timers)
# - containers/podman/v5 (proxy, no bindings)

go mod edit -droprequire github.com/google/nftables
go mod edit -droprequire github.com/go-co-op/gocron
# (if podman bindings are present)

go mod tidy
```

**Verification:**

```bash
# No Docker files
ls deploy/Dockerfile
# Should not exist

# No unwanted dependencies
grep "google/nftables\|gocron\|containers/podman" apps/hellingd/go.mod
# Should return nothing

# No TODOs/stubs
grep -r "TODO\|FIXME\|stub" apps/ --include="*.go" | grep -v _test.go
# Should return nothing (or very few)
```

### 1.7 v0.1.0-alpha Final Verification

**Run full checklist from:** `docs/roadmap/checklist.md`

```bash
# Build
make build          # zero warnings
make test           # all pass
make lint           # clean
make generate       # succeeds
make check-generated # no diff

# Frontend
cd web && bun run build  # succeeds

# Proxy works
TOKEN=$(curl -X POST http://localhost:8006/api/v1/auth/login \
  -d '{"username":"admin","password":"test"}' | jq -r '.data.access_token')

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8006/api/incus/1.0/instances | jq '.metadata'
# Returns Incus instances

# Auth works
curl -X POST http://localhost:8006/api/v1/auth/setup \
  -d '{"username":"admin","password":"test123"}'
# Returns tokens

# Dashboard works
cd web && bun dev
# Visit http://localhost:5173
# All pages load without errors
# Real data from APIs displayed

# Code hygiene
npx @redocly/cli lint api/openapi.yaml  # 0 errors
grep -c "operationId:" api/openapi.yaml # ~40
```

---

## Phase 2: v0.1.0-beta - Core Dashboard

**Gate:** Create VM -> SPICE VGA console -> exec into CT -> browse storage pools

### 2.1 WebSocket Proxy for Consoles

**Priority:** HIGH - Required for SPICE/serial/exec

**Implementation:** Extend proxy.go with WebSocket detection and forwarding

**Key files:**

- `apps/hellingd/internal/proxy/websocket.go`

**Endpoints:**

- `/api/incus/1.0/instances/{name}/console?type=vga` - SPICE VGA
- `/api/incus/1.0/instances/{name}/console?type=console` - Serial
- `/api/incus/1.0/instances/{name}/exec` - Exec PTY

**Verification:**

```javascript
// In browser console
const ws = new WebSocket(
  "ws://localhost:8006/api/incus/1.0/instances/test/console?type=vga",
);
ws.onmessage = (e) => console.log("Received:", e.data);
// Should see VGA console stream frames
```

### 2.2 SPICE Browser Console Component

**Priority:** HIGH

**Install dependencies:**

```bash
cd web
bun add spice-html5
# Or vendor files to public/spice/
```

**Component:** `web/src/components/VncConsole.tsx`

**Usage in detail page:** Add to `InstanceDetailPage.tsx` console tab

### 2.3 xterm.js Terminals

**Priority:** HIGH

**Install:**

```bash
cd web
bun add @xterm/xterm
```

**Components:**

- `web/src/components/SerialConsole.tsx` - for serial TTY
- `web/src/components/ExecTerminal.tsx` - for exec shell

### 2.4 Full Detail Pages

**Priority:** MEDIUM

**Instances:** 8 tabs (Overview, Console, Exec, Snapshots, Backups, Logs, Config, Metrics)

**Containers:** 6 tabs (Overview, Logs, Exec, Inspect, Stats, Files)

---

## Phase 3: v0.2.0 - Platform Core

**Features:**

- Schedules (systemd timers)
- Host firewall (nft CLI)
- Tags (Incus user.\* config)
- Embedded API docs (Scalar/Redoc)

---

## Phase 4: v0.3.0 and Beyond

**v0.3.0:** Observability (Prometheus, warnings, notifications)

**v0.4.0:** K8s + BMC + Clustering

**v0.5.0:** Enterprise auth (LDAP, OIDC, WebAuthn)

**v0.8.0:** Production hardening (fuzzing, performance, security)

**v1.0.0:** Packaging, ISO, release artifacts

---

## Critical Success Factors

1. **Get the proxy right first** - Everything else depends on it
2. **Code generation pipeline** - Define once, generate everywhere
3. **Shell out over libraries** - Fewer dependencies, simpler code
4. **ISO-only deployment** - One path, tested thoroughly
5. **Minimize custom code** - Let Incus/Podman do the heavy lifting

---

## Risk Mitigation

**WebSocket proxy complexity:**

- Use proven libraries (gorilla/websocket)
- Extensive testing with real SPICE/serial/exec sessions
- Fallback: direct connections if proxy fails

**PAM reliability:**

- Thorough testing on Debian 13
- Clear error messages for failures
- Document PAM config requirements

**Upstream API changes:**

- Version pin Incus/Podman in ISO
- Monitor changelogs
- Document supported versions

**Database migrations:**

- Always backup before upgrade
- Test migrations on copy first
- Use reviewed forward-only SQL migrations (goose) and keep query contracts in sqlc-managed SQL files

---

## Questions Before Implementation

1. **Authentication:** LDAP/OIDC in v0.5 or earlier?
2. **Target hardware:** Bare metal only or VMs too?
3. **MicroVMs:** Core feature or optional?
4. **BMC:** Required or optional build tag?
5. **Frontend:** Mobile responsive in v0.1?

---

**Next Steps:**

1. Complete OpenAPI spec
2. Set up code generation
3. Implement proxy middleware
4. Wire frontend to real APIs
5. Run v0.1.0-alpha verification checklist
