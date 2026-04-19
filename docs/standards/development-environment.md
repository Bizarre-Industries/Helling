# Development Environment Standard

Recommended contributor environment for non-Linux hosts is a Lima VM running Debian.

## Scope

This standard applies to macOS and Windows contributors who need Linux-native behavior for Incus, systemd, and socket-based workflows.

## Baseline

- Host: macOS or Windows
- VM manager: Lima
- Guest OS: Debian stable
- Toolchain in guest: Go, Bun, Make, git, Incus CLI

## Setup Outline

1. Install Lima on host.
2. Create Debian Lima instance with enough CPU/RAM/disk for local builds.
3. Install required tooling inside VM.
4. Clone repository in VM and run bootstrap commands.
5. Use VS Code remote development attached to the Lima VM.

## Commands (inside VM)

```bash
sudo apt update
sudo apt install -y build-essential git curl make
# Install Go/Bun per project version requirements
```

## Validation

- `go version` matches project requirement
- `bun --version` is available
- `make` is available
- Incus socket/client operations function in VM

## Notes

- This standard complements ADR-034.
- Linux-native contributors may develop directly on host without Lima.
