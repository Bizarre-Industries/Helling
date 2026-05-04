# Install Helling

Helling v0.1 installs through the Debian-first installer ISO. The ISO installs Debian, Incus, Podman, Caddy, `hellingd`, the WebUI bundle, and the first-boot service.

## Requirements

- A target machine or VM with a blank boot disk
- Network access during install
- Debian-compatible amd64 hardware
- The Helling installer ISO built from this repository

## Build The ISO

On a Debian build host with `live-build` installed:

```bash
task iso:build
```

For a source-only validation that does not require root or live-build:

```bash
task check:iso
```

The builder prepares `dist/iso/live-build`, copies the Helling payload into the installer profile, runs live-build, and emits detached signatures when signing keys are available.

## Install

1. Boot the target from the Helling ISO.
2. Complete the Debian installer prompts for hostname, disk, locale, and network.
3. Reboot into the installed system after the installer finishes.
4. Wait for the first-boot service to complete:

```bash
systemctl status helling-first-boot.service
```

The first-boot service creates `/etc/helling/setup-token`, writes `/etc/helling/helling.yaml`, starts Incus/Podman sockets, installs the Caddyfile, and starts `hellingd` plus Caddy.

## Create The First Admin

Open the WebUI:

```text
https://<host>:8006
```

Use the setup token from the installed host:

```bash
sudo cat /etc/helling/setup-token
```

The token is valid only while no Helling users exist. After the first admin is created, `hellingd` removes or truncates the token file.

For local CLI setup on the installed host:

```bash
sudo helling auth setup \
  --api http+unix:///run/helling/api.sock \
  --setup-token-file /etc/helling/setup-token
```

The CLI prompts for the admin password without echo and asks for confirmation. Non-interactive automation must pass both `--password-file` and `--setup-token-file`; do not pass secrets in argv.

## Verify

```bash
curl -kfsS https://127.0.0.1:8006/healthz
helling auth login --api http+unix:///run/helling/api.sock
helling instance ls
```

If the host has no IP after boot, confirm the VM or server has a boot disk attached and that it actually booted the installed system rather than returning to firmware or the ISO.
