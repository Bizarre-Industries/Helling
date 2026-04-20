#!/usr/bin/env bash
# scripts/check-parity.sh
#
# Enforce API ↔ CLI ↔ WebUI parity per docs/roadmap/phase0-parity-matrix.md.
# Every openapi.yaml operation must have either:
#   (a) a corresponding helling CLI command in docs/spec/cli.md, AND
#   (b) a corresponding WebUI route in docs/spec/webui-spec.md,
# OR
#   (c) an explicit exception in docs/roadmap/phase0-parity-exceptions.yaml.

set -euo pipefail

REPO="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
OPENAPI="${REPO}/api/openapi.yaml"
CLI_SPEC="${REPO}/docs/spec/cli.md"
WEBUI_SPEC="${REPO}/docs/spec/webui-spec.md"
EXCEPTIONS="${REPO}/docs/roadmap/phase0-parity-exceptions.yaml"

if [ ! -f "$OPENAPI" ]; then
  echo "○ api/openapi.yaml not present; skipping parity check"
  exit 0
fi

have_py() { command -v python3 >/dev/null 2>&1; }
have_py || {
  echo "python3 required for parity check"
  exit 1
}

# Extract operation IDs from openapi.yaml.
operation_ids() {
  python3 - "$1" <<'PYEOF'
import sys
import re

try:
    import yaml
    with open(sys.argv[1]) as f:
        doc = yaml.safe_load(f) or {}
    for path, methods in (doc.get('paths') or {}).items():
        if not isinstance(methods, dict):
            continue
        for method, op in methods.items():
            if method in ('get', 'post', 'put', 'patch', 'delete', 'head', 'options'):
                if isinstance(op, dict) and 'operationId' in op:
                    print(op['operationId'])
except ImportError:
    content = open(sys.argv[1]).read()
    for match in re.finditer(r'operationId:\s*([A-Za-z0-9_]+)', content):
        print(match.group(1))
PYEOF
}

# Read exception list as "op_id:missing_kind" pairs.
# Returns NOTHING if EXCEPTIONS file doesn't exist — this is a valid state,
# not an error. The previous version crashed with IndexError here.
exception_entries() {
  if [ ! -f "$EXCEPTIONS" ]; then
    return 0
  fi

  python3 - "$EXCEPTIONS" <<'PYEOF' || true
import sys
try:
    import yaml
except ImportError:
    sys.exit(0)

try:
    with open(sys.argv[1]) as f:
        doc = yaml.safe_load(f) or {}
except Exception:
    sys.exit(0)

for entry in (doc.get('exceptions') or []):
    op = entry.get('operation_id', '')
    missing = entry.get('missing', '')
    if op and missing:
        print(f"{op}:{missing}")
PYEOF
}

# Pre-compute exception sets.
CLI_EXEMPT=""
WEB_EXEMPT=""
while IFS=':' read -r op missing; do
  [ -z "${op:-}" ] && continue
  case "${missing:-}" in
    cli) CLI_EXEMPT="${CLI_EXEMPT}|${op}" ;;
    webui) WEB_EXEMPT="${WEB_EXEMPT}|${op}" ;;
    both)
      CLI_EXEMPT="${CLI_EXEMPT}|${op}"
      WEB_EXEMPT="${WEB_EXEMPT}|${op}"
      ;;
  esac
done < <(exception_entries)

CLI_EXEMPT="${CLI_EXEMPT#|}"
WEB_EXEMPT="${WEB_EXEMPT#|}"

is_exempt() {
  local op="$1" exempt="$2"
  [ -z "$exempt" ] && return 1
  echo "$exempt" | tr '|' '\n' | grep -qx "$op"
}

# camelCase → variants that might appear in docs.
# authLogin → "authLogin", "auth login", "auth-login", "auth_login", "auth/login"
op_variants() {
  local op="$1"
  # Insert a delimiter before each uppercase letter (except the first char), lowercase all.
  local spaced
  spaced=$(printf '%s' "$op" | python3 -c '
import sys, re
s = sys.stdin.read().strip()
# Split camelCase: insert space before each uppercase that follows a lowercase/digit.
out = re.sub(r"(?<=[a-z0-9])([A-Z])", r" \1", s).lower()
print(out)
')
  local kebab="${spaced// /-}"
  local snake="${spaced// /_}"
  local slash="${spaced// //}"
  # Unique variants (some collapse when op is single-word).
  printf '%s\n%s\n%s\n%s\n%s\n' "$op" "$spaced" "$kebab" "$snake" "$slash" | awk '!seen[$0]++'
}

in_spec() {
  local op="$1" spec="$2"
  [ ! -f "$spec" ] && return 1
  # operationId: X is a deterministic hit. Try it first.
  grep -qE "operationId:\s+${op}\b" "$spec" 2>/dev/null && return 0
  # Otherwise match any of the variants as a fixed substring (case-insensitive).
  while IFS= read -r variant; do
    [ -z "$variant" ] && continue
    grep -qiF -- "$variant" "$spec" 2>/dev/null && return 0
  done < <(op_variants "$op")
  return 1
}

in_cli_spec() {
  in_spec "$1" "$CLI_SPEC"
}

in_webui_spec() {
  in_spec "$1" "$WEBUI_SPEC"
}

gap_count=0
total=0

# Handle empty openapi gracefully (pre-Huma-spike state).
op_list=$(operation_ids "$OPENAPI" 2>/dev/null || true)
if [ -z "$op_list" ]; then
  echo "○ no operations defined in api/openapi.yaml; parity trivially satisfied"
  echo "✓ Parity gate passed (0 operations)."
  exit 0
fi

while IFS= read -r op; do
  [ -z "$op" ] && continue
  total=$((total + 1))

  cli_ok=false
  web_ok=false

  if in_cli_spec "$op"; then
    cli_ok=true
  elif is_exempt "$op" "$CLI_EXEMPT"; then
    cli_ok=true
  fi

  if in_webui_spec "$op"; then
    web_ok=true
  elif is_exempt "$op" "$WEB_EXEMPT"; then
    web_ok=true
  fi

  if ! $cli_ok || ! $web_ok; then
    gap_count=$((gap_count + 1))
    gaps=""
    $cli_ok || gaps="${gaps}cli "
    $web_ok || gaps="${gaps}webui"
    echo "✗ $op — missing: $gaps"
  fi
done <<<"$op_list"

echo ""
echo "Parity summary: $((total - gap_count))/$total operations have full parity or exception."

if [ "$gap_count" -gt 0 ]; then
  echo ""
  echo "Parity gate FAILED. Either:"
  echo "  1. Add the missing CLI command to $CLI_SPEC, or"
  echo "  2. Add the missing WebUI route to $WEBUI_SPEC, or"
  echo "  3. Add an entry to $EXCEPTIONS with reason + target version."
  exit 1
fi

echo "✓ Parity gate passed."
