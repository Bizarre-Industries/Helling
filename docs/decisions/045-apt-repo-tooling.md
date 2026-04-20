# ADR-045: APT Repository Tooling

> Status: Accepted (2026-04-21)

## Context

ADR-025 pinned the APT repository hosting layer to GitHub Pages (signed static serving). ADR-025 did not pin the tool that generates the repository files before they're committed to the `gh-pages` branch. Options considered:

- **aptly**: stateful repo manager with BoltDB, supports mirroring/snapshots, has a REST API
- **reprepro**: Debian-wiki-standard stateful manager for single-repo workflows, simpler surface
- **apt-ftparchive + hand-rolled scripts**: stateless, manual pool layout, no DB

## Decision

Use **reprepro** to generate the APT repository files published to GitHub Pages.

Rationale:

- Helling publishes one product repository — no mirror of upstreams, no cross-repo copying. aptly's headline features (snapshot, mirror, REST API) do not apply.
- reprepro is the Debian wiki's recommended tool for exactly this pattern (custom repo served via static hosting like GitHub Pages). See `https://wiki.debian.org/DebianRepository/SetupWithReprepro`.
- Debian-native, Debian-packaged, well-documented, and maintained. Matches ADR-002 (Debian-first).
- Simpler CI workflow: `reprepro includedeb helling-v1 dist/*.deb && reprepro export` + push to gh-pages. No BoltDB state to manage between CI runs.
- The historical reprepro limitation "one version per package per distribution" is not a blocker: set `Limit: -1` in `conf/distributions` to retain all versions in the pool. Rollback to old versions is `apt install helling=1.2.3` against the retained pool.

Rejected alternatives:

- **aptly**: extra state file (BoltDB) that must be cached and restored between CI runs; features unused; historical GnuPG signing quirks reported in 2018 blogs (fixed since but a red flag for a security-sensitive signing step).
- **apt-ftparchive + scripts**: even simpler but requires hand-managing the pool directory layout, Release/InRelease generation, and signing. reprepro gives us the same static output with less bespoke bash.

## Consequences

**Easier:**

- Single-command repo update in CI (`reprepro includedeb ...`)
- No intermediate BoltDB state to cache across CI runs
- Debian-native tool with wide community familiarity

**Harder:**

- No built-in mirror of upstream Debian — not needed for Helling anyway
- No REST API for runtime repo queries — also not needed; queries are static file reads over HTTPS

## Workflow

```bash
# In release CI (after nfpm produces .deb files to dist/)
mkdir -p gh-pages-repo/conf
cat > gh-pages-repo/conf/distributions <<EOF
Origin: Helling
Label: Helling
Codename: helling-v1
Architectures: amd64 arm64
Components: main
Description: Helling APT repository
SignWith: $HELLING_APT_GPG_KEY_ID
Limit: -1
EOF

cd gh-pages-repo
for deb in ../dist/*.deb; do
  reprepro includedeb helling-v1 "$deb"
done
reprepro export

# Push resulting dists/ and pool/ to gh-pages branch
```

## References

- ADR-025 (GitHub Pages as APT hosting layer)
- ADR-002 (Debian-first platform)
- `https://wiki.debian.org/DebianRepository/SetupWithReprepro`
- `docs/standards/infrastructure.md` §Release pipeline
- `docs/design/full-automation-pipeline.md` Layer 24
