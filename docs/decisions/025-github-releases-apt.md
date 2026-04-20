# ADR-025: Signed APT repository via GitHub Pages for updates

> Status: Accepted (2026-04-15)

## Context

Helling is ISO-only (ADR-021). Post-install upgrades need an APT-compatible distribution mechanism. Options:

- Signed APT repository generated and published to GitHub Pages
- Self-hosted APT server (aptly / reprepro on a VPS)
- In-place binary download (no package manager integration)

## Decision

Publish `.deb` packages as release artifacts and generate a signed APT repository index on release publication. The canonical public repository is hosted on GitHub Pages.

Repository requirements:

- `Packages` indices per architecture
- `Release` metadata file
- `InRelease` clear-signed metadata (or `Release.gpg` + `Release`)
- Stable repository layout (`dists/<suite>/...`, `pool/...`)

ISO configures an APT source pointing to the GitHub Pages repository on first boot:

```text
deb [signed-by=/usr/share/keyrings/helling-archive-keyring.gpg] https://bizarre-industries.github.io/helling bookworm main
```

Updates:

```bash
apt update && apt install --only-upgrade helling helling-proxy hellingd
```

The management plane (hellingd, helling-proxy) restarts via systemd after upgrade. The OS itself updates via standard Debian security repos + Zabbly (for Incus). These are decoupled.

GitHub Releases remains the release-event trigger and artifact source, while GitHub Pages is the APT repository serving layer.

Signing and key lifecycle:

- Repository metadata is signed with a dedicated APT archive GPG key.
- Key material is stored in GitHub Actions secrets with an encrypted offline backup.
- Public key is distributed in installer artifacts and documented for manual installation.
- Key expiration must be long-lived (multi-year) and reviewed annually.
- Rotation policy: publish new key, dual-sign transition window, update installer keyring, then retire old key.

## Consequences

- APT clients can perform standard `apt update` / `apt upgrade` using signed metadata
- GitHub Pages becomes the package index endpoint; GitHub Releases remains build provenance anchor
- `.deb` packages are still built by nfpm in the release pipeline
- Repository integrity uses APT GPG verification; artifact-level provenance can still use Cosign + SLSA
- OS updates (Debian security, Zabbly Incus) are independent of Helling updates
- "Update" button in dashboard UI runs: `apt update && apt install --only-upgrade helling helling-proxy hellingd`
- On upgrade: only management plane restarts — running VMs/containers unaffected
