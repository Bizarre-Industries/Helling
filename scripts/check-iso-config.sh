#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

bash "$REPO_ROOT/scripts/build-iso.sh" --verify-only

if ! grep -q '/cdrom/helling/target-root' "$REPO_ROOT/deploy/iso/config/includes.binary/preseed/helling.cfg"; then
  echo "FAIL: ISO preseed does not install the Helling target-root payload"
  exit 1
fi

if ! grep -q 'socket_group: helling-proxy' "$REPO_ROOT/deploy/install/helling-first-boot.sh"; then
  echo "FAIL: first-boot config does not grant Caddy access through the helling-proxy socket group"
  exit 1
fi

if ! grep -q 'setup_token_path: /etc/helling/setup-token' "$REPO_ROOT/deploy/install/helling-first-boot.sh"; then
  echo "FAIL: first-boot config does not wire the installer setup token path"
  exit 1
fi

if [ -f "$REPO_ROOT/deploy/install/50-helling.rules" ]; then
  echo "FAIL: ISO install must not ship broad polkit manage-units rules"
  exit 1
fi

if ! grep -q 'canonical_path' "$REPO_ROOT/scripts/build-iso.sh" \
  || ! grep -q 'detach-sign' "$REPO_ROOT/scripts/build-iso.sh"; then
  echo "FAIL: ISO builder must canonicalize workdir paths and emit detached signatures"
  exit 1
fi

echo "✓ ISO installer config is coherent"
