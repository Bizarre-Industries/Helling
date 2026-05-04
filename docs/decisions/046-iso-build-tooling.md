# ADR-046: ISO Build Tooling

> Status: Accepted (2026-04-21)

## Context

ADR-021 committed to a classical installer ISO flow: "Boot the Helling ISO. Answer 3 questions (hostname, disk, admin password) and it's running." Users burn the ISO to a USB stick, boot, install to disk, reboot into the installed system.

Two candidates:

- **live-build** (Debian Live project): produces ISO9660 installer images with preseed automation. On amd64 the image is hybrid BIOS+UEFI; on arm64 the image is UEFI-only. The tool used by Debian Live, Kali, Tails, Parrot, and most custom Debian installers.
- **mkosi** (systemd project): produces raw GPT disk images, UKIs, and ESPs. Designed for pre-installed image workflows ("dd the image to a disk, boot, first-boot provisioning"). Does not natively output ISO9660 hybrid installer images.

## Decision

Use **live-build** to produce the Helling installer ISO.

Rationale:

- ADR-021 explicitly describes a classical installer flow. live-build is the idiomatic Debian tool for that flow. mkosi is a better tool for a different pattern (pre-installed image + first-boot wizard) that ADR-021 did not choose.
- live-build integrates with Debian preseed for unattended install automation — directly answers the "3 questions" contract in ADR-021.
- Debian-native tool, Debian-packaged, maintained by the debian-live team. Matches ADR-002.
- Output is an ISO9660 installer image, dd-able to USB. amd64 builds include BIOS and UEFI bootloaders; arm64 builds use UEFI GRUB because Debian does not publish the legacy `grub-pc` bootloader for arm64.
- Preseed + first-boot hooks handle Helling-specific first-boot work (create a one-time setup token, start hellingd, and let the browser/CLI setup flow create the first admin).

Rejected alternatives:

- **mkosi**: produces raw GPT disk images, not ISO9660 installer ISOs. Adopting mkosi would require also revising ADR-021 to a "pre-installed image" pattern, which has different trade-offs (larger download ~4-8 GB vs installer ~1-2 GB, different first-boot UX, no classical "installer asks questions" flow). Out of scope for this ADR.
- **debian-cd / simple-cdd**: older, less maintained, thinner community than live-build for custom installer ISOs.

## Consequences

**Easier:**

- Standard Debian installer look and feel (familiar to any Debian operator)
- Preseed automation is a well-documented path
- amd64 hybrid ISO boots on BIOS and UEFI without separate artifacts; arm64 uses UEFI GRUB only
- Smaller download than a pre-installed image

**Harder:**

- live-build's configuration surface is large; Helling maintains a focused `auto/config` + `config/package-lists/*.list.chroot` set
- First-boot cannot fully provision from the installer — remaining work happens on the running target system via `first-boot.target` per `docs/spec/first-boot.md`

## Workflow

```bash
# In release CI (after .deb packages are published to the APT repo per ADR-045)
volume_label_from_version() {
  suffix="$(
    printf '%s' "$HELLING_VERSION" \
      | tr '[:lower:]' '[:upper:]' \
      | tr -cs 'A-Z0-9' '_' \
      | sed 's/^_//; s/_$//' \
      | cut -c 1-8
  )"
  [ -n "$suffix" ] || suffix="DEV"
  printf 'HELLING_%s' "$suffix" | cut -c 1-16
}

HELLING_ISO_VOLUME_LABEL="${HELLING_ISO_VOLUME_LABEL:-$(volume_label_from_version)}"
case "$HELLING_ISO_VOLUME_LABEL" in
  "" | *[!A-Z0-9_]*)
    echo "HELLING_ISO_VOLUME_LABEL must contain only A-Z, 0-9, and underscore" >&2
    exit 1
    ;;
esac
if [ "${#HELLING_ISO_VOLUME_LABEL}" -gt 16 ]; then
  echo "HELLING_ISO_VOLUME_LABEL must be at most 16 characters" >&2
  exit 1
fi

case "$HELLING_ISO_ARCH" in
  amd64) BOOTLOADERS="grub-pc grub-efi" ;;
  arm64) BOOTLOADERS="grub-efi" ;;
  *)
    echo "Unsupported Helling ISO architecture: $HELLING_ISO_ARCH" >&2
    exit 1
    ;;
esac

lb config \
  --distribution trixie \
  --architectures "$HELLING_ISO_ARCH" \
  --bootloaders "$BOOTLOADERS" \
  --iso-application "Helling" \
  --iso-volume "$HELLING_ISO_VOLUME_LABEL" \
  --binary-image iso-hybrid

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
# Output: live-image-${HELLING_ISO_ARCH}.hybrid.iso
```

The volume label is derived from `HELLING_VERSION`, normalized to uppercase
letters, digits, and underscores, prefixed with `HELLING_`, and capped at 16
characters so branch builds that use `git describe` stay compatible with both
ISO9660 and Joliet labels. If `HELLING_VERSION` contains no usable characters,
the label falls back to `HELLING_DEV`. Operator-provided
`HELLING_ISO_VOLUME_LABEL` values use the same character and length limits.

`deploy/iso/auto/config` sets `BOOTLOADERS` from the target architecture: amd64
uses `grub-pc grub-efi`; arm64 uses `grub-efi` because Debian does not ship
`grub-pc` for arm64.

## References

- ADR-021 (ISO-only installation)
- ADR-002 (Debian-first platform)
- ADR-025 (APT repository via GitHub Pages)
- ADR-045 (reprepro as repo tool)
- `docs/spec/first-boot.md`
- `docs/design/full-automation-pipeline.md` Layer 23
