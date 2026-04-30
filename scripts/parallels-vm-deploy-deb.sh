#!/usr/bin/env bash
# scripts/parallels-vm-deploy-deb.sh
#
# Release-gate deploy: build .deb via the ADR-045 reprepro flow, scp to VM, apt install.
# Per ADR-052. Used by `task vm:parallels:release-test`.
#
# Required env:
#   HELLING_VM_HOST   VM IP
#   HELLING_VM_USER   default: helling
#
# Behavior when reprepro tooling is not yet wired (ADR-045 in flight):
#   exits 0 with "SKIPPED: reprepro not configured (ADR-045)" so the smoke test
#   downstream can decide whether to bail or proceed.

set -euo pipefail

VM_HOST="${HELLING_VM_HOST:?set HELLING_VM_HOST=<vm-ip>}"
VM_USER="${HELLING_VM_USER:-helling}"

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
skip() {
  printf '○ %s\n' "$*"
  exit 0
}
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}
have() { command -v "$1" >/dev/null 2>&1; }

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# ---- reprepro tooling guard ----
# ADR-045 picks reprepro for the APT repository. Until that tooling is wired (a
# scripts/build-deb.sh or task target), this script is a no-op stub so the
# release-test task can still run end-to-end without false failures.

if ! have reprepro; then
  skip "SKIPPED: reprepro not installed on host (ADR-045 tooling not yet provisioned)"
fi
if [ ! -d "$REPO_ROOT/dist/apt" ] && [ ! -f "$REPO_ROOT/scripts/build-deb.sh" ]; then
  skip "SKIPPED: reprepro not configured (ADR-045) — no dist/apt or scripts/build-deb.sh found"
fi

# ---- build .deb ----

log "Building .deb via reprepro flow"
if [ -x "$REPO_ROOT/scripts/build-deb.sh" ]; then
  bash "$REPO_ROOT/scripts/build-deb.sh"
else
  fail "scripts/build-deb.sh not found or not executable. Wire ADR-045 reprepro tooling first."
fi

DEB="$(find "$REPO_ROOT/dist" -maxdepth 3 -name 'helling*.deb' -print -quit 2>/dev/null || true)"
[ -n "$DEB" ] && [ -f "$DEB" ] || fail "No helling*.deb produced in dist/. Check scripts/build-deb.sh output."
log "Built: $DEB"

# ---- ship + install ----

DEB_BASE="$(basename "$DEB")"
log "scp $DEB -> $VM_USER@$VM_HOST:/tmp/$DEB_BASE"
scp -q "$DEB" "$VM_USER@$VM_HOST:/tmp/$DEB_BASE"

log "apt install on VM and restart hellingd"
ssh -o BatchMode=yes "$VM_USER@$VM_HOST" "sudo apt-get install -y --reinstall /tmp/$DEB_BASE && sudo systemctl restart hellingd"

done_ "Release-shaped deploy complete"
echo "Verify: task vm:parallels:smoke"
