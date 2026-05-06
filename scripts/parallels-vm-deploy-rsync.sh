#!/usr/bin/env bash
# scripts/parallels-vm-deploy-rsync.sh
#
# Inner-loop deploy: cross-build linux/$(arch), rsync to Parallels VM, restart hellingd.
# Per ADR-052. Used by `task vm:parallels:sync` / `task vm:parallels:dev`.
#
# Required env:
#   HELLING_VM_HOST   VM IP (printed by task vm:parallels:up/bootstrap)
#   HELLING_VM_USER   default: helling
#
# Optional env:
#   HELLING_VM_ARCH   default: arch of the VM (auto-detected via uname -m)
#   HELLING_VM_SSH_PORT default: 22

set -euo pipefail

VM_HOST="${HELLING_VM_HOST:?set HELLING_VM_HOST=<vm-ip>; run task vm:parallels:up first}"
VM_USER="${HELLING_VM_USER:-helling}"
VM_SSH_PORT="${HELLING_VM_SSH_PORT:-22}"

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}
case "$VM_SSH_PORT" in
  '' | *[!0-9]*) fail "HELLING_VM_SSH_PORT must be numeric (got: $VM_SSH_PORT)" ;;
esac

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

SSH() { ssh -o BatchMode=yes -p "$VM_SSH_PORT" "$VM_USER@$VM_HOST" "$@"; }

GO_COMMANDS=(
  apps/hellingd
  apps/helling-cli
  apps/hellingd/cmd/helling-unit-link
  apps/hellingd/cmd/helling-incus-trust
  apps/hellingd/cmd/helling-firewall
)

binary_name() {
  case "$1" in
    apps/helling-cli) printf 'helling\n' ;;
    *) basename "$1" ;;
  esac
}

remote_install_payload() {
  local deploy_dir="$1"
  SSH "DEPLOY_DIR=$deploy_dir bash -s" <<'REMOTE'
set -euo pipefail

sudo install -m 0755 "$DEPLOY_DIR/hellingd" /usr/local/bin/hellingd
if [ -f "$DEPLOY_DIR/helling" ]; then
  sudo install -m 0755 "$DEPLOY_DIR/helling" /usr/local/bin/helling
  sudo install -m 0755 "$DEPLOY_DIR/helling" /usr/bin/helling
fi

for group in helling helling-proxy incus; do
  if ! getent group "$group" >/dev/null; then
    sudo groupadd --system "$group"
  fi
done
for user in helling hellingd; do
  if id -u "$user" >/dev/null 2>&1; then
    sudo usermod -aG helling,helling-proxy,incus "$user"
  fi
done
service_user="$(systemctl show -p User --value hellingd 2>/dev/null || true)"
if [ -z "$service_user" ]; then
  service_user="helling"
fi
service_group="$(id -gn "$service_user" 2>/dev/null || printf '%s' "$service_user")"

sudo install -d -o root -g root -m 0755 /usr/lib/helling
for helper in helling-unit-link helling-incus-trust; do
  if [ -f "$DEPLOY_DIR/$helper" ]; then
    sudo install -o root -g helling -m 4750 "$DEPLOY_DIR/$helper" "/usr/lib/helling/$helper"
  fi
done
if [ -f "$DEPLOY_DIR/helling-firewall" ]; then
  sudo install -o root -g helling -m 4750 "$DEPLOY_DIR/helling-firewall" /usr/lib/helling/helling-firewall
  if [ -x /usr/sbin/setcap ]; then
    sudo /usr/sbin/setcap cap_net_admin+ep /usr/lib/helling/helling-firewall || true
  elif command -v setcap >/dev/null 2>&1; then
    sudo "$(command -v setcap)" cap_net_admin+ep /usr/lib/helling/helling-firewall || true
  fi
fi

sudo install -d -o root -g helling -m 0750 /etc/helling
if [ -f /etc/helling/helling.yaml ]; then
  sudo chown root:helling /etc/helling/helling.yaml
  sudo chmod 0640 /etc/helling/helling.yaml
fi
sudo install -d -o "$service_user" -g "$service_group" -m 0700 /etc/helling/age
if [ ! -f /etc/helling/age/identity.txt ] && [ -f /var/lib/helling/age-identity.txt ]; then
  sudo install -o "$service_user" -g "$service_group" -m 0600 /var/lib/helling/age-identity.txt /etc/helling/age/identity.txt
elif [ -f /etc/helling/age/identity.txt ]; then
  sudo chown "$service_user:$service_group" /etc/helling/age/identity.txt
  sudo chmod 0600 /etc/helling/age/identity.txt
fi
if [ ! -f /etc/helling/schedule-runner.token ]; then
  token="$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | od -An -tx1 | tr -d ' \n')"
  printf '%s\n' "$token" | sudo tee /etc/helling/schedule-runner.token >/dev/null
fi
sudo chown root:helling /etc/helling/schedule-runner.token
sudo chmod 0640 /etc/helling/schedule-runner.token

sudo systemctl reset-failed hellingd >/dev/null 2>&1 || true
sudo systemctl restart hellingd
if [ -d "$DEPLOY_DIR/web" ]; then
  sudo install -d -o root -g root -m 0755 /usr/share/helling/web
  sudo rsync -a --delete "$DEPLOY_DIR/web/" /usr/share/helling/web/
fi
if [ -f "$DEPLOY_DIR/Caddyfile" ]; then
  sudo install -d -o root -g root -m 0755 /usr/share/helling /etc/caddy
  sudo install -o root -g root -m 0644 "$DEPLOY_DIR/Caddyfile" /usr/share/helling/Caddyfile
  sudo install -o root -g root -m 0644 "$DEPLOY_DIR/Caddyfile" /etc/caddy/Caddyfile
fi
if command -v caddy >/dev/null 2>&1; then
  if getent group helling-proxy >/dev/null; then
    sudo usermod -aG helling-proxy caddy 2>/dev/null || true
  fi
  sudo systemctl enable caddy >/dev/null 2>&1 || true
  sudo systemctl restart caddy
fi
REMOTE
}

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

# Try pure cross-compile first. If the host cannot produce Linux binaries with
# the installed toolchain, fall back to building inside the VM.
log "Cross-compiling linux/$HELLING_VM_ARCH from host"
HOST_BUILD_OK=1
for cmd in "${GO_COMMANDS[@]}"; do
  [ -d "$cmd" ] || continue
  name="$(binary_name "$cmd")"
  if ! GOOS=linux GOARCH="$HELLING_VM_ARCH" CGO_ENABLED=0 \
    go build -trimpath -ldflags='-s -w' -o "$OUT/$name" "./$cmd" 2>/dev/null; then
    log "Pure cross-compile of $name failed (likely needs CGO). Will build in VM."
    HOST_BUILD_OK=0
  fi
done

if [ "$HOST_BUILD_OK" = "0" ]; then
  log "Falling back to in-VM build"
  REPO_NAME="$(basename "$REPO_ROOT")"
  rsync -az --delete -e "ssh -p $VM_SSH_PORT" \
    --exclude '.git' --exclude 'node_modules' --exclude '.task' \
    --exclude 'bin' --exclude 'dist' --exclude 'web/dist' \
    "$REPO_ROOT/" "$VM_USER@$VM_HOST:/home/$VM_USER/$REPO_NAME/"
  SSH "cd /home/$VM_USER/$REPO_NAME && bash -s" <<'REMOTE'
set -euo pipefail
binary_name() {
  case "$1" in
    apps/helling-cli) printf 'helling\n' ;;
    *) basename "$1" ;;
  esac
}
mkdir -p bin
for cmd in \
  apps/hellingd \
  apps/helling-cli \
  apps/hellingd/cmd/helling-unit-link \
  apps/hellingd/cmd/helling-incus-trust \
  apps/hellingd/cmd/helling-firewall; do
  [ -d "$cmd" ] || continue
  name="$(binary_name "$cmd")"
  go build -trimpath -ldflags='-s -w' -o "bin/$name" "./$cmd"
done
REMOTE
  remote_install_payload "/home/$VM_USER/$REPO_NAME/bin"
  done_ "Deployed (in-VM build) and restarted hellingd"
  echo "Tail logs: ssh -p $VM_SSH_PORT $VM_USER@$VM_HOST sudo journalctl -fu hellingd"
  exit 0
fi

# Host-built path.
log "rsync $OUT/ -> $VM_USER@$VM_HOST:/tmp/helling-deploy/"
SSH "mkdir -p /tmp/helling-deploy"
rsync -az --delete -e "ssh -p $VM_SSH_PORT" "$OUT/" "$VM_USER@$VM_HOST:/tmp/helling-deploy/"
if [ -d web/dist ]; then
  rsync -az --delete -e "ssh -p $VM_SSH_PORT" web/dist/ "$VM_USER@$VM_HOST:/tmp/helling-deploy/web/"
fi
if [ -f deploy/install/Caddyfile ]; then
  rsync -az -e "ssh -p $VM_SSH_PORT" deploy/install/Caddyfile "$VM_USER@$VM_HOST:/tmp/helling-deploy/Caddyfile"
fi

log "Installing binaries and restarting hellingd"
remote_install_payload /tmp/helling-deploy

done_ "Deployed and restarted hellingd"
echo "Tail logs: ssh -p $VM_SSH_PORT $VM_USER@$VM_HOST sudo journalctl -fu hellingd"
