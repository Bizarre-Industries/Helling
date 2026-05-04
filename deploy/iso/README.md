# Helling Installer ISO

This directory is the live-build source for the Helling installer ISO selected
by ADR-021 and ADR-046. It builds a Debian 13 hybrid ISO with:

- Incus, Podman, Caddy, DBus, and SSH in the package list.
- A Helling target-root payload copied into the installed system by the
  installer `late_command`.
- A first-boot service that creates users, groups, directories, Caddy config,
  Incus loopback HTTPS, and socket permissions automatically.

Build from a Debian 13 host or VM with `live-build` installed:

```sh
task iso:build
```

`scripts/build-iso.sh` writes `dist/iso/helling-<version>-<arch>.iso` and a
detached ASCII-armored signature at `.iso.asc`. Set `HELLING_ISO_SIGN=0` only
for local throwaway experiments.

For the current v0.1 repo state the ISO embeds locally built `hellingd`,
`helling`, and `web/dist` artifacts. Once ADR-045 APT publishing is wired, the
same live-build profile should switch the payload source to signed `.deb`
packages from the Helling APT repo.
