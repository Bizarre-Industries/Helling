#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="${HELLING_ISO_WORKDIR:-$REPO_ROOT/dist/iso/live-build}"
OUTDIR="${HELLING_ISO_OUTDIR:-$REPO_ROOT/dist/iso}"
VERSION="${HELLING_VERSION:-$(git -C "$REPO_ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
ARCH="${HELLING_ISO_ARCH:-}"
VERIFY_ONLY=0
PREPARE_ONLY=0

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
fail() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Usage: scripts/build-iso.sh [--verify-only] [--prepare-only]

Build the Helling Debian installer ISO with live-build.

Environment:
  HELLING_ISO_ARCH     Debian architecture, default: dpkg --print-architecture
  HELLING_VERSION      ISO version label, default: git describe
  HELLING_ISO_WORKDIR  live-build working directory, default: dist/iso/live-build
  HELLING_ISO_OUTDIR   output directory, default: dist/iso
  HELLING_SKIP_WEB_BUILD=1  reuse web/dist instead of running Bun build
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --verify-only) VERIFY_ONLY=1 ;;
    --prepare-only) PREPARE_ONLY=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    *) fail "unknown argument: $1" ;;
  esac
  shift
done

require_file() {
  [ -f "$REPO_ROOT/$1" ] || fail "missing required file: $1"
}

verify_sources() {
  require_file deploy/iso/auto/config
  require_file deploy/iso/auto/clean
  require_file deploy/iso/config/includes.binary/preseed/helling.cfg
  require_file deploy/iso/config/package-lists/helling.list.chroot
  require_file deploy/install/helling-first-boot.sh
  require_file deploy/install/helling-first-boot.service
  require_file deploy/install/hellingd.service
  require_file deploy/install/Caddyfile
  bash -n "$REPO_ROOT/scripts/build-iso.sh"
  bash -n "$REPO_ROOT/deploy/install/helling-first-boot.sh"
  sh -n "$REPO_ROOT/deploy/iso/auto/config"
  sh -n "$REPO_ROOT/deploy/iso/auto/clean"
}

detect_arch() {
  if [ -n "$ARCH" ]; then
    printf '%s\n' "$ARCH"
    return
  fi
  if command -v dpkg >/dev/null 2>&1; then
    dpkg --print-architecture
    return
  fi
  case "$(uname -m)" in
    x86_64) printf 'amd64\n' ;;
    arm64 | aarch64) printf 'arm64\n' ;;
    *) fail "set HELLING_ISO_ARCH; cannot map uname -m=$(uname -m)" ;;
  esac
}

goarch_from_deb() {
  case "$1" in
    amd64) printf 'amd64\n' ;;
    arm64) printf 'arm64\n' ;;
    *) fail "unsupported ISO architecture: $1" ;;
  esac
}

canonical_path() {
  python3 -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "$1"
}

prepare_workdir() {
  local deb_arch="$1"
  local go_arch
  go_arch="$(goarch_from_deb "$deb_arch")"

  mkdir -p "$OUTDIR"
  local out_real
  local work_real
  out_real="$(canonical_path "$OUTDIR")"
  work_real="$(canonical_path "$WORKDIR")"
  case "$work_real" in
    "$out_real"/*) ;;
    *) fail "refusing to remove ISO workdir outside $out_real: $work_real" ;;
  esac

  rm -rf "$WORKDIR"
  mkdir -p "$WORKDIR" "$OUTDIR"
  rsync -a "$REPO_ROOT/deploy/iso/" "$WORKDIR/"
  mkdir -p "$WORKDIR/config/includes.binary/helling/target-root/usr/lib/helling"
  mkdir -p "$WORKDIR/config/includes.binary/helling/target-root/usr/bin"
  mkdir -p "$WORKDIR/config/includes.binary/helling/target-root/usr/share/helling/web"
  mkdir -p "$WORKDIR/config/includes.binary/helling/target-root/etc/systemd/system"
  mkdir -p "$WORKDIR/config/includes.binary/helling/target-root/usr/share/helling"

  install -m 0755 "$REPO_ROOT/deploy/install/helling-first-boot.sh" \
    "$WORKDIR/config/includes.binary/helling/target-root/usr/lib/helling/helling-first-boot"
  install -m 0644 "$REPO_ROOT/deploy/install/hellingd.service" \
    "$WORKDIR/config/includes.binary/helling/target-root/etc/systemd/system/hellingd.service"
  install -m 0644 "$REPO_ROOT/deploy/install/helling-first-boot.service" \
    "$WORKDIR/config/includes.binary/helling/target-root/etc/systemd/system/helling-first-boot.service"
  install -m 0644 "$REPO_ROOT/deploy/install/Caddyfile" \
    "$WORKDIR/config/includes.binary/helling/target-root/usr/share/helling/Caddyfile"
  log "Building linux/$go_arch binaries for ISO payload"
  (
    cd "$REPO_ROOT"
    GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
      go build -trimpath -ldflags="-s -w -X main.version=$VERSION" \
      -o "$WORKDIR/config/includes.binary/helling/target-root/usr/lib/helling/hellingd" ./apps/hellingd
    GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
      go build -trimpath -ldflags="-s -w -X main.version=$VERSION" \
      -o "$WORKDIR/config/includes.binary/helling/target-root/usr/bin/helling" ./apps/helling-cli
  )

  if [ -f "$REPO_ROOT/web/package.json" ]; then
    if [ "${HELLING_SKIP_WEB_BUILD:-0}" != "1" ]; then
      log "Building WebUI for ISO payload"
      (cd "$REPO_ROOT/web" && bun install --frozen-lockfile && bun run build)
    fi
    [ -d "$REPO_ROOT/web/dist" ] || fail "web/dist missing; run web build or unset HELLING_SKIP_WEB_BUILD"
    rsync -a "$REPO_ROOT/web/dist/" \
      "$WORKDIR/config/includes.binary/helling/target-root/usr/share/helling/web/"
  fi
}

build_iso() {
  local deb_arch="$1"
  command -v lb >/dev/null 2>&1 || fail "live-build not installed. On Debian: sudo apt-get install live-build"
  (
    cd "$WORKDIR"
    HELLING_ISO_ARCH="$deb_arch" HELLING_VERSION="$VERSION" ./auto/config
  )

  log "Running live-build"
  if [ "$(id -u)" -eq 0 ]; then
    (cd "$WORKDIR" && lb build)
  elif command -v sudo >/dev/null 2>&1; then
    (cd "$WORKDIR" && sudo env HELLING_ISO_ARCH="$deb_arch" HELLING_VERSION="$VERSION" lb build)
  else
    fail "lb build requires root; install sudo or run as root"
  fi

  local iso
  iso="$(find "$WORKDIR" -maxdepth 1 -type f \( -name 'live-image-*.hybrid.iso' -o -name 'live-image-*.iso' \) -print -quit)"
  [ -n "$iso" ] || fail "live-build completed without an ISO artifact"
  local out="$OUTDIR/helling-$VERSION-$deb_arch.iso"
  cp "$iso" "$out"
  done_ "ISO written to $out"
  sign_iso "$out"
}

sign_iso() {
  local iso_path="$1"
  if [ "${HELLING_ISO_SIGN:-1}" = "0" ]; then
    log "warning: HELLING_ISO_SIGN=0; skipping detached ISO signature"
    return
  fi
  command -v gpg >/dev/null 2>&1 || fail "gpg is required to create detached ISO signatures (set HELLING_ISO_SIGN=0 only for local experiments)"
  gpg --batch --yes --armor --detach-sign "$iso_path"
  done_ "Detached signature written to $iso_path.asc"
}

main() {
  verify_sources
  if [ "$VERIFY_ONLY" = "1" ]; then
    done_ "ISO source validation passed"
    return
  fi

  local deb_arch
  deb_arch="$(detect_arch)"
  prepare_workdir "$deb_arch"
  if [ "$PREPARE_ONLY" = "1" ]; then
    done_ "live-build workdir prepared at $WORKDIR"
    return
  fi
  build_iso "$deb_arch"
}

main "$@"
