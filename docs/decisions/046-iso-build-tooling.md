# ADR-046: ISO Build Tooling

> Status: Accepted (2026-04-21)

## Context

ADR-021 committed to a classical installer ISO flow: "Boot the Helling ISO. Answer 3 questions (hostname, disk, admin password) and it's running." Users burn the ISO to a USB stick, boot, install to disk, reboot into the installed system.

Two candidates:

- **live-build** (Debian Live project): produces hybrid BIOS+UEFI ISO9660 installer images with preseed automation. The tool used by Debian Live, Kali, Tails, Parrot, and most custom Debian installers.
- **mkosi** (systemd project): produces raw GPT disk images, UKIs, and ESPs. Designed for pre-installed image workflows ("dd the image to a disk, boot, first-boot provisioning"). Does not natively output ISO9660 hybrid installer images.

## Decision

Use **live-build** to produce the Helling installer ISO.

Rationale:

- ADR-021 explicitly describes a classical installer flow. live-build is the idiomatic Debian tool for that flow. mkosi is a better tool for a different pattern (pre-installed image + first-boot wizard) that ADR-021 did not choose.
- live-build integrates with Debian preseed for unattended install automation — directly answers the "3 questions" contract in ADR-021.
- Debian-native tool, Debian-packaged, maintained by the debian-live team. Matches ADR-002.
- Output is a hybrid ISO9660 bootable on BIOS and UEFI, dd-able to USB. One artifact, many use cases.
- Preseed + first-boot hooks handle Helling-specific first-boot work (create a one-time setup token, start hellingd, and let the browser/CLI setup flow create the first admin).

Rejected alternatives:

- **mkosi**: produces raw GPT disk images, not ISO9660 installer ISOs. Adopting mkosi would require also revising ADR-021 to a "pre-installed image" pattern, which has different trade-offs (larger download ~4-8 GB vs installer ~1-2 GB, different first-boot UX, no classical "installer asks questions" flow). Out of scope for this ADR.
- **debian-cd / simple-cdd**: older, less maintained, thinner community than live-build for custom installer ISOs.

## Consequences

**Easier:**

- Standard Debian installer look and feel (familiar to any Debian operator)
- Preseed automation is a well-documented path
- Hybrid ISO boots on BIOS and UEFI without separate artifacts
- Smaller download than a pre-installed image

**Harder:**

- live-build's configuration surface is large; Helling maintains a focused `auto/config` + `config/package-lists/*.list.chroot` set
- First-boot cannot fully provision from the installer — remaining work happens on the running target system via `first-boot.target` per `docs/spec/first-boot.md`

## Workflow

```bash
# In release CI (after .deb packages are published to the APT repo per ADR-045)
lb config \
  --distribution trixie \
  --architectures "amd64 arm64" \
  --bootloader "grub-pc grub-efi" \
  --iso-application "Helling" \
  --iso-volume "HELLING v${HELLING_VERSION}" \
  --binary-images iso-hybrid

# Add Helling APT repo as a build-time source
cat > config/archives/helling.list.chroot <<EOF
deb [signed-by=/usr/share/keyrings/helling-archive-keyring.gpg] https://bizarre-industries.github.io/helling helling-v1 main
EOF

# Add Zabbly Incus repo (ADR-002)
cat > config/archives/zabbly.list.chroot <<EOF
deb https://pkgs.zabbly.com/incus/stable trixie main
EOF

# Helling packages to install
cat > config/package-lists/helling.list.chroot <<EOF
helling
hellingd
incus
podman
caddy
EOF

# Preseed for unattended installer
cp preseed/helling.cfg config/includes.binary/preseed.cfg

lb build
# Output: live-image-amd64.hybrid.iso
```

## References

- ADR-021 (ISO-only installation)
- ADR-002 (Debian-first platform)
- ADR-025 (APT repository via GitHub Pages)
- ADR-045 (reprepro as repo tool)
- `docs/spec/first-boot.md`
- `docs/design/full-automation-pipeline.md` Layer 23
