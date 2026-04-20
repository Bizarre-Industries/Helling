#!/usr/bin/env bash
# scripts/check-coverage.sh
#
# Enforce per-package Go coverage floors per standards-quality-assurance.md §5.2.
#
# Floors:
#   internal/handlers/  - 80% minimum
#   internal/services/  - 90% minimum
#   internal/clients/   - 70% minimum
#   overall             - 80% minimum
#
# Usage: bash scripts/check-coverage.sh coverage.out

set -euo pipefail

COVERAGE_FILE="${1:-coverage.out}"

if [ ! -f "$COVERAGE_FILE" ]; then
  echo "coverage file not found: $COVERAGE_FILE"
  echo "run: go test -race -coverprofile=coverage.out ./..."
  exit 1
fi

# Per-package floors. Format: "path-prefix:min-pct"
FLOORS=(
  "internal/handlers:80"
  "internal/services:90"
  "internal/clients:70"
)

OVERALL_FLOOR=80

# Parse `go tool cover -func` output.
# Format:  <path>:<line>.<col>,<line>.<col>  <funcname>  <pct>
# Sample:  github.com/Bizarre-Industries/Helling/internal/handlers/auth.go:42.1,48.2  Login  92.3%

coverage_output="$(go tool cover -func="$COVERAGE_FILE")"

if [ -z "$coverage_output" ]; then
  echo "no coverage data in $COVERAGE_FILE"
  exit 1
fi

overall_pct="$(echo "$coverage_output" | awk '/^total:/ {gsub("%","",$3); print $3}')"
echo "Overall coverage: ${overall_pct}%"

fail_count=0
check_floor() {
  local prefix="$1"
  local min="$2"

  local matched_pct
  matched_pct="$(echo "$coverage_output" \
    | grep -E "/${prefix}/" \
    | awk '{gsub("%","",$NF); sum += $NF; n++} END { if (n > 0) printf "%.1f\n", sum/n }')"

  if [ -z "$matched_pct" ]; then
    echo "○ $prefix — no files covered yet (skipping gate)"
    return 0
  fi

  local passed
  passed="$(awk -v a="$matched_pct" -v b="$min" 'BEGIN { print (a+0 >= b+0) ? 1 : 0 }')"

  if [ "$passed" = "1" ]; then
    echo "✓ $prefix — ${matched_pct}% (floor ${min}%)"
  else
    echo "✗ $prefix — ${matched_pct}% below floor ${min}%"
    fail_count=$((fail_count + 1))
  fi
}

echo ""
echo "Per-package coverage gates:"
for entry in "${FLOORS[@]}"; do
  prefix="${entry%:*}"
  min="${entry#*:}"
  check_floor "$prefix" "$min"
done

echo ""
overall_passed="$(awk -v a="$overall_pct" -v b="$OVERALL_FLOOR" 'BEGIN { print (a+0 >= b+0) ? 1 : 0 }')"
if [ "$overall_passed" = "1" ]; then
  echo "✓ overall — ${overall_pct}% (floor ${OVERALL_FLOOR}%)"
else
  echo "✗ overall — ${overall_pct}% below floor ${OVERALL_FLOOR}%"
  fail_count=$((fail_count + 1))
fi

if [ "$fail_count" -gt 0 ]; then
  echo ""
  echo "Coverage gate FAILED: $fail_count package(s) below floor."
  exit 1
fi

echo ""
echo "✓ All coverage gates passed."
