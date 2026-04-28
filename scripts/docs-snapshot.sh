#!/usr/bin/env bash
# scripts/docs-snapshot.sh
#
# Prints a concise current-state summary of the project's docs/plans/
# checklists, intended for editor sanity-check hooks (SessionStart /
# UserPromptSubmit) so an agent has up-to-date context before making
# plans or deciding on tasks.
#
# Output is plain text on stdout; ~50 lines maximum. Skip silently if
# expected files are absent.
#
# Sources:
#   docs/roadmap/checklist.md           — release gates + per-PR ticking
#   docs/audits/webui-2026-04-27.md     — WebUI audit status snapshot
#   docs/plans/webui-phase-2-6.md       — WebUI per-phase plan
#   docs/decisions/                     — ADRs (most recent N)
#
# Usage:
#   bash scripts/docs-snapshot.sh

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$REPO_ROOT"

CHECKLIST="docs/roadmap/checklist.md"
AUDIT="docs/audits/webui-2026-04-27.md"
PLAN="docs/plans/webui-phase-2-6.md"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Helling docs/plans/checklists snapshot"
echo "Generated: $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Branch: $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
echo "HEAD: $(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -f "$CHECKLIST" ]; then
  echo
  echo "## Release gates ($CHECKLIST)"
  echo
  awk '
    /^## v[0-9].*Gate/ { if (gate) print gate, "[done="done", open="open"]"; gate=$0; done=0; open=0; next }
    /^- \[x\]/         { done++ }
    /^- \[ \]/         { open++ }
    END                { if (gate) print gate, "[done="done", open="open"]" }
  ' "$CHECKLIST"
fi

if [ -f "$AUDIT" ]; then
  echo
  echo "## WebUI audit ($AUDIT)"
  echo
  awk '
    /^## Status snapshot/ { inblock=1; print; next }
    inblock && /^## /     { inblock=0 }
    inblock               { print }
  ' "$AUDIT" | head -n 20
fi

if [ -f "$PLAN" ]; then
  echo
  echo "## WebUI plan ($PLAN)"
  grep -E '^## Phase [0-9]' "$PLAN" || true
fi

if [ -d docs/decisions ]; then
  echo
  echo "## Recent ADRs (docs/decisions/)"
  # shellcheck disable=SC2207
  adrs=($(printf '%s\n' docs/decisions/[0-9]*-*.md 2>/dev/null | sort -r))
  for adr in "${adrs[@]:0:5}"; do
    [ -e "$adr" ] && basename "$adr"
  done
fi

DIRTY="$(git status --short 2>/dev/null | wc -l | tr -d ' ')"
if [ "$DIRTY" != "0" ]; then
  echo
  echo "## Working tree: $DIRTY uncommitted change(s)"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Sanity check: read docs/roadmap/plan.md + above before deciding tasks."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
