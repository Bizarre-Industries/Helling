#!/usr/bin/env bash
# scripts/parallels-vm-deploy-rsync.sh
#
# Inner-loop deploy: cross-build linux/$(arch), rsync to Parallels VM, restart hellingd.
# Per ADR-052. Used by `task vm:parallels:sync` / `task vm:parallels:dev`.
#
# Required env:
#   HELLING_VM_HOST   VM IP (printed by parallels-vm-bootstrap.sh)
#   HELLING_VM_USER   default: helling
#
# Optional env:
#   HELLING_VM_ARCH   default: arch of the VM (auto-detected via uname -m)

set -euo pipefail

VM_HOST="${HELLING_VM_HOST:?set HELLING_VM_HOST=<vm-ip>; run parallels-vm-bootstrap.sh first}"
VM_USER="${HELLING_VM_USER:-helling}"

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

SSH() { ssh -o BatchMode=yes "$VM_USER@$VM_HOST" "$@"; }

# Detect VM arch unless caller pinned it.
if [ -z "${HELLING_VM_ARCH:-}" ]; then
  log "Detecting VM arch over SSH"
  GUEST_ARCH="$(SSH 'uname -m')"
  case "$GUEST_ARCH" in
    aarch64 | arm64) HELLING_VM_ARCH="arm64" ;;
    x86_64 | amd64) HELLING_VM_ARCH="amd64" ;;
    *) fail "Unsupported guest arch: $GUEST_ARCH" ;;
  esac
fi
log "Target arch: $HELLING_VM_ARCH"

OUT="bin/linux-$HELLING_VM_ARCH"
mkdir -p "$OUT"

# Try pure cross-compile first. CGO is required for sqlite/pam paths in hellingd,
# so on macOS we'd need a linux C toolchain. If CGO cross-compile fails, fall back
# to building inside the VM.
log "Cross-compiling linux/$HELLING_VM_ARCH from host"
HOST_BUILD_OK=1
for cmd in apps/hellingd apps/helling-cli; do
  [ -d "$cmd" ] || continue
  name=$(basename "$cmd")
  if ! GOOS=linux GOARCH="$HELLING_VM_ARCH" CGO_ENABLED=0 \
    go build -trimpath -ldflags='-s -w' -o "$OUT/$name" "./$cmd" 2>/dev/null; then
    log "Pure cross-compile of $name failed (likely needs CGO). Will build in VM."
    HOST_BUILD_OK=0
  fi
done

if [ "$HOST_BUILD_OK" = "0" ]; then
  log "Falling back to in-VM build"
  REPO_NAME="$(basename "$REPO_ROOT")"
  rsync -az --delete \
    --exclude '.git' --exclude 'node_modules' --exclude '.task' \
    --exclude 'bin' --exclude 'dist' --exclude 'web/dist' \
    "$REPO_ROOT/" "$VM_USER@$VM_HOST:/home/$VM_USER/$REPO_NAME/"
  SSH "cd /home/$VM_USER/$REPO_NAME && \
    mkdir -p bin && \
    for c in apps/hellingd apps/helling-cli; do \
      [ -d \"\$c\" ] && go build -trimpath -ldflags='-s -w' -o \"bin/\$(basename \$c)\" \"./\$c\"; \
    done && \
    sudo install -m 0755 bin/hellingd /usr/local/bin/hellingd && \
    { [ -f bin/helling ] && sudo install -m 0755 bin/helling /usr/local/bin/helling; true; } && \
    sudo systemctl restart hellingd"
  done_ "Deployed (in-VM build) and restarted hellingd"
  echo "Tail logs: ssh $VM_USER@$VM_HOST sudo journalctl -fu hellingd"
  exit 0
fi

# Host-built path.
log "rsync $OUT/ -> $VM_USER@$VM_HOST:/tmp/helling-deploy/"
SSH "mkdir -p /tmp/helling-deploy"
rsync -az --delete "$OUT/" "$VM_USER@$VM_HOST:/tmp/helling-deploy/"

log "Installing binaries and restarting hellingd"
SSH "sudo install -m 0755 /tmp/helling-deploy/hellingd /usr/local/bin/hellingd && \
  { [ -f /tmp/helling-deploy/helling ] && sudo install -m 0755 /tmp/helling-deploy/helling /usr/local/bin/helling; true; } && \
  sudo systemctl restart hellingd"

done_ "Deployed and restarted hellingd"
echo "Tail logs: ssh $VM_USER@$VM_HOST sudo journalctl -fu hellingd"
