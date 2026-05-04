# ADR-050: hellingd runs as non-root with deferred privileged systemd helper

> Status: Accepted (2026-04-20); amended by ADR-054 for the default
> `incus-admin` prohibition.

## Context

Prior text in `docs/decisions/017-systemd-timers-over-cron.md`, `docs/standards/security.md` §1+§2, and `docs/spec/systemd-units.md` assumed hellingd runs as root because it needs to:

- Write timer/service unit files under `/etc/systemd/system/` (ADR-017)
- Call `systemctl daemon-reload`, `enable`, `start`, `stop` (ADR-017)
- Access the Podman socket (`/run/podman/podman.sock`) and the Incus HTTPS listener credentials
- Shell out to `nft`, `smartctl`, `journalctl` (ADR-018)
- Emit journal records with structured fields (`HELLING_USER`, `HELLING_ACTION`, etc.)

Running hellingd as root makes a web-facing compromise immediately full-system compromise. It also invalidates the systemd hardening profile in `docs/spec/systemd-units.md` — the hardening assumes the daemon doesn't need to write outside its own state directories, yet ADR-017 as written requires `/etc/systemd/system/` write access.

Debian provides DBus/polkit patterns for daemons that need a specific slice of systemd capability, but v0.1 deliberately avoids installing a broad unit-management grant. Schedule mutation waits for a narrow helper that can be reviewed as a separate privileged surface.

## Decision

**hellingd runs as a dedicated non-root system user `helling` in group `helling`.**

Systemd interaction is deferred until a narrow privileged-helper implementation is present. The v0.1 ISO does not ship a polkit rule granting `hellingd` `org.freedesktop.systemd1.manage-units` rights.

### User, group, and filesystem layout

```text
User:   helling (system user, no shell, home = /var/lib/helling)
Group:  helling

Filesystem (created by .deb postinst):
  /etc/helling/                   root:helling 0750
    helling.yaml                  root:helling 0640
    ca/                           helling:helling 0700
      ca.key                      helling:helling 0600
      ca.crt                      helling:helling 0644
    jwt/                          helling:helling 0700
      signing.key                 helling:helling 0600
    age/                          helling:helling 0700
      identity.txt                helling:helling 0600

  /var/lib/helling/               helling:helling 0750
    helling.db                    helling:helling 0600
    backups/                      helling:helling 0700

  /var/log/helling/               helling:helling 0750  (reserved; journal is primary)

  /run/helling/                   helling:helling 0755
    api.sock                      helling:helling-proxy 0660

  /etc/systemd/system/            root:root 0755   (system-owned; hellingd does NOT write here)
```

Unit files under `/etc/systemd/system/helling-*.{timer,service}` are not managed by v0.1 hellingd. Future schedule support must go through the reviewed helper path instead of direct daemon writes.

### Privileged unit management

No broad polkit rule is installed in v0.1. Schedule support must use the narrow helper described below, with filename validation and no arbitrary transient unit body accepted from `hellingd`.

### Group memberships (created during install)

```text
helling  →  member of group `incus`        (restricted Incus access; not `incus-admin`)
helling  →  member of group `helling-proxy` (can place the API socket in the proxy group)
```

### hellingd.service (replaces any `User=root` assumption)

```ini
[Service]
Type=notify
User=helling
Group=helling
SupplementaryGroups=helling-proxy incus

ExecStart=/usr/lib/helling/hellingd

# Hardening (Option B — now viable because no root is needed)
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

ReadWritePaths=/var/lib/helling /var/log/helling /run/helling /etc/helling
# No ReadWritePaths for /etc/systemd/system/ — unit management is deferred.

# Capabilities: none. We do not need CAP_DAC_OVERRIDE under this model.
CapabilityBoundingSet=
AmbientCapabilities=

RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
SystemCallFilter=@system-service @network-io @file-system
SystemCallArchitectures=native
MemoryDenyWriteExecute=true
NoNewPrivileges=true

LimitNOFILE=65535
LimitNPROC=4096
TasksMax=4096

WatchdogSec=30
Restart=on-failure
RestartSec=5
```

### Future helper pattern (deferred for unit CRUD)

```go
// Render a validated helling-*.timer/service into a root-owned staging path.
// Invoke a tiny helper with only the unit basename as argv.
// The helper validates the basename, links/enables/removes only Helling-owned
// units, and never accepts arbitrary unit body text from hellingd.
```

### Chosen path for writing unit definitions

Two options for getting unit bodies onto the filesystem without giving `helling` write access to `/etc/systemd/system/`:

1. **Transient units** — no filesystem writes. Unit lives only as long as it is active, which is fine for `.service`-only use but breaks for `.timer` units because timers need to persist across reboots. Rejected.
2. **Root-owned drop-in directory with group-writable staging** — `/etc/systemd/system/helling-managed/` created by the installer as `root:helling` `0750`, containing a generated `*.timer` and `*.service` per schedule. Symlinks into `/etc/systemd/system/` are created by a tiny helper shipped with Helling that takes a `helling-*.{timer,service}` filename argument. The helper is auditable (~40 LOC), the filename pattern prevents directory traversal, and the group-writable stash is the only filesystem surface `helling` needs for scheduling. **Selected for future implementation.**

Alternative (considered, rejected): a dedicated `hellingsched` privileged helper daemon. Splits the concern cleanly but adds another moving part. Not worth the complexity for v0.1.

### Journal emission under non-root

`go-systemd/v22/journal` (no-cgo build) emits directly to the journal socket `/run/systemd/journal/socket`. This socket is world-writable by default on Debian; no special permission is needed. `HELLING_USER`, `HELLING_ACTION`, and other structured fields work identically under non-root. (Addresses §6.15 exception from the audit.)

## Why

- **Smallest surface area consistent with Helling's requirements.** Non-root `hellingd` with no broad polkit rule keeps the v0.1 daemon inside the documented host boundary; the future helper carries only the specific unit-link operation.
- **Keeps ADR-027/037 split honest.** Previously hellingd was root and only the edge service was non-root. Now hellingd is non-root while Caddy remains the unprivileged edge.
- **Supersedes ADR-029 cleanly.** ADR-029's custom proxy user is replaced by packaged Caddy plus the `helling-proxy` socket group; this ADR establishes the same low-privilege pattern for `hellingd`.
- **Validates the systemd hardening profile.** `docs/spec/systemd-units.md` existed, but was incompatible with `User=root`. Under this ADR, every hardening directive in that file actually bites.
- **No root-operation grant in v0.1.** A compromised `helling` user cannot ask polkit to create or manage arbitrary system units. Schedule support waits for the helper.

## Consequences

**Easier:**

- Web-facing compromise is capability-bounded to Helling's own scope.
- systemd hardening profile (`ProtectSystem=strict`, no capabilities) is legitimate, not theater.
- Audit story for supply-chain compromise is much stronger.

**Harder:**

- `.deb postinst` becomes more complex once schedules land: create user, group, SUID helper binary, and directory ownerships.
- The SUID helper is net-new code that must be kept minimal (~40 LOC) and fuzzed.
- Lima dev environment needs the same user/group/helper setup once schedules land, or a dev-mode shim that runs as root (guard behind an ADR-034 flag).
- Uninstall (`apt purge helling`) must remove the `helling` user and any helper artifacts.

## Follow-up documents to update when this ADR is accepted

- `docs/decisions/017-systemd-timers-over-cron.md` — strike "it runs as root, so this is fine"; replace with the deferred helper path from this ADR.
- `docs/decisions/018-shell-out-over-libraries.md` — add `go-systemd/v22/journal` (no-cgo) as a permitted library for journal emission (this is the "coreos/go-systemd" exception from §6.15 of the audit, but no-cgo and narrow-scope).
- `docs/decisions/029-hellingprox-system-user.md` — add a cross-reference to this ADR; frame both as "all Helling daemons non-root".
- `docs/spec/systemd-units.md` — replace `User=root` assumptions; drop `CAP_DAC_OVERRIDE` from the hardening profile; add `User=helling`, `Group=helling`, `SupplementaryGroups=helling-proxy incus`.
- `docs/spec/caddy.md` — no change, Caddy was already non-root.
- `docs/spec/threat-model.md` — update "hellingd compromise → root" to "hellingd compromise → helling-scoped capability"; update blast radius analysis.
- `docs/standards/security.md` §2 — drop `CapabilityBoundingSet=CAP_DAC_OVERRIDE`; drop `AmbientCapabilities=`; add `User=helling`, `Group=helling`, `SupplementaryGroups=helling-proxy incus`.
- `docs/design/tools-and-frameworks.md` — keep `go-systemd/v22/journal` (no-cgo) for structured journal emission; systemd DBus unit management stays deferred.
- `apps/hellingd/internal/systemd/` (future implementation) — use the reviewed helper path, not a broad polkit `manage-units` grant.
- `debian/` packaging tree — ship the SUID helper and user/group creation in postinst once schedules land.

## References

- ADR-017 (superseded in part — systemctl shell-out replaced with the deferred helper path)
- ADR-018 (expanded — go-systemd/v22/journal exception)
- ADR-027 (two-daemon split)
- ADR-029 (hellingprox non-root — same pattern applied here)
- systemd DBus API: <https://www.freedesktop.org/wiki/Software/systemd/dbus/>
- Cockpit project non-root pattern reference
