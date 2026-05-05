# Parallels Debian Dev Image

This Packer template builds a reusable Debian 13 Parallels VM image for the
Helling inner development loop. It is not the Helling product installer and it
does not build the Helling live-build ISO.

Use it once per local machine, then use the existing rsync/deb deploy tasks for
daily work:

```bash
task vm:parallels:build-image
task vm:parallels:up
task vm:parallels:dev
task vm:parallels:smoke
```

The default image uses the pinned Debian 13.4.0 netinst ISO for the host architecture,
creates the `helling` user, installs sudo/SSH, Incus, Podman, Go, Bun, Caddy,
live-build, and the Helling Go toolchain, then exports a `.pvm` under
`dist/packer/`.

The installer uses a generated one-time password only to satisfy Debian user
creation. Packer connects with the SSH key, then provisioning locks the password
and disables SSH password auth before the VM is registered; normal access is by
`HELLING_VM_SSHKEY` only. If the matching private key is not
`${HELLING_VM_SSHKEY%.pub}`, set `HELLING_VM_SSH_PRIVATE_KEY`.

Override inputs with environment variables consumed by
`scripts/parallels-vm-build-dev.sh`:

```bash
HELLING_PACKER_DEBIAN_VERSION=13.4.0
HELLING_PACKER_ISO_URL=file:///path/to/debian-13.4.0-arm64-netinst.iso
HELLING_PACKER_ISO_CHECKSUM=sha256:<digest>
HELLING_VM_NAME=helling-dev
HELLING_VM_CPUS=6
HELLING_VM_MEM_MB=12288
HELLING_VM_DISK_GB=60
```

Full Helling ISO generation remains release-gate only. Use `task iso:verify`
for normal checks and set `HELLING_ISO_RELEASE_GATE=1` only when intentionally
running `task iso:build` for a version gate.
