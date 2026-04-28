#!/usr/bin/env bash
# scripts/check-webui-tracking.sh
#
# Enforces F-ID ↔ checklist consistency for WebUI audit work.
#
# Modes:
#   default                  — non-strict; warns and exits 0
#   --strict                 — fails on any unticked F-ID referenced by a commit
#   --range <git-range>      — scan a specific commit range (default: @{u}..HEAD)
#   --staged <file>          — scan a single commit-message file (commit-msg hook)
#
# Used by:
#   - lefthook.yml pre-push (strict)
#   - .claude/settings.local.json PostToolUse on `git commit` / `gh pr create` (warn)
#   - manual:  scripts/check-webui-tracking.sh --range origin/main..HEAD
#
# Behavior:
#   1. Greps commit messages in range for F-\d+ identifiers.
#   2. For each F-ID, checks docs/roadmap/checklist.md for `[x] **F-XX**`
#      or an inline `✅` marker.
#   3. If any F-ID is referenced in commits but unticked, prints a reminder.
#   4. In strict mode, exits non-zero.
#
# Cross-references:
#   docs/audits/webui-2026-04-27.md     — canonical record
#   docs/plans/webui-phase-2-6.md       — per-finding fix sketches
#   docs/roadmap/checklist.md           — per-PR ticking surface

set -euo pipefail

STRICT=0
RANGE="@{u}..HEAD"
MODE="range"
STAGED_FILE=""

while [ $# -gt 0 ]; do
  case "$1" in
    --strict)
      STRICT=1
      shift
      ;;
    --range)
      RANGE="$2"
      shift 2
      ;;
    --staged)
      MODE="staged"
      STAGED_FILE="$2"
      shift 2
      ;;
    -h | --help)
      sed -n '1,30p' "$0"
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

REPO_ROOT="$(git rev-parse --show-toplevel)"
CHECKLIST="$REPO_ROOT/docs/roadmap/checklist.md"
AUDIT="$REPO_ROOT/docs/audits/webui-2026-04-27.md"

if [ ! -f "$CHECKLIST" ] || [ ! -f "$AUDIT" ]; then
  # Tracking files absent — likely not in a Helling clone; skip silently.
  exit 0
fi

# Collect F-IDs referenced by commits or staged message.
if [ "$MODE" = "staged" ]; then
  IDS="$(grep -oE 'F-[0-9]+' "$STAGED_FILE" 2>/dev/null | sort -u || true)"
else
  # Fall back to last 5 commits if upstream not set.
  if ! git rev-parse --verify --quiet "$RANGE" >/dev/null 2>&1; then
    RANGE="HEAD~5..HEAD"
  fi
  IDS="$(git log --format='%B' "$RANGE" 2>/dev/null | grep -oE 'F-[0-9]+' | sort -u || true)"
fi

if [ -z "$IDS" ]; then
  exit 0
fi

UNTICKED=""
for id in $IDS; do
  # Strict-tick pattern: a line containing "[x] **F-XX**" in checklist.
  if ! grep -qE "^\s*-\s*\[x\]\s*\*\*${id}\*\*" "$CHECKLIST"; then
    # Also accept inline `✅` marker on a line mentioning the F-ID.
    if ! grep -qE "${id}.*✅" "$CHECKLIST"; then
      UNTICKED="${UNTICKED}${id} "
    fi
  fi
done

if [ -z "$UNTICKED" ]; then
  exit 0
fi

cat >&2 <<EOF
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WebUI tracking reminder
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Commit(s) reference these F-IDs but the checklist still shows them open:
  ${UNTICKED}

Update docs/roadmap/checklist.md (tick the matching line with the commit SHA)
and re-run. See:
  docs/audits/webui-2026-04-27.md
  docs/plans/webui-phase-2-6.md
  docs/roadmap/checklist.md
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

if [ "$STRICT" = "1" ]; then
  exit 1
fi
exit 0
