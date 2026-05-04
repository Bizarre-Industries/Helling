#!/usr/bin/env bash
set -euo pipefail

MARKER=/var/lib/helling/.first-boot-complete
ZABBLY_FINGERPRINT=4EFC590696CB15B87C73A3AD82CC8797C838DCFD
APT_TIMEOUT_OPTS=(-o Acquire::http::Timeout=30 -o Acquire::https::Timeout=30 -o DPkg::Lock::Timeout=120)

log() { printf 'helling-first-boot: %s\n' "$*"; }
fail() {
  printf 'helling-first-boot: ERROR: %s\n' "$*" >&2
  exit 1
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    fail "must run as root"
  fi
}

ensure_group() {
  local name="$1"
  if ! getent group "$name" >/dev/null; then
    groupadd --system "$name"
  fi
}

ensure_user() {
  local name="$1"
  local group="$2"
  local home="$3"
  if ! id -u "$name" >/dev/null 2>&1; then
    useradd --system --gid "$group" --home-dir "$home" --shell /usr/sbin/nologin "$name"
  fi
}

add_user_to_group_if_present() {
  local user_name="$1"
  local group_name="$2"
  if getent group "$group_name" >/dev/null; then
    usermod -aG "$group_name" "$user_name"
  fi
}

configure_zabbly_incus_repo() {
  if [ -f /etc/apt/sources.list.d/zabbly-incus-stable.sources ]; then
    return
  fi

  install -d -m 0755 /etc/apt/keyrings
  local bootstrap_missing=()
  local bootstrap_package
  for bootstrap_package in ca-certificates curl gnupg; do
    if ! dpkg-query -W -f='${Status}' "$bootstrap_package" 2>/dev/null | grep -q "install ok installed"; then
      bootstrap_missing+=("$bootstrap_package")
    fi
  done
  if [ "${#bootstrap_missing[@]}" -gt 0 ]; then
    if ! apt-get "${APT_TIMEOUT_OPTS[@]}" update -qq \
      || ! DEBIAN_FRONTEND=noninteractive apt-get "${APT_TIMEOUT_OPTS[@]}" install -y -qq "${bootstrap_missing[@]}"; then
      log "warning: could not install Zabbly key prerequisites; continuing with installer-provided packages"
      return
    fi
  fi
  if ! curl --connect-timeout 10 --max-time 30 -fsSL https://pkgs.zabbly.com/key.asc -o /etc/apt/keyrings/zabbly.asc; then
    log "warning: could not fetch Zabbly Incus key; continuing with installer-provided packages"
    return
  fi
  local fingerprint
  fingerprint="$(gpg --show-keys --with-colons /etc/apt/keyrings/zabbly.asc | awk -F: '$1 == "fpr" { print $10; exit }')"
  if [ "$fingerprint" != "$ZABBLY_FINGERPRINT" ]; then
    fail "unexpected Zabbly key fingerprint: $fingerprint"
  fi

  cat >/etc/apt/sources.list.d/zabbly-incus-stable.sources <<'SOURCES'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: trixie
Components: main
Architectures: amd64 arm64
Signed-By: /etc/apt/keyrings/zabbly.asc
SOURCES
}

ensure_packages() {
  local packages=(
    ca-certificates
    caddy
    curl
    dbus
    gnupg
    incus
    podman
    sudo
  )
  local missing=()
  local package
  for package in "${packages[@]}"; do
    if ! dpkg-query -W -f='${Status}' "$package" 2>/dev/null | grep -q "install ok installed"; then
      missing+=("$package")
    fi
  done
  if [ "${#missing[@]}" -eq 0 ]; then
    return
  fi
  apt-get "${APT_TIMEOUT_OPTS[@]}" update -qq
  DEBIAN_FRONTEND=noninteractive apt-get "${APT_TIMEOUT_OPTS[@]}" install -y -qq "${missing[@]}"
}

create_identities_and_paths() {
  ensure_group helling
  ensure_group helling-proxy
  ensure_group incus
  ensure_user helling helling /var/lib/helling

  add_user_to_group_if_present helling helling-proxy
  add_user_to_group_if_present helling incus
  if id -u caddy >/dev/null 2>&1; then
    add_user_to_group_if_present caddy helling-proxy
  fi

  install -d -o root -g helling -m 0750 /etc/helling
  install -d -o helling -g helling -m 0700 /etc/helling/age /etc/helling/certs
  install -d -o helling -g helling -m 0750 /var/lib/helling /var/log/helling
  install -d -o helling -g helling -m 0700 /var/lib/helling/jwt
  install -d -o helling -g helling-proxy -m 0755 /run/helling
}

write_setup_token_if_missing() {
  if [ ! -f /etc/helling/setup-token ]; then
    umask 0177
    dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\n' >/etc/helling/setup-token
    printf '\n' >>/etc/helling/setup-token
  fi
  chown root:helling /etc/helling/setup-token
  chmod 0660 /etc/helling/setup-token
}

write_config_if_missing() {
  if [ -f /etc/helling/helling.yaml ]; then
    chown root:helling /etc/helling/helling.yaml
    chmod 0640 /etc/helling/helling.yaml
    return
  fi
  cat >/etc/helling/helling.yaml <<'YAML'
state_dir: /var/lib/helling
server:
  socket_path: /run/helling/api.sock
  socket_group: helling-proxy
  socket_mode: 432
log:
  level: info
  format: json
auth:
  session_ttl_hours: 168
  access_ttl_minutes: 15
  jwt_signing_key_path: /var/lib/helling/jwt/ed25519.key
  setup_token_path: /etc/helling/setup-token
  login_rate_limit_per_15m: 5
  argon2_time_cost: 3
  argon2_memory_kib: 65536
  argon2_parallelism: 4
incus:
  socket_path: /var/lib/incus/user.socket
  project: default
YAML
  chown root:helling /etc/helling/helling.yaml
  chmod 0640 /etc/helling/helling.yaml
}

configure_services() {
  if [ -f /usr/share/helling/Caddyfile ]; then
    install -o root -g root -m 0644 /usr/share/helling/Caddyfile /etc/caddy/Caddyfile
  fi

  timeout 30 systemctl daemon-reload
  timeout 30 systemctl enable incus >/dev/null 2>&1 || true
  timeout 30 systemctl enable incus-user.socket >/dev/null 2>&1 || true
  timeout 30 systemctl enable podman.socket >/dev/null 2>&1 || true
  timeout 45 systemctl restart incus >/dev/null 2>&1 || true
  timeout 30 systemctl restart incus-user.socket >/dev/null 2>&1 || true
  timeout 30 systemctl restart podman.socket >/dev/null 2>&1 || true

  if command -v incus >/dev/null 2>&1; then
    timeout 30 incus admin init --auto >/dev/null 2>&1 || true
    timeout 30 incus config set core.https_address 127.0.0.1:8443 >/dev/null 2>&1 || true
  fi

  timeout 30 systemctl enable hellingd.service caddy.service >/dev/null
  timeout 30 systemctl restart hellingd.service
  timeout 30 systemctl restart caddy.service
}

health_gate() {
  for _ in $(seq 1 30); do
    if curl --connect-timeout 2 --max-time 5 -fsS --unix-socket /run/helling/api.sock http://helling/healthz >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
  curl --connect-timeout 2 --max-time 5 -fsS --unix-socket /run/helling/api.sock http://helling/healthz >/dev/null
  curl --connect-timeout 2 --max-time 5 -kfsS https://127.0.0.1:8006/healthz >/dev/null
}

main() {
  require_root
  if [ -f "$MARKER" ]; then
    log "already complete"
    exit 0
  fi

  log "configuring package repositories and required packages"
  configure_zabbly_incus_repo
  ensure_packages
  log "creating users, groups, and runtime paths"
  create_identities_and_paths
  write_setup_token_if_missing
  write_config_if_missing
  log "enabling platform services"
  configure_services
  log "verifying health"
  health_gate
  install -o root -g root -m 0644 /dev/null "$MARKER"
  systemctl disable helling-first-boot.service >/dev/null 2>&1 || true
  log "complete; open https://<host>:8006 and finish admin setup with the token in /etc/helling/setup-token"
}

main "$@"
