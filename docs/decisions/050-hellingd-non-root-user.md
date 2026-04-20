# ADR-050: hellingd runs as non-root via DBus + polkit for systemd interaction

> Status: Accepted (2026-04-20)

## Context

Prior text in `docs/decisions/017-systemd-timers-over-cron.md`, `docs/standards/security.md` §1+§2, and `docs/spec/systemd-units.md` assumed hellingd runs as root because it needs to:

- Write timer/service unit files under `/etc/systemd/system/` (ADR-017)
- Call `systemctl daemon-reload`, `enable`, `start`, `stop` (ADR-017)
- Access the Podman socket (`/run/podman/podman.sock`) and the Incus HTTPS listener credentials
- Shell out to `nft`, `smartctl`, `journalctl` (ADR-018)
- Emit journal records with structured fields (`HELLING_USER`, `HELLING_ACTION`, etc.)

Running hellingd as root makes a web-facing compromise immediately full-system compromise. It also invalidates the systemd hardening profile in `docs/spec/systemd-units.md` — the hardening assumes the daemon doesn't need to write outside its own state directories, yet ADR-017 as written requires `/etc/systemd/system/` write access.

Debian provides a standard non-root path for daemons that need a specific slice of systemd capability: **DBus + polkit**. `systemd1` exposes `ManagerInterface` over the system bus; `polkit` rules can grant a specific user permission to a specific subset of unit-management actions.

This pattern is used by cockpit, snapd, and systemd-homed. It is a battle-tested path for exactly this scenario.

## Decision

**hellingd runs as a dedicated non-root system user `helling` in group `helling`.**

Systemd interaction uses DBus calls (`org.freedesktop.systemd1` via `godbus/dbus/v5`), authorized by a polkit policy file shipped in the Helling `.deb` package.

The polkit policy restricts the `helling` user to managing only units matching `helling-*.service` and `helling-*.timer`. No other unit management is permitted.

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

  /run/helling/                   helling:helling 0750
    hellingd.sock                 helling:helling 0660

  /etc/systemd/system/            root:root 0755   (system-owned; hellingd does NOT write here)
```

Unit files under `/etc/systemd/system/helling-*.{timer,service}` are managed **through DBus**, not through direct filesystem writes.

### Polkit policy (`/usr/share/polkit-1/actions/industries.bizarre.helling.policy`)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE policyconfig PUBLIC "-//freedesktop//DTD PolicyKit Policy Configuration 1.0//EN"
  "http://www.freedesktop.org/standards/PolicyKit/1/policyconfig.dtd">
<policyconfig>
  <vendor>Bizarre Industries</vendor>
  <vendor_url>https://bizarre.industries</vendor_url>

  <action id="industries.bizarre.helling.manage-unit">
    <description>Manage Helling systemd units</description>
    <message>Helling requires permission to manage helling-* systemd units</message>
    <defaults>
      <allow_any>no</allow_any>
      <allow_inactive>no</allow_inactive>
      <allow_active>auth_admin</allow_active>
    </defaults>
    <annotate key="org.freedesktop.policykit.imply">org.freedesktop.systemd1.manage-units</annotate>
  </action>
</policyconfig>
```

### Polkit rule (`/etc/polkit-1/rules.d/50-helling.rules`)

```javascript
polkit.addRule(function (action, subject) {
  if (action.id == "org.freedesktop.systemd1.manage-units" && subject.user == "helling") {
    var unit = action.lookup("unit");
    if (unit && unit.match(/^helling-.*\.(service|timer)$/)) {
      return polkit.Result.YES;
    }
  }
  return polkit.Result.NOT_HANDLED;
});
```

### Group memberships (created during install)

```text
helling  →  member of group `podman`       (reads /run/podman/podman.sock)
helling  →  member of group `systemd-journal`   (queries journal)
helling  →  member of group `incus-admin` (or equivalent)  (loopback HTTPS cert access)
```

### hellingd.service (replaces any `User=root` assumption)

```ini
[Service]
Type=notify
User=helling
Group=helling
SupplementaryGroups=podman systemd-journal

ExecStart=/usr/lib/helling/hellingd

# Hardening (Option B — now viable because no root is needed)
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

ReadWritePaths=/var/lib/helling /var/log/helling /run/helling /etc/helling
# No ReadWritePaths for /etc/systemd/system/ — unit management goes via DBus.

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

### DBus call pattern (replaces direct `systemctl` shell-out for unit CRUD)

```go
import "github.com/godbus/dbus/v5"

// Connect to system bus as the helling user; polkit authorizes the call.
conn, err := dbus.SystemBus()
if err != nil {
    return fmt.Errorf("systemd dbus: %w", err)
}

systemd := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")

// Write the unit file to /etc/systemd/system/ — this step is the issue.
// Solution: unit files for helling-* are owned by root:helling 0640 by the
// .deb postinst creating an empty drop-in directory at
// /etc/systemd/system/helling-managed.d/ that `helling:helling` can write into,
// OR unit file bodies are rendered into transient units via
// StartTransientUnit on the DBus API (no filesystem writes needed at all).
```

### Chosen path for writing unit definitions

Two options for getting unit bodies onto the filesystem without giving `helling` write access to `/etc/systemd/system/`:

1. **Transient units via `StartTransientUnit`** — DBus-only, no filesystem writes. Unit lives only as long as it's active, which is fine for `.service`-only use but **breaks for `.timer` units** because timers need to persist across reboots. Rejected.
2. **Root-owned drop-in directory with group-writable staging** — `/etc/systemd/system/helling-managed/` created by `.deb postinst` as `root:helling` `0750`, containing a generated `*.timer` and `*.service` per schedule. Symlinks into `/etc/systemd/system/` created by a tiny SUID helper shipped in the `.deb` (`/usr/lib/helling/helling-unit-link`, owned `root:helling` mode `4750`) that takes a `helling-*.{timer,service}` filename argument and `systemctl link`s it, with DBus reload. The helper is auditable (~40 LOC), the filename pattern prevents directory traversal, and the group-writable stash is the only filesystem surface `helling` needs for scheduling. **Selected.**

Alternative (considered, rejected): a dedicated `hellingsched` privileged helper daemon. Splits the concern cleanly but adds another moving part. Not worth the complexity for v0.1.

### Journal emission under non-root

`go-systemd/v22/journal` (no-cgo build) emits directly to the journal socket `/run/systemd/journal/socket`. This socket is world-writable by default on Debian; no special permission is needed. `HELLING_USER`, `HELLING_ACTION`, and other structured fields work identically under non-root. (Addresses §6.15 exception from the audit.)

## Why

- **Smallest surface area consistent with Helling's requirements.** DBus + polkit is the standard Debian pattern for exactly this scenario.
- **Keeps ADR-027 (two-daemon split) honest.** Previously hellingd was root and hellingprox was non-root — the split bought limited isolation. Now both are non-root.
- **Makes ADR-029 less of an outlier.** ADR-029 established `hellingprox` as a dedicated low-privilege user. This ADR establishes the same pattern for `hellingd`.
- **Validates the systemd hardening profile.** `docs/spec/systemd-units.md` existed, but was incompatible with `User=root`. Under this ADR, every hardening directive in that file actually bites.
- **No worse than root for the threat model.** A compromised `helling` user can manage `helling-*` units. That's the capability the daemon already needed. It can't suddenly install a rootkit or read `/etc/shadow`.

## Consequences

**Easier:**

- Web-facing compromise is capability-bounded to Helling's own scope.
- systemd hardening profile (`ProtectSystem=strict`, no capabilities) is legitimate, not theater.
- Audit story for supply-chain compromise is much stronger.

**Harder:**

- `.deb postinst` is more complex: create user, group, polkit policy file, polkit rule, SUID helper binary, directory ownerships.
- The SUID helper is net-new code that must be kept minimal (~40 LOC) and fuzzed.
- Lima dev environment needs the same user/group/polkit setup, or a dev-mode shim that runs as root (guard behind an ADR-034 flag).
- Uninstall (`apt purge helling`) must revert polkit rules and remove the `helling` user.

## Follow-up documents to update when this ADR is accepted

- `docs/decisions/017-systemd-timers-over-cron.md` — strike "it runs as root, so this is fine"; replace with the DBus + polkit + SUID-helper path from this ADR.
- `docs/decisions/018-shell-out-over-libraries.md` — add `go-systemd/v22/journal` (no-cgo) as a permitted library for journal emission (this is the "coreos/go-systemd" exception from §6.15 of the audit, but no-cgo and narrow-scope).
- `docs/decisions/029-hellingprox-system-user.md` — add a cross-reference to this ADR; frame both as "all Helling daemons non-root".
- `docs/spec/systemd-units.md` — replace `User=root` assumptions; drop `CAP_DAC_OVERRIDE` from the hardening profile; add `User=helling`, `Group=helling`, `SupplementaryGroups=podman systemd-journal`.
- `docs/spec/caddy.md` — no change, Caddy was already non-root.
- `docs/spec/threat-model.md` — update "hellingd compromise → root" to "hellingd compromise → helling-scoped capability"; update blast radius analysis.
- `docs/standards/security.md` §1 — strike "(running as root)" from the Podman-socket description; replace with "accessible to the `helling` system user via the `podman` supplementary group (configured during ISO install per ADR-050)".
- `docs/standards/security.md` §2 — drop `CapabilityBoundingSet=CAP_DAC_OVERRIDE`; drop `AmbientCapabilities=`; add `User=helling`, `Group=helling`, `SupplementaryGroups=podman systemd-journal`.
- `docs/design/tools-and-frameworks.md` — add `godbus/dbus/v5` to the hellingd dependency table (for systemd DBus); add `go-systemd/v22/journal` (no-cgo) for structured journal emission.
- `apps/hellingd/internal/systemd/` (implementation) — use DBus client, not `exec.Command("systemctl", ...)`, for unit management.
- `debian/` packaging tree — ship polkit policy, polkit rule, SUID helper, user/group creation in postinst.

## References

- ADR-017 (superseded in part — systemctl shell-out replaced with DBus+polkit)
- ADR-018 (expanded — go-systemd/v22/journal exception)
- ADR-027 (two-daemon split)
- ADR-029 (hellingprox non-root — same pattern applied here)
- Debian polkit documentation: <https://www.freedesktop.org/software/polkit/docs/latest/>
- systemd DBus API: <https://www.freedesktop.org/wiki/Software/systemd/dbus/>
- Cockpit project non-root pattern reference
