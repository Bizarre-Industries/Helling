#!/usr/bin/env bash
# Build or validate the Packer-managed Debian Parallels dev VM image.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PACKER_DIR="$REPO_ROOT/deploy/packer/parallels-dev"

VM_NAME="${HELLING_VM_NAME:-helling-dev}"
VM_CPUS="${HELLING_VM_CPUS:-4}"
VM_MEM_MB="${HELLING_VM_MEM_MB:-8192}"
VM_DISK_GB="${HELLING_VM_DISK_GB:-40}"
VM_USER="${HELLING_VM_USER:-helling}"
VM_SSHKEY="${HELLING_VM_SSHKEY:-$HOME/.ssh/id_ed25519.pub}"
VM_SSH_PRIVATE_KEY="${HELLING_VM_SSH_PRIVATE_KEY:-${VM_SSHKEY%.pub}}"
DEBIAN_VERSION="${HELLING_PACKER_DEBIAN_VERSION:-13.4.0}"
DEBIAN_ARCH="${HELLING_PACKER_DEBIAN_ARCH:-}"
ISO_URL="${HELLING_PACKER_ISO_URL:-}"
ISO_CHECKSUM="${HELLING_PACKER_ISO_CHECKSUM:-}"
GO_VERSION="${HELLING_GO_VERSION:-1.26.2}"
GO_SHA256="${HELLING_GO_SHA256:-}"
OUTPUT_DIR="${HELLING_PACKER_OUTPUT_DIR:-$REPO_ROOT/dist/packer/$VM_NAME}"
STARTUP_VIEW="${HELLING_PACKER_STARTUP_VIEW:-headless}"
FORCE="${HELLING_PACKER_FORCE:-0}"
REPLACE_REGISTERED="${HELLING_PACKER_REPLACE_REGISTERED:-0}"

MODE="build"
while [ "$#" -gt 0 ]; do
  case "$1" in
    --init-only) MODE="init" ;;
    --validate-only) MODE="validate" ;;
    --register-only) MODE="register" ;;
    -h | --help)
      cat <<'USAGE'
Usage: scripts/parallels-vm-build-dev.sh [--init-only|--validate-only|--register-only]

Build a Debian 13 Parallels dev VM image for Helling with Packer.

Useful environment:
  HELLING_VM_NAME, HELLING_VM_CPUS, HELLING_VM_MEM_MB, HELLING_VM_DISK_GB
  HELLING_VM_USER, HELLING_VM_SSHKEY, HELLING_VM_SSH_PRIVATE_KEY
  HELLING_PACKER_DEBIAN_ARCH, HELLING_PACKER_DEBIAN_VERSION
  HELLING_PACKER_ISO_URL, HELLING_PACKER_ISO_CHECKSUM
  HELLING_PACKER_OUTPUT_DIR, HELLING_PACKER_FORCE=1
  HELLING_PACKER_REPLACE_REGISTERED=1  allow force rebuild while VM name is registered
USAGE
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      exit 1
      ;;
  esac
  shift
done

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}
have() { command -v "$1" >/dev/null 2>&1; }

case "$VM_NAME" in
  '' | *[!A-Za-z0-9._-]*) fail "HELLING_VM_NAME may contain only letters, numbers, dot, underscore, and dash" ;;
esac
[[ "$VM_USER" =~ ^[a-z_][a-z0-9_-]{0,31}$ ]] || fail "HELLING_VM_USER must be a valid Debian user name"
for numeric in VM_CPUS VM_MEM_MB VM_DISK_GB; do
  value="${!numeric}"
  case "$value" in
    '' | *[!0-9]*) fail "HELLING_${numeric#VM_} must be numeric (got: $value)" ;;
  esac
done

if [ -z "$DEBIAN_ARCH" ]; then
  case "$(uname -m)" in
    x86_64 | amd64) DEBIAN_ARCH="amd64" ;;
    arm64 | aarch64) DEBIAN_ARCH="arm64" ;;
    *) fail "set HELLING_PACKER_DEBIAN_ARCH; cannot map host arch $(uname -m)" ;;
  esac
fi
case "$DEBIAN_ARCH" in
  amd64 | arm64) ;;
  *) fail "HELLING_PACKER_DEBIAN_ARCH must be amd64 or arm64" ;;
esac
if [ "$DEBIAN_VERSION" != "13.4.0" ] && { [ -z "$ISO_URL" ] || [ -z "$ISO_CHECKSUM" ]; }; then
  fail "HELLING_PACKER_DEBIAN_VERSION=$DEBIAN_VERSION requires both HELLING_PACKER_ISO_URL and HELLING_PACKER_ISO_CHECKSUM"
fi

if [ "$MODE" != "register" ]; then
  have packer || fail "packer not found. Install with: brew install hashicorp/tap/packer"
fi
if [ "$MODE" != "validate" ] && [ "$MODE" != "init" ]; then
  have prlctl || fail "prlctl not found. Install Parallels Desktop."
fi

SSH_PUBLIC_KEY=""
if [ "$MODE" != "register" ]; then
  [ -f "$VM_SSHKEY" ] || fail "SSH public key not found: $VM_SSHKEY"
  [ -f "$VM_SSH_PRIVATE_KEY" ] || fail "SSH private key not found: $VM_SSH_PRIVATE_KEY (set HELLING_VM_SSH_PRIVATE_KEY)"
  SSH_PUBLIC_KEY="$(sed -n '1p' "$VM_SSHKEY")"
fi
SSH_PASSWORD=""
SSH_PASSWORD_HASH=""
if [ "$MODE" != "register" ]; then
  SSH_PASSWORD="$(openssl rand -base64 32)"
  SSH_PASSWORD_HASH="$(printf '%s\n' "$SSH_PASSWORD" | openssl passwd -6 -stdin)"
fi
DISK_SIZE_MB="$((VM_DISK_GB * 1024))"

packer_vars=(
  "-var=vm_name=$VM_NAME"
  "-var=debian_arch=$DEBIAN_ARCH"
  "-var=debian_version=$DEBIAN_VERSION"
  "-var=iso_url=$ISO_URL"
  "-var=iso_checksum=$ISO_CHECKSUM"
  "-var=cpus=$VM_CPUS"
  "-var=memory_mb=$VM_MEM_MB"
  "-var=disk_size_mb=$DISK_SIZE_MB"
  "-var=output_directory=$OUTPUT_DIR"
  "-var=ssh_username=$VM_USER"
  "-var=ssh_password_hash=$SSH_PASSWORD_HASH"
  "-var=ssh_public_key=$SSH_PUBLIC_KEY"
  "-var=ssh_private_key_file=$VM_SSH_PRIVATE_KEY"
  "-var=go_version=$GO_VERSION"
  "-var=go_sha256=$GO_SHA256"
  "-var=startup_view=$STARTUP_VIEW"
)

register_pvm() {
  local pvm=""
  if [ -d "$OUTPUT_DIR/$VM_NAME.pvm" ]; then
    pvm="$OUTPUT_DIR/$VM_NAME.pvm"
  else
    pvm="$(find "$OUTPUT_DIR" -maxdepth 2 -type d -name '*.pvm' -print -quit 2>/dev/null || true)"
  fi
  [ -n "$pvm" ] || fail "could not find a .pvm under $OUTPUT_DIR"

  if prlctl list -a 2>/dev/null | awk '{print $NF}' | grep -qx "$VM_NAME"; then
    done_ "Parallels VM '$VM_NAME' is already registered"
    return
  fi
  log "Registering $pvm as Parallels VM '$VM_NAME'"
  prlctl register "$pvm"
  done_ "Registered '$VM_NAME'"
}

if [ "$MODE" = "register" ]; then
  register_pvm
  exit 0
fi

log "Initializing Packer plugins"
packer init "$PACKER_DIR"
[ "$MODE" = "init" ] && exit 0

log "Validating Packer template"
packer validate "${packer_vars[@]}" "$PACKER_DIR"
[ "$MODE" = "validate" ] && exit 0

if prlctl list -a 2>/dev/null | awk '{print $NF}' | grep -qx "$VM_NAME"; then
  if [ "$FORCE" = "1" ] && [ "$REPLACE_REGISTERED" = "1" ]; then
    log "HELLING_PACKER_REPLACE_REGISTERED=1: deleting registered VM '$VM_NAME' before rebuild"
    prlctl stop "$VM_NAME" >/dev/null 2>&1 || true
    prlctl delete "$VM_NAME"
  else
    fail "Parallels VM '$VM_NAME' already exists. Set HELLING_VM_NAME for a side-by-side build, or explicitly delete/snapshot it and rerun with HELLING_PACKER_FORCE=1 HELLING_PACKER_REPLACE_REGISTERED=1."
  fi
fi
if [ -e "$OUTPUT_DIR" ] && [ "$FORCE" != "1" ]; then
  fail "Packer output already exists: $OUTPUT_DIR. Set HELLING_PACKER_FORCE=1 to rebuild."
fi

build_args=()
[ "$FORCE" = "1" ] && build_args+=("-force")

log "Building Parallels Debian dev image ($DEBIAN_ARCH, Debian $DEBIAN_VERSION)"
packer build "${build_args[@]}" "${packer_vars[@]}" "$PACKER_DIR"
register_pvm
done_ "Packer dev image ready"
echo "Next:"
echo "  task vm:parallels:up"
echo "  export HELLING_VM_HOST=<vm-ip if prlctl cannot resolve it>"
