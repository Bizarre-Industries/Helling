# Development Environment Standard

Standard local environment and workflow for Helling contributors.

## Scope

Applies to all contributors. Linux-native development is supported. macOS/Windows should use Lima (ADR-034) for Linux-native Incus/systemd behavior.

## Required Toolchain

- Go 1.26
- Bun
- make
- git
- optional: task (Taskfile workflow)

## Recommended Environments

1. Linux host: develop directly on host.
2. macOS/Windows: Lima Debian VM.

## Lima Baseline (macOS/Windows)

- VM manager: Lima
- Guest OS: Debian stable
- Sizing: enough CPU/RAM/disk for Go + web builds and local checks

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
