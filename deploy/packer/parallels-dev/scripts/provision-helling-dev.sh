#!/usr/bin/env bash
set -euo pipefail

dev_user="${HELLING_VM_USER:-helling}"
go_version="${HELLING_GO_VERSION:-1.26.2}"
go_sha256="${HELLING_GO_SHA256:-}"
debian_arch="${HELLING_DEBIAN_ARCH:-}"
tools_flavor="${HELLING_PARALLELS_TOOLS_FLAVOR:-lin}"
zabbly_fingerprint=4EFC590696CB15B87C73A3AD82CC8797C838DCFD
apt_timeout_opts=(-o Acquire::http::Timeout=30 -o Acquire::https::Timeout=30 -o DPkg::Lock::Timeout=120)

log() { printf '>> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

case "$debian_arch" in
  amd64 | arm64) ;;
  '') case "$(uname -m)" in
    x86_64 | amd64) debian_arch="amd64" ;;
    aarch64 | arm64) debian_arch="arm64" ;;
    *) fail "unsupported guest arch: $(uname -m)" ;;
  esac ;;
  *) fail "unsupported Debian arch: $debian_arch" ;;
esac

if [ -z "$go_sha256" ]; then
  case "$go_version/$debian_arch" in
    1.26.2/arm64) go_sha256="c958a1fe1b361391db163a485e21f5f228142d6f8b584f6bef89b26f66dc5b23" ;;
    1.26.2/amd64) go_sha256="990e6b4bbba816dc3ee129eaeaf4b42f17c2800b88a2166c265ac1a200262282" ;;
    *) fail "HELLING_GO_SHA256 is required when HELLING_GO_VERSION is not 1.26.2 for $debian_arch" ;;
  esac
fi

log "installing Debian packages"
export DEBIAN_FRONTEND=noninteractive
apt-get "${apt_timeout_opts[@]}" update -qq
apt-get "${apt_timeout_opts[@]}" install -y -qq \
  binutils-gold build-essential ca-certificates curl dbus git gnupg jq libcap2-bin \
  live-build make nftables podman rsync sqlite3 sudo systemd unzip

install -d -m 0755 /etc/apt/keyrings
curl --connect-timeout 10 --max-time 30 -fsSL https://pkgs.zabbly.com/key.asc -o /etc/apt/keyrings/zabbly.asc
fingerprint="$(gpg --show-keys --with-colons /etc/apt/keyrings/zabbly.asc | awk -F: '$1 == "fpr" { print $10; exit }')"
[ "$fingerprint" = "$zabbly_fingerprint" ] || fail "unexpected Zabbly key fingerprint: $fingerprint"
cat >/etc/apt/sources.list.d/zabbly-incus-stable.sources <<'SOURCES'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: trixie
Components: main
Architectures: amd64 arm64
Signed-By: /etc/apt/keyrings/zabbly.asc
SOURCES

apt-get "${apt_timeout_opts[@]}" update -qq
apt-get "${apt_timeout_opts[@]}" install -y -qq caddy incus
systemctl enable --now incus >/dev/null
incus admin init --auto >/dev/null
systemctl enable --now incus-user.socket >/dev/null
systemctl enable caddy >/dev/null

log "installing Go $go_version"
go_tarball="go${go_version}.linux-${debian_arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "https://go.dev/dl/${go_tarball}" -o "$tmp/$go_tarball"
printf '%s  %s\n' "$go_sha256" "$tmp/$go_tarball" | sha256sum -c -
rm -rf /usr/local/go
tar -C /usr/local -xzf "$tmp/$go_tarball"

cat >/etc/profile.d/helling-dev-tools.sh <<'PROFILE'
export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$HOME/.bun/bin:$PATH
PROFILE

log "configuring Helling service accounts"
for group in helling helling-proxy; do
  getent group "$group" >/dev/null || groupadd --system "$group"
done
id -u helling >/dev/null 2>&1 || useradd --system --gid helling --home-dir /var/lib/helling --shell /usr/sbin/nologin helling
id -u hellingd >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin hellingd
usermod -aG helling,helling-proxy,incus "$dev_user" 2>/dev/null || true
usermod -aG helling-proxy,incus helling 2>/dev/null || true
usermod -aG helling,helling-proxy,incus hellingd 2>/dev/null || true

install -d -m 0755 /etc/systemd/system /etc/helling
cat >/etc/systemd/system/hellingd.service <<'UNIT'
[Unit]
Description=Helling backend daemon
Documentation=https://github.com/Bizarre-Industries/Helling
After=network-online.target incus.service incus-user.socket
Wants=network-online.target incus.service incus-user.socket

[Service]
Type=simple
User=helling
Group=helling
SupplementaryGroups=helling-proxy incus
UMask=0007
ExecStart=/usr/lib/helling/hellingd -config /etc/helling/helling.yaml
Restart=on-failure
RestartSec=5
RuntimeDirectory=helling
RuntimeDirectoryMode=0755
StateDirectory=helling
StateDirectoryMode=0750
LogsDirectory=helling
LogsDirectoryMode=0750
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ReadWritePaths=/var/lib/helling /var/log/helling /run/helling /etc/helling /etc/systemd/system
ReadOnlyPaths=/usr/bin /usr/lib/helling
CapabilityBoundingSet=CAP_NET_ADMIN
AmbientCapabilities=
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
SystemCallArchitectures=native
MemoryDenyWriteExecute=true
NoNewPrivileges=false
LimitNOFILE=65535
LimitNPROC=4096
TasksMax=4096

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload

if [ -f /tmp/install-tools.sh ]; then
  log "installing repo toolchain for $dev_user"
  chmod 0755 /tmp/install-tools.sh
  sudo -u "$dev_user" -H env \
    PATH="/usr/local/go/bin:/home/$dev_user/go/bin:/home/$dev_user/.local/bin:/home/$dev_user/.bun/bin:$PATH" \
    bash /tmp/install-tools.sh go
  sudo -u "$dev_user" -H env \
    PATH="/usr/local/go/bin:/home/$dev_user/go/bin:/home/$dev_user/.local/bin:/home/$dev_user/.bun/bin:$PATH" \
    bash /tmp/install-tools.sh frontend
fi

log "locking provisioning password and disabling SSH password authentication"
passwd -l "$dev_user" >/dev/null
install -d -m 0755 /etc/ssh/sshd_config.d
cat >/etc/ssh/sshd_config.d/99-helling-dev-key-only.conf <<'SSHD'
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
SSHD
systemctl reload ssh >/dev/null 2>&1 || systemctl reload sshd >/dev/null 2>&1 || true

tools_iso="/home/$dev_user/prl-tools-${tools_flavor}.iso"
if [ -f "$tools_iso" ]; then
  log "attempting Parallels Tools install"
  apt-get install -y -qq dkms perl "linux-headers-$(uname -r)" >/dev/null 2>&1 || true
  mkdir -p /mnt/prl-tools
  if mount -o loop "$tools_iso" /mnt/prl-tools; then
    if [ -x /mnt/prl-tools/install ]; then
      /mnt/prl-tools/install --install-unattended-with-deps >/dev/null 2>&1 \
        || /mnt/prl-tools/install --install-unattended >/dev/null 2>&1 \
        || warn "Parallels Tools installer did not complete"
    fi
    umount /mnt/prl-tools || true
  fi
fi

log "Helling Parallels dev image provisioning complete"
