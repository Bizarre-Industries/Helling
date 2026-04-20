# ADR-050: hellingd Runs as Non-Root `helling` System User

> Status: Accepted
>
> Resolves: §6.1 from Helling Docs Audit (2026-04-20)

## Context

Earlier ADRs and specs assumed hellingd runs as root (ADR-017, systemd-units.md). The consequence of root execution is a wide blast radius: a successful RCE on hellingd grants root access to the entire system — all user certificates, Incus instance data, Podman containers, cryptographic keys, journalctl audit trail, and capability to manage systemd units.

ADR-027 (two-daemon split) sought to reduce blast radius by splitting edge (Caddy, as `hellingprox` user) from management (hellingd) to contain edge-layer compromise. However, root-level hellingd negates this isolation.

## Decision

**hellingd runs as a non-root system user named `helling`.** The `helling` user is created at install time with no shell and a reserved UID (e.g. 999 on Debian). Group membership and socket permissions grant hellingd access to necessary resources:

- **Incus socket:** `/var/snap/incus/common/lxd/unix.socket` (group `incus` or socket ACL)
- **Podman socket:** `/run/user/$(id -u helling)/podman/podman.sock` — hellingd runs Podman in rootless mode as the `helling` user
- **systemd interaction:** DBus over `org.freedesktop.systemd1.Manager` with a polkit rule restricting the `helling` user to `helling-*` units only
- **Incus trust store:** read/write via Incus socket with per-user client certificates
- **Helling config and state:** directories with appropriate permissions (`/etc/helling`, `/var/lib/helling`, `/var/log/helling`, `/run/helling`)

**systemd unit writes:** Instead of writing to `/etc/systemd/system/` directly, hellingd calls systemd DBus methods:

- `StartTransientUnit` — for one-shot schedule execution
- `EnableUnitFiles` — to add schedule timers to the system
- `DisableUnitFiles` — to remove them

**Polkit rule:** A `/etc/polkit-1/rules.d/helling-systemd.rules` file grants the `helling` user permission to start/enable/disable units matching the pattern `helling-*`.

Example polkit rule (JavaScript):

```javascript
polkit.addRule(function (action, subject) {
  if (
    subject.user == "helling" &&
    action.id.match(
      /^org\.freedesktop\.systemd1\.manage-unit-files$|^org\.freedesktop\.systemd1\.manage-units$/
    ) &&
    action.lookup("unit").match(/^helling-/)
  ) {
    return polkit.Result.YES;
  }
});
```

## Consequences

**Reduced blast radius:**

- RCE on hellingd grants access to the `helling` user's resources only (Incus scope, Podman rootless namespace, Helling's own secrets, systemd `helling-*` units)
- Does not grant root access to the host filesystem, kernel, other users, or host-level systemd units
- DAC_OVERRIDE capability not needed; discretionary access control via group membership and socket ACLs is sufficient

**Increased operational complexity:**

- DBus interaction via systemd API instead of direct file writes (tested; robust, but requires systemd API knowledge)
- Polkit rule must be shipped and installed correctly
- Podman rootless setup (e.g. subuid/subgid ranges) must be configured at install time for the `helling` user

**Unchanged:**

- User client certificates, auth, and proxy behavior (all via existing Incus/Podman APIs)
- Schedule lifecycle (ADR-017 — still uses systemd timers, just writes via DBus instead of files)
- Audit logging (ADR-019 — still journal-based)

## Implementation Notes

1. **Install-time user setup:** Debian package postinst creates the `helling` user with subuid/subgid ranges for rootless Podman.
2. **systemd unit file:** `hellingd.service` specifies `User=helling`, `Group=helling`; removes `CAP_DAC_OVERRIDE` from hardening baseline.
3. **DBus socket access:** Ensure `org.freedesktop.systemd1` socket is world-accessible (standard on modern systemd); no permission changes needed.
4. **Podman integration:** hellingd talks to `$XDG_RUNTIME_DIR/podman/podman.sock` (rootless socket). Deployment docs must note this requires rootless Podman setup for the `helling` user.
5. **Go code:** Use `github.com/godbus/dbus/v5` or equivalent to emit DBus method calls. Helper function: `startSystemdUnit(unitName string, ...properties) error`.

## References

- ADR-017 — systemd Timers Over In-Process Cron (amended to use DBus)
- ADR-027 — Two-Daemon Split (unchanged; this clarifies the daemon-level isolation)
- docs/spec/systemd-units.md — updated with `User=helling`, polkit rule reference
- docs/standards/security.md — updated to reflect non-root model

## Cross-Links

- Platform requirements: docs/standards/infrastructure.md
- Upgrade safety: docs/runbooks/upgrade-rollback.md (CA/key material restoration unchanged)
- Threat model: docs/spec/threat-model.md (trust boundary recalibrated to `helling` user scope)
