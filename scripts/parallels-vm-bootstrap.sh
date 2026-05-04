#!/usr/bin/env bash
# scripts/parallels-vm-bootstrap.sh
#
# Provision a Debian 13 dev VM in Parallels Desktop for Helling per ADR-052.
# Idempotent. Re-runs are safe.
#
# Defaults (override via env):
#   HELLING_VM_NAME      helling-dev
#   HELLING_VM_CPUS      4
#   HELLING_VM_MEM_MB    8192
#   HELLING_VM_DISK_GB   40
#   HELLING_VM_USER      helling
#   HELLING_VM_SSHKEY    ~/.ssh/id_ed25519.pub
#   HELLING_VM_SSH_PORT  22
#   HELLING_GO_VERSION   1.26.2
#   HELLING_GO_SHA256    required when overriding HELLING_GO_VERSION
#
# Outputs (printed to stdout):
#   HELLING_VM_HOST      VM IP address (use this to export for vm:parallels:* tasks)
#
# Requires: Parallels Desktop with prlctl in PATH.

set -euo pipefail

VM_NAME="${HELLING_VM_NAME:-helling-dev}"
VM_CPUS="${HELLING_VM_CPUS:-4}"
VM_MEM_MB="${HELLING_VM_MEM_MB:-8192}"
VM_DISK_GB="${HELLING_VM_DISK_GB:-40}"
VM_USER="${HELLING_VM_USER:-helling}"
VM_SSHKEY="${HELLING_VM_SSHKEY:-$HOME/.ssh/id_ed25519.pub}"
VM_SSH_PORT="${HELLING_VM_SSH_PORT:-22}"
VM_HOST_OVERRIDE="${HELLING_VM_HOST:-}"
GO_VERSION="${HELLING_GO_VERSION:-1.26.2}"
GO_SHA256="${HELLING_GO_SHA256:-}"

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
skip() { printf '○ %s\n' "$*"; }
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}
have() { command -v "$1" >/dev/null 2>&1; }

# ---- preflight ----

have prlctl || fail "prlctl not found. Install Parallels Desktop: https://www.parallels.com/products/desktop/"
[ -f "$VM_SSHKEY" ] || fail "SSH public key not found: $VM_SSHKEY (set HELLING_VM_SSHKEY=path/to/id_*.pub)"
[[ "$VM_NAME" =~ ^[A-Za-z0-9._-]{1,64}$ ]] || fail "HELLING_VM_NAME may contain only letters, numbers, dot, underscore, and dash"
[[ "$VM_USER" =~ ^[a-z_][a-z0-9_-]{0,31}$ ]] || fail "HELLING_VM_USER must be a valid Debian user name"
case "$VM_SSH_PORT" in
  '' | *[!0-9]*) fail "HELLING_VM_SSH_PORT must be numeric (got: $VM_SSH_PORT)" ;;
esac
for numeric_var in VM_CPUS VM_MEM_MB VM_DISK_GB; do
  numeric_value="${!numeric_var}"
  case "$numeric_value" in
    '' | *[!0-9]*) fail "HELLING_${numeric_var#VM_} must be numeric (got: $numeric_value)" ;;
  esac
done
[[ "$GO_VERSION" =~ ^[0-9]+[.][0-9]+([.][0-9]+)?$ ]] || fail "HELLING_GO_VERSION must look like 1.26.2"
if [ -n "$GO_SHA256" ]; then
  [[ "$GO_SHA256" =~ ^[0-9a-f]{64}$ ]] || fail "HELLING_GO_SHA256 must be a lowercase 64-character SHA-256 hex digest"
fi

UNAME_M="$(uname -m)"
case "$UNAME_M" in
  arm64 | aarch64) DEB_ARCH="arm64" ;;
  x86_64 | amd64) DEB_ARCH="amd64" ;;
  *) fail "Unsupported host arch: $UNAME_M" ;;
esac
log "Host arch: $UNAME_M -> Debian arch: $DEB_ARCH"

# ---- VM existence check ----

if prlctl list -a 2>/dev/null | awk '{print $NF}' | grep -qx "$VM_NAME"; then
  skip "VM '$VM_NAME' already exists"
  STATE="$(prlctl list -a 2>/dev/null | awk -v n="$VM_NAME" '$NF==n {print $2}')"
  if [ "$STATE" != "running" ]; then
    log "Starting VM '$VM_NAME' (was: $STATE)"
    prlctl start "$VM_NAME"
  fi
else
  log "Creating VM '$VM_NAME' (Debian 13, $DEB_ARCH, ${VM_CPUS}c/${VM_MEM_MB}M/${VM_DISK_GB}G)"
  # Use Parallels' built-in Debian template if present; otherwise fall back to a
  # generic Linux profile that can boot a downloaded cloud image. Operators on
  # first install will be prompted by Parallels for the ISO/image — this script
  # does not auto-download to keep its blast radius narrow.
  prlctl create "$VM_NAME" --distribution debian --no-hdd
  prlctl set "$VM_NAME" --cpus "$VM_CPUS"
  prlctl set "$VM_NAME" --memsize "$VM_MEM_MB"
  prlctl set "$VM_NAME" --device-add hdd --size "$((VM_DISK_GB * 1024))"
  prlctl set "$VM_NAME" --device-set net0 --type bridged
  log "VM created. Attach a Debian 13 $DEB_ARCH installer ISO and complete first-boot install:"
  log "  prlctl set $VM_NAME --device-set cdrom0 --image /path/to/debian-13-$DEB_ARCH.iso --connect"
  log "  prlctl start $VM_NAME"
  log "After first-boot install creates user '$VM_USER', re-run this script to finish provisioning."
  exit 0
fi

# ---- wait for SSH ----

if [ -n "$VM_HOST_OVERRIDE" ]; then
  VM_IP="$VM_HOST_OVERRIDE"
  skip "Using HELLING_VM_HOST=$VM_IP"
else
  log "Resolving VM IP (max 60s)"
  VM_IP=""
  for _ in $(seq 1 30); do
    VM_IP="$(prlctl list -f --no-header "$VM_NAME" 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$/) {print $i; exit}}')"
    [ -n "$VM_IP" ] && break
    sleep 2
  done
  [ -n "$VM_IP" ] || fail "Could not resolve VM IP. Is Parallels Tools installed in the guest? If using Parallels shared networking, set HELLING_VM_HOST=127.0.0.1 and HELLING_VM_SSH_PORT=<forwarded-port>."
  done_ "VM IP: $VM_IP"
fi

log "Waiting for SSH on $VM_IP:$VM_SSH_PORT (max 60s)"
SSH_READY=0
for _ in $(seq 1 30); do
  if nc -z -w 2 "$VM_IP" "$VM_SSH_PORT" 2>/dev/null; then
    SSH_READY=1
    break
  fi
  sleep 2
done
[ "$SSH_READY" = "1" ] || fail "SSH not reachable. Ensure sshd is running and the firewall allows port $VM_SSH_PORT."

# ---- inject SSH key ----

log "Authorizing host SSH key for $VM_USER@$VM_IP (will prompt for password the first time)"
KNOWN_HOST="$VM_IP"
[ "$VM_SSH_PORT" = "22" ] || KNOWN_HOST="[$VM_IP]:$VM_SSH_PORT"
ssh-keygen -F "$KNOWN_HOST" >/dev/null 2>&1 || ssh-keyscan -p "$VM_SSH_PORT" -H "$VM_IP" >>"$HOME/.ssh/known_hosts" 2>/dev/null
if ! ssh -p "$VM_SSH_PORT" -o BatchMode=yes -o ConnectTimeout=5 "$VM_USER@$VM_IP" true 2>/dev/null; then
  # Pipe pubkey via stdin so the value isn't interpolated into the remote shell.
  ssh -p "$VM_SSH_PORT" "$VM_USER@$VM_IP" 'mkdir -p ~/.ssh && chmod 700 ~/.ssh && key=$(cat) && grep -qF "$key" ~/.ssh/authorized_keys 2>/dev/null || echo "$key" >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys' <"$VM_SSHKEY"
fi

SSH() { ssh -p "$VM_SSH_PORT" -o BatchMode=yes "$VM_USER@$VM_IP" "$@"; }

# ---- guest packages ----

log "Installing Debian packages inside guest"
SSH "sudo -n true" 2>/dev/null || fail "User '$VM_USER' must have passwordless sudo. Add: '$VM_USER ALL=(ALL) NOPASSWD:ALL' to /etc/sudoers.d/$VM_USER inside the VM, then re-run."

SSH "sudo apt-get update -qq && sudo apt-get install -y -qq \
  build-essential binutils-gold git curl make ca-certificates rsync unzip \
  dbus systemd \
  incus podman"
SSH "sudo incus admin init --auto"
SSH "sudo systemctl enable --now incus-user.socket >/dev/null 2>&1 || true"

# ---- Go + Bun + repo tooling ----

log "Bootstrapping repo toolchain inside guest (delegates to scripts/install-tools.sh)"
if [ -z "$GO_SHA256" ]; then
  case "$GO_VERSION/$DEB_ARCH" in
    1.26.2/arm64) GO_SHA256="c958a1fe1b361391db163a485e21f5f228142d6f8b584f6bef89b26f66dc5b23" ;;
    1.26.2/amd64) GO_SHA256="990e6b4bbba816dc3ee129eaeaf4b42f17c2800b88a2166c265ac1a200262282" ;;
    *) fail "HELLING_GO_SHA256 is required when HELLING_GO_VERSION is not 1.26.2 for $DEB_ARCH" ;;
  esac
fi
GO_TARBALL="go${GO_VERSION}.linux-${DEB_ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"
SSH "if ! command -v go >/dev/null 2>&1 || [ \"\$(go env GOVERSION 2>/dev/null)\" != \"go$GO_VERSION\" ]; then \
  tmp=\$(mktemp -d) && \
  curl -fsSL '$GO_URL' -o \"\$tmp/$GO_TARBALL\" && \
  printf '%s  %s\n' '$GO_SHA256' \"\$tmp/$GO_TARBALL\" | sha256sum -c - && \
  sudo rm -rf /usr/local/go && \
  sudo tar -C /usr/local -xzf \"\$tmp/$GO_TARBALL\" && \
  rm -rf \"\$tmp\"; \
fi && \
sudo sh -c 'printf \"%s\n\" \"export PATH=/usr/local/go/bin:\\\$HOME/go/bin:\\\$HOME/.local/bin:\\\$HOME/.bun/bin:\\\$PATH\" > /etc/profile.d/helling-dev-tools.sh'"

# Stream the host copy of install-tools.sh into the guest so the source of truth stays
# in one place. The script is idempotent so re-running on each bootstrap is fine.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SSH "export PATH=/usr/local/go/bin:\$HOME/go/bin:\$HOME/.local/bin:\$HOME/.bun/bin:\$PATH && cat > /tmp/install-tools.sh && bash /tmp/install-tools.sh go && bash /tmp/install-tools.sh frontend" <"$SCRIPT_DIR/install-tools.sh"

# ---- hellingd systemd unit drop-in ----

log "Laying down hellingd systemd unit drop-in"
SSH "sudo install -d -m 0755 /etc/systemd/system && sudo tee /etc/systemd/system/hellingd.service >/dev/null <<'UNIT'
[Unit]
Description=Helling backend daemon (dev)
After=network.target dbus.service incus-user.socket
Wants=dbus.service incus-user.socket

[Service]
Type=simple
ExecStart=/usr/local/bin/hellingd
Restart=on-failure
User=hellingd
Group=hellingd
StateDirectory=helling
StateDirectoryMode=0750
RuntimeDirectory=helling
RuntimeDirectoryMode=0755
EnvironmentFile=-/etc/helling/hellingd.env

[Install]
WantedBy=multi-user.target
UNIT
sudo getent group helling-proxy >/dev/null || sudo groupadd --system helling-proxy
sudo getent passwd hellingd >/dev/null || sudo useradd --system --no-create-home --shell /usr/sbin/nologin hellingd
sudo install -d -m 0755 /etc/helling
sudo usermod -aG hellingd '$VM_USER'
sudo usermod -aG helling-proxy '$VM_USER'
sudo usermod -aG helling-proxy hellingd
sudo usermod -aG incus hellingd
sudo systemctl daemon-reload"

done_ "VM '$VM_NAME' is ready at $VM_IP"
echo
echo "Export this for vm:parallels:* tasks:"
echo "  export HELLING_VM_HOST=$VM_IP"
echo "  export HELLING_VM_USER=$VM_USER"
if [ "$VM_SSH_PORT" != "22" ]; then
  echo "  export HELLING_VM_SSH_PORT=$VM_SSH_PORT"
fi
echo
echo "Next:"
echo "  task vm:parallels:dev      # build, sync, restart"
echo "  task vm:parallels:smoke    # health probe"
echo "  task vm:parallels:logs     # journalctl -fu hellingd"
