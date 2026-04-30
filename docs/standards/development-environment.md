# Development Environment Standard

Standard local environment and workflow for Helling contributors.

## Scope

Applies to all contributors. Linux-native development is supported. macOS contributors should use Parallels Desktop (ADR-052) as the primary path; Lima (ADR-034) remains a documented fallback for contributors without Parallels. Windows contributors continue to use Lima.

## Required Toolchain

- Go 1.26
- Bun
- make
- git
- optional: task (Taskfile workflow)

## Recommended Environments

1. Linux host: develop directly on host.
2. macOS with Parallels Desktop: Debian VM provisioned via `scripts/parallels-vm-bootstrap.sh` and `task vm:parallels:up` (per ADR-052). Primary macOS path.
3. macOS or Windows with Lima: Debian VM (per ADR-034). Fallback path; supported but secondary.

## Parallels Baseline (macOS — primary)

- VM manager: Parallels Desktop (commercial; user-supplied license).
- Guest OS: Debian stable.
- VM name: `helling-dev`.
- Sizing defaults: 4 vCPU, 8 GB RAM, 40 GB disk. Override via `HELLING_VM_CPUS`, `HELLING_VM_MEM_MB`, `HELLING_VM_DISK_GB`.
- Networking: Parallels bridged interface so the host can reach the VM by IP for `rsync` + `ssh`.
- Auth: contributor's SSH public key (`HELLING_VM_SSHKEY`, default `~/.ssh/id_ed25519.pub`) injected via cloud-init.

Bootstrap (one-time):

```bash
bash scripts/parallels-vm-bootstrap.sh
```

This installs `build-essential git curl make ca-certificates dbus policykit-1 systemd incus podman` plus Go and Bun inside the VM, and lays down a `hellingd` systemd unit drop-in so `systemctl restart hellingd` works after the first deploy.

Daily loop:

```bash
task vm:parallels:up      # boot VM if stopped
task vm:parallels:dev     # build:linux + rsync + restart hellingd
task vm:parallels:smoke   # health probe + smoke checks
task vm:parallels:logs    # journalctl -fu hellingd
```

Release-shaped path:

```bash
task vm:parallels:release-test  # builds .deb (ADR-045), installs in VM, smokes
```

## Lima Baseline (macOS/Windows — fallback)

- VM manager: Lima.
- Guest OS: Debian stable.
- Sizing: enough CPU/RAM/disk for Go + web builds and local checks.

Inside VM:

```bash
sudo apt update
sudo apt install -y build-essential git curl make
```

Install Go/Bun per repository requirements, then bootstrap project.

## Standard Local Workflow

See `docs/spec/local-dev.md` for normative step-by-step workflow.

Common command sequence:

```bash
make dev-setup
make generate
make fmt-check
make lint
make test
```

Task workflow equivalent:

```bash
task install
task hooks
task check
```

## Hook Installation

If lefthook is enabled in the repository:

```bash
task hooks
```

Expected behavior:

- pre-commit runs fast checks
- pre-push runs full checks

## Validation

- `go version` reports 1.26.x
- `bun --version` is available
- generation/lint/test commands run locally
- Git hooks install and execute correctly

## Notes

- This standard complements ADR-034.
- Environment details can evolve, but required checks/gates may not be skipped.
