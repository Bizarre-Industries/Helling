# ADR-025: GitHub Releases as APT source for updates

> Status: Accepted (2026-04-15)

## Context

Helling is ISO-only (ADR-021). Post-install upgrades need a mechanism. Options:

- Self-hosted APT server (aptly / reprepro on a VPS)
- GitHub Releases as APT source (no server infrastructure)
- In-place binary download (no package manager integration)

## Decision

Publish `.deb` packages as GitHub Release assets built by `nfpm`. The ISO configures an APT source pointing to GitHub Releases on first boot:

```
deb [trusted=yes] https://github.com/Bizarre-Industries/helling/releases/download/ /
```

Updates:

```bash
apt update && apt install --only-upgrade helling helling-proxy hellingd
```

The management plane (hellingd, helling-proxy) restarts via systemd after upgrade. The OS itself updates via standard Debian security repos + Zabbly (for Incus). These are decoupled.

No aptly/reprepro infrastructure required. GitHub Releases provides download hosting, versioning, and artifact provenance (Cosign + SLSA signatures on each asset).

## Consequences

- Zero APT server infrastructure to operate
- GitHub Releases is the single source of truth for Helling package versions
- `.deb` packages built by nfpm in GoReleaser pipeline
- Cosign + SLSA signatures on release assets allow `apt` to verify before install
- OS updates (Debian security, Zabbly Incus) are independent of Helling updates
- "Update" button in dashboard UI runs: `apt update && apt install --only-upgrade helling helling-proxy hellingd`
- On upgrade: only management plane restarts — running VMs/containers unaffected
