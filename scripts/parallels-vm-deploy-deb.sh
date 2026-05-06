#!/usr/bin/env bash
# scripts/parallels-vm-deploy-deb.sh
#
# Release-gate deploy: build .deb via the ADR-045 reprepro flow, scp to VM, apt install.
# Per ADR-052. Used by `task vm:parallels:release-test`.
#
# Required env:
#   HELLING_VM_HOST   VM IP
#   HELLING_VM_USER   default: helling
#   HELLING_VM_SSH_PORT default: 22
#
# Skip behavior is opt-in only:
#   set HELLING_VM_RELEASE_ALLOW_SKIPS=1 to allow ADR-045 tooling gaps to exit 0.

set -euo pipefail

VM_HOST="${HELLING_VM_HOST:?set HELLING_VM_HOST=<vm-ip>}"
VM_USER="${HELLING_VM_USER:-helling}"
VM_SSH_PORT="${HELLING_VM_SSH_PORT:-22}"
SSH_OPTS=(
  -o BatchMode=yes
  -o ConnectTimeout=10
  -o ServerAliveInterval=5
  -o ServerAliveCountMax=3
  -p "$VM_SSH_PORT"
)
SCP_OPTS=(
  -q
  -o BatchMode=yes
  -o ConnectTimeout=10
  -o ServerAliveInterval=5
  -o ServerAliveCountMax=3
  -P "$VM_SSH_PORT"
)

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
skip() {
  if [ "${HELLING_VM_RELEASE_ALLOW_SKIPS:-0}" = "1" ]; then
    printf '○ %s\n' "$*"
    exit 0
  fi
  fail "$* (set HELLING_VM_RELEASE_ALLOW_SKIPS=1 only for non-gating local smoke)"
}
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}
case "$VM_SSH_PORT" in
  '' | *[!0-9]*) fail "HELLING_VM_SSH_PORT must be numeric (got: $VM_SSH_PORT)" ;;
esac
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
if [ -z "$DEB" ] || [ ! -f "$DEB" ]; then
  fail "No helling*.deb produced in dist/. Check scripts/build-deb.sh output."
fi
log "Built: $DEB"

# ---- ship + install ----

DEB_BASE="$(basename "$DEB")"
log "scp $DEB -> $VM_USER@$VM_HOST:/tmp/$DEB_BASE"
timeout 60 scp "${SCP_OPTS[@]}" "$DEB" "$VM_USER@$VM_HOST:/tmp/$DEB_BASE"

log "apt install on VM and restart hellingd"
timeout 180 ssh "${SSH_OPTS[@]}" "$VM_USER@$VM_HOST" "sudo apt-get install -y --reinstall /tmp/$DEB_BASE && sudo systemctl restart hellingd"

done_ "Release-shaped deploy complete"
echo "Verify: task vm:parallels:smoke"
