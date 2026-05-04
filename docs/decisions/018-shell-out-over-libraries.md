# ADR-018: Shell Out Over Go Libraries for Host Operations

> Status: Accepted
>
> Amended 2026-05-04: v0.1 keeps only the structured journal emission exception. Root-level systemd unit mutation is deferred until the narrow helper path is implemented and reviewed; no broad DBus/polkit grant is installed.

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

One narrow exception to the shell-out-first rule is currently active:

**Exception 1: Structured journal emission.** `go-systemd/v22/journal` (built without cgo — pure Go client of the journal socket protocol) is used **only** for emitting structured journal records with `HELLING_*` indexed fields (see ADR-019). Shell-out to `systemd-cat` does not support structured fields at journal-write time, and `log/slog` writing to stderr → journald via `StandardOutput=journal` emits records without indexed fields, breaking the audit query contract in ADR-019.

Scope limits for this exception:

- Import is limited to `github.com/coreos/go-systemd/v22/journal` (the journal emission package only; **not** `dbus`, `sdjournal`, or `util` packages from the same module).
- Build must be cgo-free: `CGO_ENABLED=0` applies to this package; the journal client uses the socket protocol, not libsystemd.
- Journal **reads** remain shell-out (`journalctl --output=json`), per the main table above. Reading via the library would re-introduce cgo.

**Deferred: systemd unit management.** v0.1 does not import a DBus client or install a polkit rule for hellingd. Future schedule CRUD must use the narrow helper path from ADR-050 and land with its own security review.

### Still explicitly forbidden

- `google/nftables` — use `nft --json` shell-out.
- `lxc/incus/v6` Go bindings — use the v0.1 admin-only proxy path to the restricted Incus user socket; delegated HTTPS loopback is deferred with ADR-024/036.
- `containers/podman/v5` Go bindings — use the proxy to the Podman socket (ADR-014).
- `go-co-op/gocron` — use systemd timers (ADR-017).
- Any SMART disk library — use `smartctl --json` shell-out.

## Consequences

**Easier:**

- Fewer Go dependencies (remove `google/nftables`, avoid full `coreos/go-systemd`, avoid Incus/Podman Go SDKs)
- Debuggable: `nft --json list ruleset` works identically from shell and from Go
- Stable interfaces: CLI tools have stronger backward-compatibility guarantees than Go libraries
- hellingd go.mod stays small (~10-12 deps including the narrow exception above)

**Harder:**

- Error handling requires parsing stderr in addition to exit codes
- Performance: `exec.Command` has more overhead than a library call (but these are infrequent operations)
- Testing: must mock or have real tools available in test environment
- Path assumptions: tools must be in PATH (guaranteed on the ISO, but worth documenting)
- Future ADRs that want to expand host-operation imports must land an explicit widening of the exception.

## References

- ADR-017 (systemd timers — future helper path)
- ADR-019 (journal over sqlite audit — uses `go-systemd/v22/journal` exception)
- ADR-050 (hellingd non-root — defers systemd mutation until helper review)
