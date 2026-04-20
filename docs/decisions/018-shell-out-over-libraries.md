# ADR-018: Shell Out Over Go Libraries for Host Operations

> Status: Accepted
>
> Amended 2026-04-20: Two narrow library exceptions added (`go-systemd/v22/journal` for structured journal emission, `godbus/dbus/v5` for systemd DBus interaction per ADR-050). Exceptions are scoped to specific capabilities; the default remains shell-out.

## Context

Several host-level operations have Go libraries available: `google/nftables` for firewall rules, `coreos/go-systemd` for systemd interaction, various SMART disk libraries. These libraries add dependency weight, version conflicts, and maintenance burden for operations that happen infrequently (firewall rule changes, disk health checks, systemd unit management).

The tools themselves (`nft`, `smartctl`, `systemctl`, `apt`) are already installed on the system — they ship in the ISO. Their CLI interfaces are stable, well-documented, and output structured data (JSON where available).

## Decision

For infrequent host operations, shell out to CLI tools instead of importing Go libraries:

| Operation             | Tool                             | Output format |
| --------------------- | -------------------------------- | ------------- |
| Host firewall rules   | `nft --json list ruleset`        | JSON          |
| SMART disk health     | `smartctl --json --all /dev/sdX` | JSON          |
| Package updates       | `apt`                            | text          |
| ZFS pool status       | `zpool status -p`                | text          |
| LVM details           | `lvs --reportformat json`        | JSON          |
| Disk wiping           | `wipefs`                         | text          |
| Journal query (reads) | `journalctl --output=json`       | JSON          |

Implementation pattern:

```go
func (s *FirewallService) ListRules() ([]Rule, error) {
    out, err := exec.CommandContext(ctx, "nft", "--json", "list", "table", "inet", "helling").CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("firewall.ListRules: %w", err)
    }
    var result nftResult
    if err := json.Unmarshal(out, &result); err != nil {
        return nil, fmt.Errorf("firewall.ListRules: parse nft output: %w", err)
    }
    return result.toRules(), nil
}
```

### Exceptions

Two narrow exceptions to the shell-out-first rule, both introduced to support requirements that shell-out cannot meet:

**Exception 1: Structured journal emission.** `go-systemd/v22/journal` (built without cgo — pure Go client of the journal socket protocol) is used **only** for emitting structured journal records with `HELLING_*` indexed fields (see ADR-019). Shell-out to `systemd-cat` does not support structured fields at journal-write time, and `log/slog` writing to stderr → journald via `StandardOutput=journal` emits records without indexed fields, breaking the audit query contract in ADR-019.

Scope limits for this exception:

- Import is limited to `github.com/coreos/go-systemd/v22/journal` (the journal emission package only; **not** `dbus`, `sdjournal`, or `util` packages from the same module).
- Build must be cgo-free: `CGO_ENABLED=0` applies to this package; the journal client uses the socket protocol, not libsystemd.
- Journal **reads** remain shell-out (`journalctl --output=json`), per the main table above. Reading via the library would re-introduce cgo.

**Exception 2: systemd DBus for unit management.** `godbus/dbus/v5` is used for calling `org.freedesktop.systemd1` operations per ADR-050 (non-root hellingd uses DBus + polkit instead of `sudo systemctl`). Shell-out to `systemctl` is not viable for hellingd because:

- The `helling` system user cannot call `systemctl enable` / `disable` for system units without polkit mediation.
- The polkit policy (ADR-050) authorizes the DBus call path, not the `systemctl` CLI path.

Scope limits for this exception:

- Only `org.freedesktop.systemd1.ManagerInterface` and `org.freedesktop.systemd1.Unit` are called.
- All DBus calls are gated by the polkit rule in ADR-050 (`subject.user == "helling"` and `unit.match(/^helling-.*\.(service|timer)$/)`).
- SUID helper `/usr/lib/helling/helling-unit-link` (introduced in ADR-050) handles `systemctl link` on behalf of hellingd. The helper itself shells out to `systemctl` because it runs as root.

### Still explicitly forbidden

- `google/nftables` — use `nft --json` shell-out.
- `lxc/incus/v6` Go bindings — use the proxy to Incus HTTPS loopback (ADR-014).
- `containers/podman/v5` Go bindings — use the proxy to the Podman socket (ADR-014).
- `go-co-op/gocron` — use systemd timers (ADR-017).
- Any SMART disk library — use `smartctl --json` shell-out.

## Consequences

**Easier:**

- Fewer Go dependencies (remove `google/nftables`, avoid full `coreos/go-systemd`, avoid Incus/Podman Go SDKs)
- Debuggable: `nft --json list ruleset` works identically from shell and from Go
- Stable interfaces: CLI tools have stronger backward-compatibility guarantees than Go libraries
- hellingd go.mod stays small (~10-12 deps including the two narrow exceptions above)

**Harder:**

- Error handling requires parsing stderr in addition to exit codes
- Performance: `exec.Command` has more overhead than a library call (but these are infrequent operations)
- Testing: must mock or have real tools available in test environment
- Path assumptions: tools must be in PATH (guaranteed on the ISO, but worth documenting)
- Two exceptions above require vigilance against scope creep — future ADRs that want to expand these imports must land an explicit widening of the exception.

## References

- ADR-017 (systemd timers — uses both exceptions: DBus for CRUD, SUID helper for link)
- ADR-019 (journal over sqlite audit — uses `go-systemd/v22/journal` exception)
- ADR-050 (hellingd non-root — introduces the DBus exception)
