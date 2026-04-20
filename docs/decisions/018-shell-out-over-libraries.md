# ADR-018: Shell Out Over Go Libraries for Host Operations

> Status: Accepted

## Context

Several host-level operations have Go libraries available: `google/nftables` for firewall rules, `coreos/go-systemd` for systemd interaction, various SMART disk libraries. These libraries add dependency weight, version conflicts, and maintenance burden for operations that happen infrequently (firewall rule changes, disk health checks, systemd unit management).

The tools themselves (`nft`, `smartctl`, `systemctl`, `apt`) are already installed on the system — they ship in the ISO. Their CLI interfaces are stable, well-documented, and output structured data (JSON where available).

## Decision

For infrequent host operations, shell out to CLI tools instead of importing Go libraries:

| Operation                | Tool                             | Output format |
| ------------------------ | -------------------------------- | ------------- |
| Host firewall rules      | `nft --json list ruleset`        | JSON          |
| SMART disk health        | `smartctl --json --all /dev/sdX` | JSON          |
| systemd timer management | `systemctl`                      | text/JSON     |
| Package updates          | `apt`                            | text          |
| ZFS pool status          | `zpool status -p`                | text          |
| LVM details              | `lvs --reportformat json`        | JSON          |
| Disk wiping              | `wipefs`                         | text          |

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

## Consequences

**Easier:**

- Fewer Go dependencies (remove `google/nftables`, avoid `coreos/go-systemd`)
- Debuggable: `nft --json list ruleset` works identically from shell and from Go
- Stable interfaces: CLI tools have stronger backward-compatibility guarantees than Go libraries
- hellingd go.mod stays at 6 dependencies

**Harder:**

- Error handling requires parsing stderr in addition to exit codes
- Performance: exec.Command has more overhead than a library call (but these are infrequent operations)
- Testing: must mock or have real tools available in test environment
- Path assumptions: tools must be in PATH (guaranteed on the ISO, but worth documenting)
