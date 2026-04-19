# Helling Architecture

Helling is an OS. Boot the ISO, answer 3 questions (hostname, disk, admin password), and it's running. Incus and Podman ship in the ISO. systemd manages everything. There is one deployment model.

## System Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                        Web Browser                              │
│  React 19 + Ant Design Pro + refine                             │
│  Three panels: resource tree │ tabbed detail │ task log          │
└────────────────────────┬───────────────────────────────────────┘
                         │ HTTPS :8006
┌────────────────────────▼───────────────────────────────────────┐
│                helling-proxy (non-root)                          │
│  TLS (self-signed on first boot / ACME) │ serves web/dist/      │
│  Proxies /api/* to hellingd Unix socket                         │
└────────────────────────┬───────────────────────────────────────┘
                         │ Unix socket
┌────────────────────────▼───────────────────────────────────────┐
│                   hellingd (root, systemd)                       │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Proxy Layer                                                │  │
│  │  /api/incus/* → /var/lib/incus/unix.socket                │  │
│  │  /api/podman/* → /run/podman/podman.sock                  │  │
│  │                                                            │  │
│  │  Before forwarding:                                        │  │
│  │    1. Validate JWT                                         │  │
│  │    2. Map user → Incus project (?project= param)          │  │
│  │    3. Log to journal (async)                               │  │
│  │    4. Auto-snapshot before destructive ops                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Helling Handlers (~60 endpoints)                           │  │
│  │  Auth: PAM + JWT + TOTP + API tokens                      │  │
│  │  Users: PAM CRUD                                           │  │
│  │  Schedules: systemd timer management                       │  │
│  │  Webhooks: HMAC event delivery                             │  │
│  │  BMC: bmclib power/sensors/SEL                             │  │
│  │  K8s: CAPN cluster provisioning                            │  │
│  │  System: config, upgrade, diagnostics                      │  │
│  │  Host Firewall: nft CLI (for Podman networking)            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ State (SQLite — Helling state only)                        │  │
│  │  users, tokens, sessions, schedules, webhooks,             │  │
│  │  webhook_deliveries, bmc_servers, k8s_clusters,            │  │
│  host_firewall_rules, notifications, microvm_instances     │  │
│  │                                                            │  │
│  │  NO instance state, NO storage state, NO network state     │  │
│  │  (Incus and Podman are the source of truth for their data) │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
                         │
┌────────────────────────▼───────────────────────────────────────┐
│                   Debian 13 "Trixie" (ISO)                      │
│  incusd ── QEMU/KVM ── LXC ── OCI ── ZFS ── OVN ── Cowsql    │
│  podman ── compose ── pods ── containers ── registries          │
│  cloud-hypervisor ── per-VM sockets ── hellingd process mgr     │
│  nftables ── smartmontools ── systemd ── AppArmor ── chrony     │
│  bmclib (optional) ── CAPN ── distrobuilder                     │
└────────────────────────────────────────────────────────────────┘
│  Bare Metal: BMC/IPMI/Redfish via bmclib (power, sensors, KVM) │
└────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Proxied request (instance list, container start, storage pool create, etc.)

```
Browser → helling-proxy (TLS)
  → hellingd Unix socket
    → JWT validation
    → User → Incus project mapping (RBAC)
    → Audit log (slog → systemd journal, async)
    → httputil.ReverseProxy → Incus/Podman Unix socket
    → Native upstream response → pass through to browser
```

### Helling request (login, create schedule, add BMC, etc.)

```
Browser → helling-proxy (TLS)
  → hellingd Unix socket
    → JWT validation (except public endpoints: login, setup, health)
    → Helling handler
    → SQLite / systemd / bmclib / nft CLI
    → Helling envelope response {data, error}
```

### WebSocket (console, exec)

```
Browser → helling-proxy (TLS, WebSocket upgrade)
  → hellingd Unix socket (WebSocket upgrade)
    → JWT validation
    → httputil.ReverseProxy with WebSocket support
    → Incus/Podman WebSocket endpoint
    → Bidirectional stream (SPICE VGA, serial terminal, exec PTY)
```

## What hellingd Implements vs. What It Proxies

### Implements (Helling-specific, ~60 endpoints)

| Domain        | Endpoints                                       | Why                                                                      |
| ------------- | ----------------------------------------------- | ------------------------------------------------------------------------ |
| Auth          | login, setup, refresh, logout, TOTP, API tokens | Helling auth layer — Incus/Podman don't have user management             |
| Users         | CRUD                                            | PAM user management — maps to Incus projects for RBAC                    |
| Schedules     | CRUD                                            | systemd timer management — Incus doesn't have scheduling                 |
| Webhooks      | CRUD                                            | HMAC event delivery — Incus has events but no webhooks                   |
| BMC           | CRUD, power, sensors                            | bmclib integration — Incus doesn't manage BMC                            |
| K8s           | create, list, delete, kubeconfig                | CAPN cluster provisioning — Incus provides VMs, Helling orchestrates K8s |
| System        | info, config, upgrade, diagnostics              | Helling system management                                                |
| Host Firewall | CRUD                                            | nftables host rules via nft CLI — Incus ACLs handle VM/CT firewalling    |
| Health        | health check                                    | hellingd + Incus + Podman status                                         |
| Events        | SSE stream                                      | Aggregates Incus events + Helling events                                 |

### Proxies (upstream, everything else)

- **Incus:** Instances, storage, networks, profiles, projects, cluster, images, operations, events, metrics, warnings, certificates — full Incus REST API at `/api/incus/1.0/*`
- **Podman:** Containers, pods, images, volumes, networks, secrets, system — full Podman libpod API at `/api/podman/v5.0/libpod/*`
- **Cloud Hypervisor:** Full CH REST API per microVM at `/api/ch/{name}/*`, routed to per-VM Unix socket at `/run/ch-{name}/api.sock`

## Compute: Four Workload Types

| Type                    | Engine           | Boot   | Use Case                                   |
| ----------------------- | ---------------- | ------ | ------------------------------------------ |
| VMs (QEMU/KVM)          | Incus            | 2-5s   | Full isolation, Windows, different kernels |
| System Containers (LXC) | Incus            | <1s    | Lightweight Linux, shared kernel           |
| App Containers (OCI)    | Podman (host)    | <1s    | Docker-compatible, compose, pods           |
| microVMs                | Cloud Hypervisor | <100ms | Ephemeral workloads, CI runners, sandboxes |

All four are always available. The ISO ships with Incus, Podman, and Cloud Hypervisor.

## Storage

Incus manages pools: ZFS, LVM, Btrfs, dir, Ceph, NFS, iSCSI. Dashboard shows pool cards, SMART per disk, ZFS status. All storage operations go through the Incus proxy. Podman volumes managed through the Podman proxy.

## Networking

Incus manages networks: bridge, macvlan, sriov, OVN. Incus Network ACLs handle VM/CT firewalling. Host-level nftables rules for Podman networking managed by Helling (ADR-018: shell out to `nft`). All Incus network operations go through the proxy.

## Clustering

Incus built-in Raft (Cowsql). Cluster operations go through the Incus proxy. Helling adds: BMC-based HA fencing (detect offline node → power-off via bmclib → Incus evacuates workloads).

## Auth

PAM → JWT with project claims. TOTP + WebAuthn + recovery codes. API tokens. Rate limiting (5 failures → 15min lockout). Audit logging via systemd journal (ADR-019).

## Tags

Stored as `user.tag.*` config keys on Incus resources (ADR-020). Cluster-synced automatically by Incus. Podman resources use `helling.tag.*` labels. No SQLite tag tables.

## Scheduling

Backup and snapshot schedules managed via systemd timers (ADR-017). No in-process cron engine. `Persistent=true` catches missed runs after reboot.

## Audit

Every API mutation logged via slog to systemd journal (ADR-019). Queryable via `journalctl -t hellingd`. Dashboard audit page shells out to journalctl with filters.

## Boot Sequence

```
1. BIOS/UEFI → GRUB → Linux kernel → systemd
2. incusd starts → loads cluster state, restores instances
3. podman socket activated on first request
4. hellingd starts:
   a. Validate helling.yaml (viper)
   b. Open + auto-migrate SQLite (Helling state only)
   c. Verify Incus socket exists (required — fail if not)
   d. Verify Podman socket exists (required — fail if not)
   e. Verify cloud-hypervisor binary exists (required — fail if not)
   f. Load microVM instance table from SQLite (resume CH processes for any recorded running instances)
   g. Open Unix socket → register proxy + handlers → serve
5. helling-proxy starts:
   a. Generate self-signed cert if none exists
   b. Serve React SPA on :8006, proxy /api/* to hellingd socket
```

## Upgrade Path

```
helling system upgrade:
  1. Check for updates (GitHub Releases APT source — ADR-025)
  2. Download .deb packages
  3. Verify checksums + Cosign signatures
  4. Stop helling-proxy
  5. Stop hellingd
  6. Backup SQLite to /var/lib/helling/backups/
  7. apt install helling (replaces binaries, runs postinst)
  8. Start hellingd (auto-migrates schema if needed)
  9. Health check (GET /health)
  10. If failed: rollback (restore previous .deb + DB backup)
  11. Start helling-proxy

helling system upgrade --rollback:
  Restore previous .deb packages + DB backup → restart services
```

## Filesystem

```
/etc/helling/helling.yaml                Config (viper, hot-reload)
/etc/helling/certs/                      TLS certificates
/etc/systemd/system/helling-*.timer      Backup/snapshot schedules
/etc/systemd/system/helling-*.service    Schedule execution units
/var/lib/helling/helling.db              SQLite (Helling state only)
/var/lib/helling/backups/                DB backups for rollback
/var/lib/helling/templates/              App template definitions (compose files)
/var/log/journal/                        Audit logs (via systemd journal)
/opt/helling/web/                        React dashboard (web/dist/)
```

## Dependencies

hellingd has 6 Go dependencies (see ADR-014, ADR-018):

```
github.com/go-chi/chi/v5          # HTTP router
github.com/golang-jwt/jwt/v5      # JWT
github.com/spf13/viper            # Config
github.com/bmc-toolbox/bmclib/v2  # BMC (optional build tag)
gorm.io/gorm                      # SQLite ORM
gorm.io/driver/sqlite             # SQLite driver
```

Everything else is handled by the proxy (Incus/Podman), systemd (scheduling, audit), and CLI tools (nft, smartctl, systemctl).
