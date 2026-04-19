# API Specification

hellingd exposes four API surfaces:

1. **Helling API** (`/api/v1/*`) — ~40 endpoints for Helling-specific features. Documented below.
2. **Incus Proxy** (`/api/incus/*`) — Full Incus REST API, forwarded to `/var/lib/incus/unix.socket`. See [Incus REST API spec](https://linuxcontainers.org/incus/docs/main/rest-api-spec/).
3. **Podman Proxy** (`/api/podman/*`) — Full Podman libpod API, forwarded to `/run/podman/podman.sock`. See [Podman API spec](https://docs.podman.io/en/latest/_static/api.html).
4. **Cloud Hypervisor Proxy** (`/api/ch/{name}/*`) — Full CH REST API per microVM, forwarded to per-VM Unix socket at `/run/ch-{name}/api.sock`.

All three require a valid JWT in the `Authorization: Bearer <token>` header, except public endpoints (health, login, setup).

The proxy adds JWT validation, RBAC project scoping (maps Helling user to Incus project via `?project=` param), and audit logging (to systemd journal) before forwarding.

---

## Response Formats

### Helling endpoints (`/api/v1/*`)

Success:

```json
{ "data": { ... } }
```

List with pagination:

```json
{ "data": [ ... ], "meta": { "total": 100, "page": 1, "per_page": 50 } }
```

Error:

```json
{
  "error": "message",
  "code": "ERROR_CODE",
  "action": "what to do",
  "doc_link": "url"
}
```

Pagination query params: `?page=1&per_page=50&sort=name&order=asc`

### Proxied endpoints (`/api/incus/*`, `/api/podman/*`)

Native upstream response format. No transformation by Helling. See upstream documentation.

---

## 1. Auth

| Method | Endpoint                  | Description                                             |
| ------ | ------------------------- | ------------------------------------------------------- |
| POST   | /api/v1/auth/setup        | First admin creation (one-time, locks after)            |
| POST   | /api/v1/auth/login        | PAM authenticate → JWT pair (access + refresh)          |
| POST   | /api/v1/auth/refresh      | Refresh access token (refresh token in httpOnly cookie) |
| POST   | /api/v1/auth/logout       | Clear refresh token cookie                              |
| POST   | /api/v1/auth/mfa/complete | Complete MFA challenge (TOTP code)                      |
| POST   | /api/v1/auth/totp/setup   | Enable TOTP 2FA (returns QR + recovery codes)           |
| POST   | /api/v1/auth/totp/verify  | Verify TOTP code to confirm setup                       |
| DELETE | /api/v1/auth/totp         | Disable TOTP 2FA                                        |
| GET    | /api/v1/auth/tokens       | List API tokens for current user                        |
| POST   | /api/v1/auth/tokens       | Create API token (name, scope, expiry)                  |
| DELETE | /api/v1/auth/tokens/{id}  | Revoke API token                                        |

## 2. Users

PAM-backed user management. Users map to Incus projects for RBAC scoping.

| Method | Endpoint           | Description                                          |
| ------ | ------------------ | ---------------------------------------------------- |
| GET    | /api/v1/users      | List users                                           |
| POST   | /api/v1/users      | Create user (PAM useradd + Incus project assignment) |
| GET    | /api/v1/users/{id} | Get user detail                                      |
| PUT    | /api/v1/users/{id} | Update user (role, project assignment)               |
| DELETE | /api/v1/users/{id} | Delete user (PAM userdel)                            |

## 3. Schedules

Backup and snapshot schedules managed via systemd timers (ADR-017).

| Method | Endpoint                   | Description                                      |
| ------ | -------------------------- | ------------------------------------------------ |
| GET    | /api/v1/schedules          | List schedules (reads systemd timer units)       |
| POST   | /api/v1/schedules          | Create schedule (writes .timer + .service units) |
| GET    | /api/v1/schedules/{id}     | Get schedule detail + last run status            |
| PUT    | /api/v1/schedules/{id}     | Update schedule                                  |
| DELETE | /api/v1/schedules/{id}     | Delete schedule (removes timer + service units)  |
| POST   | /api/v1/schedules/{id}/run | Trigger schedule immediately                     |

## 4. Webhooks

HMAC-SHA256 signed event delivery with retry.

| Method | Endpoint                   | Description                          |
| ------ | -------------------------- | ------------------------------------ |
| GET    | /api/v1/webhooks           | List webhooks                        |
| POST   | /api/v1/webhooks           | Create webhook (URL, events, secret) |
| GET    | /api/v1/webhooks/{id}      | Get webhook detail + delivery log    |
| PUT    | /api/v1/webhooks/{id}      | Update webhook                       |
| DELETE | /api/v1/webhooks/{id}      | Delete webhook                       |
| POST   | /api/v1/webhooks/{id}/test | Send test delivery                   |

## 5. BMC

BMC/IPMI/Redfish management via bmclib.

| Method | Endpoint                 | Description                                  |
| ------ | ------------------------ | -------------------------------------------- |
| GET    | /api/v1/bmc              | List managed BMC endpoints                   |
| POST   | /api/v1/bmc              | Add BMC endpoint (IP, credentials)           |
| GET    | /api/v1/bmc/{id}         | Get BMC detail                               |
| DELETE | /api/v1/bmc/{id}         | Remove BMC endpoint                          |
| POST   | /api/v1/bmc/{id}/power   | Power on/off/cycle/reset                     |
| GET    | /api/v1/bmc/{id}/sensors | Read sensor data (temperature, fan, voltage) |
| GET    | /api/v1/bmc/{id}/sel     | System event log                             |

## 6. Kubernetes

K8s cluster provisioning via CAPN (Cluster API Provider for Nested virtualization).

| Method | Endpoint                             | Description                                       |
| ------ | ------------------------------------ | ------------------------------------------------- |
| GET    | /api/v1/kubernetes                   | List K8s clusters                                 |
| POST   | /api/v1/kubernetes                   | Create cluster (CAPN: provision VMs, install K8s) |
| GET    | /api/v1/kubernetes/{name}            | Cluster detail (nodes, status, version)           |
| DELETE | /api/v1/kubernetes/{name}            | Delete cluster (destroy VMs)                      |
| POST   | /api/v1/kubernetes/{name}/scale      | Scale worker pool                                 |
| POST   | /api/v1/kubernetes/{name}/upgrade    | Rolling upgrade                                   |
| GET    | /api/v1/kubernetes/{name}/kubeconfig | Download kubeconfig                               |

## 7. System

Helling system management.

| Method | Endpoint                   | Description                                          |
| ------ | -------------------------- | ---------------------------------------------------- |
| GET    | /api/v1/system/info        | Hostname, OS, kernel, CPU, RAM, uptime, version      |
| GET    | /api/v1/system/hardware    | Disks (SMART), NICs, GPUs                            |
| GET    | /api/v1/system/config      | Current helling.yaml config                          |
| PUT    | /api/v1/system/config      | Update config (writes helling.yaml, triggers reload) |
| POST   | /api/v1/system/upgrade     | Check for and apply system upgrade                   |
| GET    | /api/v1/system/diagnostics | Self-test: Incus socket, Podman socket, disk, memory |

## 8. Host Firewall

nftables rules for host-level and Podman networking. Incus VM/CT firewalling uses Incus Network ACLs (managed through the Incus proxy).

| Method | Endpoint                   | Description               |
| ------ | -------------------------- | ------------------------- |
| GET    | /api/v1/firewall/host      | List host nftables rules  |
| POST   | /api/v1/firewall/host      | Create host firewall rule |
| DELETE | /api/v1/firewall/host/{id} | Delete host firewall rule |

## 9. Audit

Query systemd journal for audit entries (ADR-019).

| Method | Endpoint             | Description                                                 |
| ------ | -------------------- | ----------------------------------------------------------- |
| GET    | /api/v1/audit        | Query audit log (filters: user, since, until, method, path) |
| GET    | /api/v1/audit/export | Export as CSV or JSON                                       |

## 10. Notifications

In-dashboard notification channels.

| Method | Endpoint                                 | Description                                          |
| ------ | ---------------------------------------- | ---------------------------------------------------- |
| GET    | /api/v1/notifications/channels           | List notification channels                           |
| POST   | /api/v1/notifications/channels           | Create channel (Discord, Slack, email, Gotify, ntfy) |
| DELETE | /api/v1/notifications/channels/{id}      | Delete channel                                       |
| POST   | /api/v1/notifications/channels/{id}/test | Send test notification                               |

## 11. Infrastructure

| Method | Endpoint       | Description                                          |
| ------ | -------------- | ---------------------------------------------------- |
| GET    | /api/v1/health | No auth. Returns 200 if hellingd is running.         |
| GET    | /api/v1/events | SSE stream: aggregates Incus events + Helling events |

---

## Error Codes

| Code             | HTTP Status | Description                                  |
| ---------------- | ----------- | -------------------------------------------- |
| AUTH_ERROR       | 401         | Invalid credentials, expired token           |
| FORBIDDEN        | 403         | Insufficient permissions                     |
| NOT_FOUND        | 404         | Resource not found                           |
| VALIDATION_ERROR | 422         | Invalid request body                         |
| RATE_LIMITED     | 429         | Too many requests                            |
| INTERNAL_ERROR   | 500         | Internal server error (details never leaked) |
| INCUS_ERROR      | 502         | Incus socket unreachable or returned error   |
| PODMAN_ERROR     | 502         | Podman socket unreachable or returned error  |
| BMC_ERROR        | 502         | BMC/IPMI communication error                 |

---

## Proxy Paths

### Incus

`/api/incus/*` → forwards to Incus REST API at `/var/lib/incus/unix.socket`

All Incus API operations are available: instances, storage, networks, profiles, projects, cluster, images, operations, events, metrics, warnings, certificates. See [Incus REST API documentation](https://linuxcontainers.org/incus/docs/main/rest-api-spec/).

The proxy adds `?project=<user-project>` for RBAC scoping (non-admin users see only their project's resources).

### Podman

`/api/podman/*` → forwards to Podman libpod API at `/run/podman/podman.sock`

All Podman API operations are available: containers, pods, images, volumes, networks, secrets, system. See [Podman API documentation](https://docs.podman.io/en/latest/_static/api.html).

### Cloud Hypervisor

`/api/ch/{name}/*` → forwards to CH REST API at `/run/ch-{name}/api.sock` (per-VM Unix socket)

Full CH REST API per microVM. See [Cloud Hypervisor API documentation](https://www.cloudhypervisor.org/docs/api/).

MicroVM lifecycle (Helling-owned, not CH-proxied):

| Method | Endpoint                | Description                                         |
| ------ | ----------------------- | --------------------------------------------------- |
| GET    | /api/v1/microvms        | List microVMs (name, PID, status, socket path)      |
| POST   | /api/v1/microvms        | Create microVM (spawn CH process, record in SQLite) |
| DELETE | /api/v1/microvms/{name} | Delete microVM (SIGTERM CH process, remove socket)  |
