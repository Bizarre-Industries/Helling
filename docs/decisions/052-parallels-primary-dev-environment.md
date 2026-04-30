# ADR-052: Parallels Desktop as primary macOS dev/test environment

> Status: Accepted (2026-04-30)
>
> Relates to: ADR-034 (Lima dev environment) — Lima retained as documented fallback.

## Context

Helling targets Debian-first host behavior. Backend (`hellingd`) requires:

- systemd, DBus, polkit (ADR-050 — non-root daemon via systemd interaction)
- Incus on the host (ADR-001, ADR-024, ADR-036)
- Podman on the host (ADR-004, ADR-035)
- Real APT install path for release-shaped validation (ADR-021, ADR-045, ADR-046)

None of these run natively on macOS. Contributors on macOS therefore need a Debian VM to exercise `hellingd` end-to-end. ADR-034 selected Lima for that role on grounds of being free, OSS, and Apple-Silicon-native.

In practice the project's primary maintainer runs Parallels Desktop and uses it as the active deploy + test target during implementation. Parallels gives stronger I/O performance, snapshot UX, GUI affordances for inspecting VM state, and a more robust nested-virtualization story for exercising Incus inside the VM. The repo today has no Parallels tooling, no Linux cross-build target, no rsync-to-VM helper, no `.deb`-install path, and no smoke-test runner that talks to a remote VM. "Test on Parallels" is improvised every iteration.

## Decision

Parallels Desktop on macOS becomes the primary documented Debian dev/test VM for Helling.

- Default contributor path on macOS: Parallels Desktop guest running Debian 13 (`helling-dev` VM name).
- Lima (ADR-034) remains a documented and supported fallback for contributors who cannot or will not run Parallels (license cost, organizational policy).
- Two deploy paths are wired:
  1. `rsync` + `systemctl restart hellingd` for the inner dev loop (fast, used per save / per build).
  2. `.deb` install via the ADR-045 reprepro path for release-gate validation (slow, release-shaped).
- Tooling lives in `scripts/parallels-vm-bootstrap.sh`, `scripts/parallels-vm-deploy-rsync.sh`, `scripts/parallels-vm-deploy-deb.sh`, and `Taskfile.yaml` `vm:parallels:*` + `build:linux` targets.
- CI continues to run on GitHub-hosted Linux runners. This ADR does not change CI.

## Consequences

Easier:

- A real Debian system with systemd, DBus, polkit, Incus, and Podman is one `task vm:parallels:up` away on every macOS contributor laptop that has Parallels Desktop installed.
- Inner dev loop (`task vm:parallels:dev`) is a build + rsync + service restart that finishes in seconds.
- Release-gate validation (`task vm:parallels:release-test`) drives the same `.deb` flow that ships to users via ADR-045 / ADR-046.
- VM snapshots make destructive testing (Incus storage operations, polkit edge cases) cheap to roll back.

Harder / costs:

- Parallels Desktop is commercial software with a license cost. Contributors who cannot fund a license use the Lima fallback (ADR-034) instead.
- Two VM tooling surfaces (Parallels primary, Lima fallback) must stay in sync. Mitigated by mirroring the task namespace (`vm:parallels:*` and `vm:lima:*`) so command shape is identical.
- Cross-compile from macOS to linux with CGO (sqlite/pam) is constrained. The deploy scripts try pure cross-compile first and fall back to building inside the VM.

## References

- `docs/spec/local-dev.md` — supported host platforms + first-time setup.
- `docs/standards/development-environment.md` — Parallels baseline + Lima fallback baseline.
- `docs/roadmap/checklist.md` — v0.1.0-alpha tracking items.
- ADR-034 — Lima dev environment (now fallback).
- ADR-045 — reprepro APT repository tooling (consumed by the `.deb` deploy path).
- ADR-050 — non-root hellingd via DBus + polkit (constrains target environment).
