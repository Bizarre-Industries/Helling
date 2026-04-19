# API Specification

hellingd exposes three API surfaces:

1. **Helling API** (`/api/v1/*`) for Helling-specific features.
2. **Incus Proxy** (`/api/incus/*`) forwarded to the local Incus HTTPS API with per-user mTLS identity.
3. **Podman Proxy** (`/api/podman/*`) forwarded to `/run/podman/podman.sock`.

Proxied requests are authenticated and audited. Incus proxy requests run under the caller's dedicated TLS certificate identity (ADR-024).

---

## Helling API Domains

## 1. Auth

| Method | Endpoint                  | Description                         |
| ------ | ------------------------- | ----------------------------------- |
| POST   | /api/v1/auth/setup        | First admin creation (one-time)     |
| POST   | /api/v1/auth/login        | PAM authenticate and issue JWT pair |
| POST   | /api/v1/auth/refresh      | Refresh access token                |
| POST   | /api/v1/auth/logout       | Revoke session                      |
| POST   | /api/v1/auth/mfa/complete | Complete MFA challenge              |
| POST   | /api/v1/auth/totp/setup   | Enable TOTP                         |
| POST   | /api/v1/auth/totp/verify  | Verify TOTP setup                   |
| DELETE | /api/v1/auth/totp         | Disable TOTP                        |
| GET    | /api/v1/auth/tokens       | List API tokens                     |
| POST   | /api/v1/auth/tokens       | Create API token                    |
| DELETE | /api/v1/auth/tokens/{id}  | Revoke API token                    |

## 2. Users

| Method | Endpoint           | Description                                    |
| ------ | ------------------ | ---------------------------------------------- |
| GET    | /api/v1/users      | List users                                     |
| POST   | /api/v1/users      | Create user and provision Incus trust identity |
| GET    | /api/v1/users/{id} | Get user detail                                |
| PUT    | /api/v1/users/{id} | Update user role and trust scope               |
| DELETE | /api/v1/users/{id} | Delete user and revoke trust identity          |

## 3. Schedules

| Method | Endpoint                   | Description      |
| ------ | -------------------------- | ---------------- |
| GET    | /api/v1/schedules          | List schedules   |
| POST   | /api/v1/schedules          | Create schedule  |
| GET    | /api/v1/schedules/{id}     | Get schedule     |
| PUT    | /api/v1/schedules/{id}     | Update schedule  |
| DELETE | /api/v1/schedules/{id}     | Delete schedule  |
| POST   | /api/v1/schedules/{id}/run | Run schedule now |

## 4. Webhooks

| Method | Endpoint                   | Description        |
| ------ | -------------------------- | ------------------ |
| GET    | /api/v1/webhooks           | List webhooks      |
| POST   | /api/v1/webhooks           | Create webhook     |
| GET    | /api/v1/webhooks/{id}      | Get webhook        |
| PUT    | /api/v1/webhooks/{id}      | Update webhook     |
| DELETE | /api/v1/webhooks/{id}      | Delete webhook     |
| POST   | /api/v1/webhooks/{id}/test | Send test delivery |

## 5. BMC

Deferred from v0.1 (target v0.4).

| Method | Endpoint                 | Description         |
| ------ | ------------------------ | ------------------- |
| GET    | /api/v1/bmc              | List BMC endpoints  |
| POST   | /api/v1/bmc              | Add BMC endpoint    |
| GET    | /api/v1/bmc/{id}         | Get BMC detail      |
| DELETE | /api/v1/bmc/{id}         | Remove BMC endpoint |
| POST   | /api/v1/bmc/{id}/power   | Power control       |
| GET    | /api/v1/bmc/{id}/sensors | Sensor data         |
| GET    | /api/v1/bmc/{id}/sel     | System event log    |

## 6. Kubernetes

| Method | Endpoint                             | Description                                      |
| ------ | ------------------------------------ | ------------------------------------------------ |
| GET    | /api/v1/kubernetes                   | List clusters                                    |
| POST   | /api/v1/kubernetes                   | Create cluster (k3s via cloud-init on Incus VMs) |
| GET    | /api/v1/kubernetes/{name}            | Cluster detail                                   |
| DELETE | /api/v1/kubernetes/{name}            | Delete cluster                                   |
| POST   | /api/v1/kubernetes/{name}/scale      | Scale worker pool                                |
| POST   | /api/v1/kubernetes/{name}/upgrade    | Rolling upgrade                                  |
| GET    | /api/v1/kubernetes/{name}/kubeconfig | Download kubeconfig                              |

## 7. System

| Method | Endpoint                   | Description           |
| ------ | -------------------------- | --------------------- |
| GET    | /api/v1/system/info        | Host and version info |
| GET    | /api/v1/system/hardware    | Hardware detail       |
| GET    | /api/v1/system/config      | Current config        |
| PUT    | /api/v1/system/config      | Update config         |
| POST   | /api/v1/system/upgrade     | Apply upgrades        |
| GET    | /api/v1/system/diagnostics | Self-test             |

## 8. Host Firewall

| Method | Endpoint                   | Description               |
| ------ | -------------------------- | ------------------------- |
| GET    | /api/v1/firewall/host      | List host nftables rules  |
| POST   | /api/v1/firewall/host      | Create host firewall rule |
| DELETE | /api/v1/firewall/host/{id} | Delete host firewall rule |

## 9. Audit

| Method | Endpoint             | Description       |
| ------ | -------------------- | ----------------- |
| GET    | /api/v1/audit        | Query audit log   |
| GET    | /api/v1/audit/export | Export audit data |

## 10. Notifications

Deferred from v0.1 (target v0.3).

| Method | Endpoint                                 | Description    |
| ------ | ---------------------------------------- | -------------- |
| GET    | /api/v1/notifications/channels           | List channels  |
| POST   | /api/v1/notifications/channels           | Create channel |
| DELETE | /api/v1/notifications/channels/{id}      | Delete channel |
| POST   | /api/v1/notifications/channels/{id}/test | Test channel   |

## 11. Infrastructure

| Method | Endpoint       | Description              |
| ------ | -------------- | ------------------------ |
| GET    | /api/v1/health | Health check             |
| GET    | /api/v1/events | Helling SSE event stream |

---

## WebSocket Console Protocol (Incus)

Console and exec sessions follow the upstream Incus operation-secret workflow. Helling does not rewrite this protocol.

1. Client opens an operation via Incus API, for example:
   - `POST /api/incus/1.0/instances/{name}/console`
   - `POST /api/incus/1.0/instances/{name}/exec`
2. Response contains an operation object with websocket secret values in `metadata.fds`.
3. Client connects websocket channel(s) through the proxy:
   - `GET /api/incus/1.0/operations/{id}/websocket?secret={fd_secret}`
4. For console flows, data and control channels are both connected.
5. For exec flows, stdin/stdout/stderr websocket channels are connected per returned file-descriptor mapping.

Proxy behavior requirements:

- Preserve websocket upgrade semantics and upstream payloads without protocol translation.
- Authenticate and authorize before operation creation and websocket upgrades.
- Emit audit for session open/close events, not per-frame traffic.
- For Incus upstream calls, present caller-specific client certificate identity on the loopback HTTPS transport.

---

## Events Model

Helling exposes two event surfaces:

1. **Incus upstream events (WebSocket pass-through):**
   - `GET /api/incus/1.0/events`
   - Protocol and event type semantics are native Incus behavior.
2. **Helling-native events (SSE):**
   - `GET /api/v1/events`
   - Emits Helling control-plane events such as schedule execution, webhook delivery, and auth/audit lifecycle notifications.

Helling-native SSE behavior:

- Content type: `text/event-stream`
- Reconnect via standard `Last-Event-ID`
- Heartbeat comments are sent periodically to keep connections alive
- Optional query filters: `type`, `since`

---

## Proxy Paths

- `/api/incus/*` forwards to Incus REST API via local HTTPS endpoint with per-user client certificate authentication.
- `/api/podman/*` forwards to Podman libpod API via Unix socket.
- Incus Unix socket access is reserved for host administrator CLI operations and is not used for delegated user proxy authorization.

MicroVM API proxy routes are deferred from v0.1 (ADR-006).
