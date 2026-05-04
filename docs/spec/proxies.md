# Proxy Specification

This document specifies the behavior of the authenticated reverse proxy layer that forwards requests to Incus and Podman APIs.

## Overview

The proxy forwards three types of requests in the long-term design. v0.1 keeps raw Incus and Podman proxy routes admin-only; non-admin proxy requests are rejected until ADR-024 per-user Incus mTLS is wired end to end.

1. **Incus requests:** `/api/incus/*` → Incus restricted user socket in v0.1; HTTPS loopback with per-user mTLS is deferred
2. **Podman requests:** `/api/podman/*` → Podman Unix socket
3. **Helling requests:** `/api/v1/*` → Helling-specific handlers (no proxy)

The proxy middleware runs in hellingd (non-root `helling` user per ADR-050) and intercepts all requests after JWT validation.

## Request Flow

```text
Client Request
  ↓
[JWT Validation]
  ↓
[Route Match]
  ├─ /api/incus/* → Incus Proxy
  ├─ /api/podman/* → Podman Proxy
  └─ /api/v1/* → Helling Handler
  ↓
[For Proxies: Enforce Admin Gate]
  ├─ Incus: Reject non-admin; forward through restricted user socket in v0.1
  └─ Podman: Reject non-admin mutation unless explicitly allowed
  ↓
[Forward Request + Audit Log (async)]
  ↓
[Client Response]
```

## TLS Client Lifecycle (Incus Proxy)

This section is the ADR-024 target design, not current v0.1 behavior. Per-user client certificates will be created during user onboarding and reused across requests once loopback HTTPS delegated transport ships.

### Certificate Pool

- **Storage:** SQLite, encrypted with age per ADR-039
- **Key type:** Ed25519 per ADR-031
- **Validity:** 90 days; auto-renewed at 60 days (see docs/spec/internal-ca.md)
- **Loading:** On first request for a user, certificate is decrypted and cached in memory

### Connection Pooling

- **Model:** Per-user HTTP/2 connection pool (TLS session reused across requests)
- **Pool size:** Limited by `net.http.max_idle_conns_per_host` (default 2; configurable)
- **TTL:** 5-minute idle timeout; connection closed after timeout
- **Cache eviction:** LRU when memory pressure detected or pool exceeds max entries per user

### Implementation

```go
type IncusClient struct {
    User          string
    TLSConfig     *tls.Config      // Per-user client cert
    HTTPClient    *http.Client     // With connection pooling
    LastUsed      time.Time
}

var clientCache = &sync.Map{} // Per-user cache

func getIncusClient(user string) (*IncusClient, error) {
    // Check cache
    if cached, ok := clientCache.Load(user); ok {
        return cached.(*IncusClient), nil
    }
    // Load cert from DB, create TLSConfig, build HTTP client
    client := &IncusClient{User: user, ...}
    clientCache.Store(user, client)
    return client, nil
}
```

## WebSocket Semantics

WebSocket upgrade requests (console, exec, serial) follow the same v0.1 admin-only proxy gate. Per-user mTLS identity is deferred with the rest of ADR-024.

### JWT Validity During WebSocket Connection

- **At upgrade:** JWT must be valid (checked before Accept-Upgrade)
- **During connection:** JWT expiry is NOT re-validated (connection lives for operation lifetime)
- **Rationale:** Operations (migration, migration, console sessions) may exceed access token TTL (15 min)
- **Security:** Audit log records WebSocket open/close with user and operation ID

### Audit Events

Every WebSocket is logged:

```json
{
  "user": "alice",
  "action": "websocket_open",
  "target": "instance:vm-web-1",
  "operation_id": "abc123def456",
  "timestamp": "2026-04-20T14:23:45Z"
}
```

At close (successful or error):

```json
{
  "user": "alice",
  "action": "websocket_close",
  "target": "instance:vm-web-1",
  "operation_id": "abc123def456",
  "reason": "normal_close|read_error|write_error",
  "duration_seconds": 1847,
  "timestamp": "2026-04-20T16:54:32Z"
}
```

### Idle Timeout

- **WebSocket idle timeout:** 15 minutes (no data in either direction)
- **Implementation:** Ping/Pong frame every 7 minutes; close on no pong response within 1 minute
- **Behavior:** Client receives `1000 (normal close)`; server-side logs as `idle_timeout`

### Per-User Concurrent WebSocket Cap

- **Limit:** 10 concurrent WebSocket connections per user (configurable via `http.max_websocket_per_user`)
- **Enforcement:** New WebSocket rejected with `429 Too Many Requests` if limit exceeded
- **Rationale:** Prevents resource exhaustion from runaway console clients

## Error Normalization

Helling unifies error response formats from Incus and Podman.

### Incus Native Format

```json
{
  "type": "error",
  "error": "Instance not found",
  "error_code": 404,
  "metadata": {}
}
```

### Podman Native Format

```json
{
  "Errors": ["container not found"],
  "ErrorResponse": {
    "StatusCode": 404
  }
}
```

### Normalized Helling Format (Proxy Response)

```json
{
  "data": null,
  "error": "Instance not found",
  "code": 404,
  "action": "Check instance name; use 'incus list' to list available instances",
  "source": "incus",
  "meta": {
    "request_id": "req-xyz789",
    "upstream_error": "Instance not found"
  }
}
```

### Mapping Rules

| Incus Code | Podman Code | Normalized |
| ---------- | ----------- | ---------- |
| 404        | 404         | 404        |
| 403        | 403         | 403        |
| 500, 502   | 500         | 500        |
| Others     | Others      | 400        |

### Source Annotation

All errors include `source: "incus"` or `source: "podman"` for frontend logging and UI hints.

## Header Rules

### Request Headers: Stripping

Before forwarding to upstream APIs, remove Helling-specific headers:

- `X-Helling-*` — all Helling-internal headers stripped
- `X-Request-ID` — regenerated per upstream request
- `Authorization` — re-issued as TLS client cert (Incus) or dropped (Podman Unix socket)

### Response Headers: Re-attachment

After receiving response from upstream, inject:

- `X-Request-ID` — client's original request ID (for tracing)
- `X-Forwarded-For` — client's IP (for audit and rate limiting)
- `X-Forwarded-User` — authenticated user (for upstream audit if applicable)

### Content Security Policy

All responses from `/api/incus/*` and `/api/podman/*` inherit CSP from hellingd config:

```text
Content-Security-Policy: default-src 'none'; frame-ancestors 'self'; upgrade-insecure-requests
```

## SPICE Console Proxy

SPICE console access (VGA console via SPICE protocol per ADR-010) is forwarded to Incus.

- **Path:** `/api/incus/1.0/instances/{name}/console?type=vga`
- **Protocol:** WebSocket tunnel to SPICE server
- **Auth:** JWT validated before upgrade; v0.1 requires admin role for raw Incus proxy paths
- **Idle timeout:** 15 minutes (same as other WebSockets)

## Rate Limiting

Proxy requests count toward per-user rate limits:

- **API rate limit:** 1000 requests per user per hour (configurable)
- **Enforcement:** Checked at middleware layer before proxy
- **Headers:** `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- **Excess:** `429 Too Many Requests` + retry-after header

## Audit Logging

All proxy requests are logged asynchronously to systemd journal:

```json
{
  "timestamp": "2026-04-20T14:23:45Z",
  "user": "alice",
  "method": "POST",
  "path": "/api/incus/1.0/instances",
  "source_ip": "192.168.1.50",
  "status": 200,
  "duration_ms": 145,
  "request_id": "req-xyz789"
}
```

Audit logs are intended for forensic analysis and do not block request processing.
