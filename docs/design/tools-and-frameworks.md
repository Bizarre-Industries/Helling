# Tools & Frameworks

**Date:** 2026-04-20
**Principle:** Use what the OS gives you. Shell out or proxy instead of importing. Keep Go dependencies minimal and justified. Every import below is traceable to an ADR.

---

## Backend (hellingd)

| Concern                | Tool                                               | What it does                                                                                                                              | ADR                                 |
| ---------------------- | -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| HTTP routing           | `net/http` ServeMux + `huma/v2/humago`             | ServeMux baseline + Huma adapter                                                                                                          | ADR-043                             |
| API types + validation | `danielgtaylor/huma/v2`                            | Validation + OpenAPI 3.1 from Go struct tags                                                                                              | ADR-043                             |
| Auth (PAM)             | `msteinert/pam/v2`                                 | PAM conversation (cgo — requires `libpam0g-dev` on Debian build host); see `docs/spec/pam.md` for the service-name / config-path contract | ADR-030                             |
| Auth (JWT)             | `golang-jwt/jwt/v5`                                | Ed25519-signed access + refresh tokens                                                                                                    | ADR-031                             |
| Auth (TOTP)            | `pquerna/otp`                                      | QR generation, code verify                                                                                                                | —                                   |
| Database               | `database/sql` + `sqlc`                            | Helling state only (SQLite)                                                                                                               | ADR-038                             |
| SQLite driver          | `mattn/go-sqlite3`                                 | cgo-backed SQLite — cgo already mandatory for libpam                                                                                      | ADR-038                             |
| Migrations             | `goose`                                            | Forward-only SQL migrations                                                                                                               | ADR-038                             |
| Password hashing       | `golang.org/x/crypto/argon2`                       | argon2id (RFC 9106 / OWASP baseline)                                                                                                      | ADR-030                             |
| Config                 | `gopkg.in/yaml.v3`                                 | YAML + env vars                                                                                                                           | —                                   |
| BMC                    | `bmc-toolbox/bmclib/v2`                            | IPMI + Redfish (v0.4 feature)                                                                                                             | —                                   |
| HTTP proxy             | `net/http/httputil.ReverseProxy`                   | Proxies to Incus HTTPS and Podman Unix socket                                                                                             | ADR-014, ADR-036                    |
| systemd unit mgmt      | `godbus/dbus/v5` + SUID `helling-unit-link` helper | DBus-based unit management under non-root hellingd                                                                                        | ADR-017, ADR-018 (amended), ADR-050 |
| Journal emit           | `coreos/go-systemd/v22/journal`                    | Structured-field audit emission                                                                                                           | ADR-018 (exception), ADR-019        |
| Audit log reads        | `journalctl -o json` (shell-out)                   | Journal query path for audit API                                                                                                          | ADR-018, ADR-019                    |
| Firewall               | `nft --json` (shell-out)                           | Host nftables rules                                                                                                                       | ADR-018                             |
| Disk health            | `smartctl --json` (shell-out)                      | SMART data                                                                                                                                | ADR-018                             |
| System info            | OS CLI tools (shell-out)                           | CPU, RAM, disk, NICs                                                                                                                      | ADR-018                             |

### hellingd go.mod (target)

Small dependency set centred on stdlib routing, auth, config, SQLite persistence, and two narrow systemd bindings. Everything else is in stdlib, systemd, or CLI tools invoked via `exec.Command`.

### What hellingd does NOT import

| Don't import                          | Use instead                                                      |
| ------------------------------------- | ---------------------------------------------------------------- |
| `lxc/incus/v6`                        | Proxy to Incus HTTPS loopback (no SDK dependency)                |
| `containers/podman/v5`                | Proxy to Podman Unix socket                                      |
| `google/nftables`                     | Shell out to `nft --json`                                        |
| `go-co-op/gocron`                     | systemd timers via DBus                                          |
| `coreos/go-systemd/v22/dbus` (client) | `godbus/dbus/v5` directly for unit lifecycle (ADR-018 amendment) |

Note on the ADR-018 exceptions: shelling out to `systemctl` for every unit operation costs ~50ms per call and produces unstructured output. Two narrow Go imports replace those specific paths only:

1. `godbus/dbus/v5` for unit Start/Stop/Status/list ops (hellingd runs non-root per ADR-050; polkit grants the `helling` user only the unit actions it needs)
2. `coreos/go-systemd/v22/journal` for structured-field audit emission

`systemctl link` still shells out because it requires root — handled via the SUID helper introduced in ADR-050.

### Add when needed (later versions)

| Dependency             | Introduced in | Purpose                                          |
| ---------------------- | ------------- | ------------------------------------------------ |
| `go-webauthn/webauthn` | v0.5+         | WebAuthn ceremony (see `docs/spec/auth-v0.5.md`) |
| `go-ldap/ldap/v3`      | v0.5+         | LDAP bind, search, sync                          |
| `coreos/go-oidc/v3`    | v0.5+         | OIDC discovery, token verify                     |
| `filippo.io/age`       | v0.3+         | Encrypted backup blobs (ADR-039)                 |

---

## CLI (helling)

| Concern           | Tool                                         |
| ----------------- | -------------------------------------------- |
| Command framework | `spf13/cobra`                                |
| Config            | `gopkg.in/yaml.v3` + env loader              |
| API client        | Generated from Helling spec (`oapi-codegen`) |
| Man pages         | `cobra/doc.GenManTree()`                     |
| Shell completions | Cobra built-in                               |

~15 commands. ~800 lines of Go. Users use `incus`/`podman`/`kubectl` directly for anything outside Helling-managed surfaces (ADR-016).

---

## Frontend (web)

| Concern          | Tool                                                      | Notes                                                              |
| ---------------- | --------------------------------------------------------- | ------------------------------------------------------------------ |
| Framework        | React 19                                                  |                                                                    |
| Build            | Vite (SPA, no SSR)                                        |                                                                    |
| Base components  | `antd` v6                                                 | 6.x line, released Nov 2025                                        |
| Admin components | `@ant-design/pro-components@beta` (3.x)                   | 3.x is on the `beta` dist-tag; peer dep `antd: ^6.0.0`             |
| Charts           | `@ant-design/charts`                                      | G2-based, antd-theme integrated                                    |
| Data fetching    | `@tanstack/react-query` via `@hey-api/openapi-ts`         | Generated hooks/options from Helling OpenAPI                       |
| Terminal         | `@xterm/xterm`                                            | Renamed from `xterm.js` in 2024                                    |
| VM VGA console   | `spice-html5` (bundled from Debian `spice-html5` package) | Served as static assets from `/usr/share/spice-html5/`; no npm dep |
| Code editor      | `@uiw/react-codemirror` (CodeMirror 6)                    | Dynamic import                                                     |
| Routing          | `react-router` v7                                         | Package was renamed from `react-router-dom` in v7                  |
| HTTP             | `fetch` wrapper                                           | JWT injection + request-id propagation                             |
| Icons            | `lucide-react`                                            |                                                                    |

### On `pro-components@beta`

pro-components 3.x is published on npm under the `beta` dist-tag (latest: `3.1.12-0`, 2026-03-29). The `latest` tag still points at 2.8.10 which is antd-v5-only. Pin in `package.json` with an explicit version, not `^`, until 3.x promotes to `latest`.

### On `spice-html5` bundling

The `@canonical/spice-html5` npm package does not exist. The published alternatives are outdated or not directly usable in a Vite build. Debian 13 (Trixie) ships `spice-html5 0.3.0-2` as a system package. Helling's build pipeline copies the Debian-packaged asset bundle into `web/public/spice/` at build time (sourced from `/usr/share/spice-html5/` on the build host). Loaded via dynamic import in the VM VGA console tab per ADR-010.

This keeps SPICE as the v0.1 VM console protocol (ADR-010 reasoning stands — Incus `type=vga` is natively SPICE) without taking a fragile npm dependency.

### Frontend does NOT import

| Don't import                      | Use instead                                             |
| --------------------------------- | ------------------------------------------------------- |
| noVNC                             | SPICE browser console is the v0.1 default (ADR-010)     |
| `@canonical/spice-html5`          | Doesn't exist. Use Debian-packaged `spice-html5` bundle |
| `axios` for generated API surface | Fetch client generated by `@hey-api/openapi-ts`         |
| `orval`                           | `@hey-api/openapi-ts`                                   |
| `@refinedev/*`                    | antd v6 + React Query + generated SDK                   |

---

## System Tools (shell out, ADR-018)

| Tool                             | Provides                                       | Invocation                                         |
| -------------------------------- | ---------------------------------------------- | -------------------------------------------------- |
| `nft --json`                     | Host nftables rules                            | `exec.Command("nft", "--json", ...)`               |
| `smartctl --json`                | SMART disk health                              | `exec.Command("smartctl", "--json", "--all", ...)` |
| `systemctl`                      | Unit lifecycle (root-only ops via SUID helper) | Via `/usr/lib/helling/helling-unit-link`           |
| `zpool status`                   | ZFS pool details                               | `exec.Command("zpool", "status", ...)`             |
| `lvs --reportformat json`        | LVM volume details                             | `exec.Command("lvs", "--reportformat", "json")`    |
| `journalctl -t hellingd -o json` | Audit log queries                              | `exec.Command("journalctl", ...)`                  |
| `apt`                            | Package updates                                | `exec.Command("apt", ...)`                         |
| `wipefs`                         | Disk wiping                                    | `exec.Command("wipefs", ...)`                      |
| `podman compose`                 | Compose stack lifecycle                        | `exec.Command("podman", "compose", ...)`           |

All tools ship in the ISO. Stable CLI interfaces. JSON output where available.

---

## Incus: Proxy, Don't Wrap

Previous architecture: 150+ Go handlers wrapping Incus API calls. Each handler decoded HTTP, called Incus Go client, re-encoded response.

Current architecture (ADR-014 + ADR-036): `httputil.ReverseProxy` forwards requests to Incus HTTPS loopback with per-user TLS client cert identity. Zero Go code per Incus endpoint. New Incus features work automatically.

Helling adds on top of the proxy:

- JWT auth + RBAC project scoping (ADR-024)
- Auto-snapshot before destructive operations (compute.md §Auto-Snapshot)
- Tags via Incus `user.tag.*` config keys (ADR-020)
- Audit logging to systemd journal (ADR-019)

**Minimum Incus version: 6.14.0** (earlier versions have CVE-2025-52889, CVE-2025-52890, CVE-2025-4115). See `docs/spec/platform.md` for the full deployment version pinning.

---

## Podman: Proxy, Don't Wrap

Same pattern as Incus. `httputil.ReverseProxy` forwards to Podman socket (`/run/podman/podman.sock`). The `helling` system user is a member of the `podman` group to access the socket (ADR-050). Helling adds compose stack tracking and app template deployment on top.

---

## Base OS

| Component | Version            | Notes                                        |
| --------- | ------------------ | -------------------------------------------- |
| Debian    | 13 (Trixie)        | ADR-002                                      |
| Incus     | ≥ 6.14.0           | CVE-pinned (see platform.md)                 |
| Podman    | Debian-shipped     | socket at `/run/podman/podman.sock`          |
| K3s       | latest stable      | SQLite default, etcd opt-in for HA (ADR-005) |
| Caddy     | 2.x Debian-shipped | ADR-037                                      |
| AppArmor  | Debian default     | MAC layer                                    |
