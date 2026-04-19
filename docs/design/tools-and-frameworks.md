# Tools & Frameworks

**Date:** 2026-04-15
**Principle:** Use what the OS gives you. Shell out instead of importing. Proxy instead of reimplementing. Six Go dependencies.

---

## Backend (hellingd)

| Concern | Tool | What it does | Custom code |
|---|---|---|---|
| HTTP routing | `go-chi/chi/v5` | Middleware chain, URL params | ~100 lines (router) |
| API types + validation | `oapi-codegen` (strict-server) | Types + router from spec | Handler implementations only |
| Auth (PAM) | `msteinert/pam` | PAM conversation | ~100 lines |
| Auth (JWT) | `golang-jwt/jwt/v5` | Token create/verify | ~200 lines |
| Auth (TOTP) | `pquerna/otp` | QR generation, code verify | ~100 lines |
| Database | `gorm.io/gorm` + SQLite | Helling state only (ADR-018) | Model definitions |
| Config | `spf13/viper` | YAML + env vars | Config struct |
| BMC | `bmc-toolbox/bmclib/v2` | IPMI, Redfish | Thin wrapper |
| Proxy | `net/http/httputil` | Reverse proxy to Unix sockets | ~300 lines |
| Scheduling | `systemd timers` | Backup/snapshot cron (ADR-017) | Unit file generation |
| Audit | `log/slog` → journal | Structured logging (ADR-019) | Zero — slog is stdlib |
| Firewall | `nft` CLI (shell out) | Host nftables rules (ADR-018) | exec.Command wrapper |
| Disk health | `smartctl` CLI (shell out) | SMART data (ADR-018) | exec.Command wrapper |
| System info | `shirou/gopsutil/v4` | CPU, RAM, disk, NICs | Direct calls |

### hellingd go.mod (target)

```
require (
    github.com/go-chi/chi/v5
    github.com/golang-jwt/jwt/v5
    github.com/spf13/viper
    github.com/bmc-toolbox/bmclib/v2  // optional build tag
    gorm.io/gorm
    gorm.io/driver/sqlite
)
```

Six dependencies. Everything else is stdlib, systemd, or CLI tools.

### What hellingd does NOT import

| Don't import | Use instead |
|---|---|
| `lxc/incus/v6` | Proxy to Incus Unix socket |
| `containers/podman/v5` | Proxy to Podman Unix socket |
| `google/nftables` | Shell out to `nft --json` |
| `go-co-op/gocron` | systemd timers |
| `coreos/go-systemd` | Shell out to `systemctl` |

### Add when needed (later versions)

| Dependency | When | What |
|---|---|---|
| `go-webauthn/webauthn` | v0.5.0 | WebAuthn ceremony |
| `go-ldap/ldap/v3` | v0.5.0 | LDAP bind, search, sync |
| `coreos/go-oidc/v3` | v0.5.0 | OIDC discovery, token verify |

---

## CLI (helling)

| Concern | Tool |
|---|---|
| Command framework | `spf13/cobra` |
| Config | `spf13/viper` |
| API client | Generated from Helling spec (oapi-codegen) |
| Man pages | `cobra/doc.GenManTree()` |
| Shell completions | Cobra built-in |

~15 commands. ~800 lines of Go. Users use `incus`/`podman`/`kubectl` for everything else (ADR-016).

---

## Frontend (web)

| Concern | Tool |
|---|---|
| Components | `antd` v6 |
| Admin components | `@ant-design/pro-components` |
| Charts | `@ant-design/charts` |
| CRUD framework | `@refinedev/core` + `@refinedev/antd` |
| Data fetching | `@tanstack/react-query` (via orval for Helling API) |
| Terminal | `@xterm/xterm` |
| SPICE console | `spice-js` or `spice-html5` (dynamic import) |
| Code editor | `@monaco-editor/react` (dynamic import) |
| Routing | `react-router-dom` v7 |
| HTTP | `axios` (JWT interceptor) |
| Icons | `lucide-react` |

### Frontend does NOT import

| Don't import | Use instead |
|---|---|
| `@novnc/novnc` | SPICE (ADR-010) |

---

## System Tools (shell out, ADR-018)

| Tool | Provides | Invocation |
|---|---|---|
| `nft --json` | Host nftables rules | `exec.Command("nft", "--json", ...)` |
| `smartctl --json` | SMART disk health | `exec.Command("smartctl", "--json", "--all", ...)` |
| `systemctl` | Timer management | `exec.Command("systemctl", ...)` |
| `zpool status` | ZFS pool details | `exec.Command("zpool", "status", ...)` |
| `lvs --reportformat json` | LVM volume details | `exec.Command("lvs", "--reportformat", "json")` |
| `journalctl -t hellingd -o json` | Audit log queries | `exec.Command("journalctl", ...)` |
| `apt` | Package updates | `exec.Command("apt", ...)` |
| `wipefs` | Disk wiping | `exec.Command("wipefs", ...)` |
| `podman compose` | Compose stack lifecycle | `exec.Command("podman", "compose", ...)` |

All tools ship in the ISO. Stable CLI interfaces. JSON output where available.

---

## Incus: Proxy, Don't Wrap

Previous architecture: 150+ Go handlers wrapping Incus API calls. Each handler decoded HTTP, called Incus Go client, re-encoded response.

New architecture (ADR-014): `httputil.ReverseProxy` forwards requests to Incus Unix socket. Zero Go code per Incus endpoint. New Incus features work automatically.

Helling adds on top of the proxy:
- JWT auth + RBAC project scoping
- Auto-snapshot before destructive operations
- Tags via Incus `user.tag.*` config keys (ADR-020)
- Audit logging to systemd journal (ADR-019)

---

## Podman: Proxy, Don't Wrap

Same pattern as Incus. `httputil.ReverseProxy` forwards to Podman socket. Helling adds compose stack tracking and app template deployment on top.
