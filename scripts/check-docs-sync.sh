#!/usr/bin/env bash
# scripts/check-docs-sync.sh
#
# Enforces docs/plans/checklists ↔ source-code synchronization. Runs at
# git pre-push and editor PostToolUse to catch PRs that touch code without
# updating the corresponding docs surfaces.
#
# Two checks:
#
#   1. Generic docs-sync gate
#      If commits in range touch source areas (apps/, web/, api/openapi.yaml,
#      deploy/, scripts/) without also touching docs/, README, CONTRIBUTING,
#      CHANGELOG, AGENTS, SECURITY, LICENSE, or carrying a recognized
#      doc-bypass marker (chore/style/test/ci/build commit type, `Refs: F-XX`,
#      `Refs: ADR-NNN`, `Refs: #NN`, or `[skip-docs]`), warn or fail.
#
#   2. WebUI F-ID gate
#      Any commit referencing F-\d+ must have the matching line in
#      docs/roadmap/checklist.md ticked `[x] **F-XX**` or carry an inline
#      `✅` marker.
#
# Modes:
#   default                  — non-strict; warns and exits 0
#   --strict                 — fails on any miss
#   --range <git-range>      — scan a specific commit range (default: @{u}..HEAD)
#   --staged <file>          — scan a single commit-message file (commit-msg hook)
#
# Used by:
#   - lefthook.yml pre-push (strict)
#   - editor PostToolUse on `git commit` / `gh pr create` (warn-only via
#     local agent config, gitignored)
#   - manual:  scripts/check-docs-sync.sh --range origin/main..HEAD
#
# Cross-references:
#   docs/audits/webui-2026-04-27.md     — canonical WebUI audit
#   docs/plans/webui-phase-2-6.md       — per-finding fix sketches
#   docs/roadmap/checklist.md           — per-PR ticking surface
#   docs/spec/                          — domain specifications
#   docs/standards/                     — coding/security/QA standards
#   docs/decisions/                     — architecture decision records

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
      sed -n '1,40p' "$0"
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

cd "$REPO_ROOT"

# ────────────────────────────────────────────────────────────────────
# Resolve range
# ────────────────────────────────────────────────────────────────────
if [ "$MODE" = "range" ]; then
  if ! git rev-parse --verify --quiet "$RANGE" >/dev/null 2>&1; then
    RANGE="HEAD~5..HEAD"
  fi
fi

# ────────────────────────────────────────────────────────────────────
# Collect commit messages and changed files
# ────────────────────────────────────────────────────────────────────
if [ "$MODE" = "staged" ]; then
  MESSAGES="$(cat "$STAGED_FILE" 2>/dev/null || true)"
  CHANGED=""
else
  MESSAGES="$(git log --format='%B' "$RANGE" 2>/dev/null || true)"
  CHANGED="$(git diff --name-only "$RANGE" 2>/dev/null || true)"
fi

if [ -z "$MESSAGES" ] && [ -z "$CHANGED" ]; then
  exit 0
fi

WARNINGS=""

# ────────────────────────────────────────────────────────────────────
# Check 1 — generic docs-sync gate
# ────────────────────────────────────────────────────────────────────
SOURCE_HIT=0
DOCS_HIT=0

if [ -n "$CHANGED" ]; then
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    case "$f" in
      apps/* | web/* | deploy/* | api/openapi.yaml | api/*.go | scripts/*.sh) SOURCE_HIT=1 ;;
      docs/* | README.md | CONTRIBUTING.md | CHANGELOG.md | AGENTS.md | SECURITY.md | LICENSE) DOCS_HIT=1 ;;
    esac
  done <<<"$CHANGED"
fi

if [ "$SOURCE_HIT" = "1" ] && [ "$DOCS_HIT" = "0" ]; then
  BYPASS=0
  case "$MESSAGES" in
    *"[skip-docs]"* | *"Refs: F-"* | *"Refs: ADR-"* | *"Refs: #"*) BYPASS=1 ;;
  esac
  if echo "$MESSAGES" | head -n1 | grep -qE '^(chore|style|test|ci|build)(\(.+\))?!?: '; then
    BYPASS=1
  fi
  if [ "$BYPASS" = "0" ]; then
    WARNINGS="${WARNINGS}generic: commits touch source (apps/, web/, deploy/, api/openapi.yaml, scripts/) but no docs were updated.\n  Add a docs/* / README / CONTRIBUTING / CHANGELOG change, or include 'Refs: F-XX' / 'Refs: ADR-NNN' / 'Refs: #NN' in the commit body, or '[skip-docs]' if intentional.\n"
  fi
fi

# ────────────────────────────────────────────────────────────────────
# Check 2 — WebUI F-ID gate
#
# Only F-IDs declared via `Closes: F-XX` (closing claim) trigger the gate.
# `Refs: F-XX` / `Mentions: F-XX` / inline mentions are advisory only —
# they do not force a checklist tick. This keeps tracking-infrastructure
# commits (which may list dozens of F-IDs) from spuriously failing the gate.
# ────────────────────────────────────────────────────────────────────
if [ -f "$CHECKLIST" ]; then
  IDS="$(echo "$MESSAGES" | grep -oE '^[[:space:]]*Closes:[[:space:]]*F-[0-9]+' | grep -oE 'F-[0-9]+' | sort -u || true)"
  if [ -n "$IDS" ]; then
    UNTICKED=""
    for id in $IDS; do
      if ! grep -qE "^\s*-\s*\[x\]\s*\*\*${id}\*\*" "$CHECKLIST"; then
        if ! grep -qE "${id}.*✅" "$CHECKLIST"; then
          UNTICKED="${UNTICKED}${id} "
        fi
      fi
    done
    if [ -n "$UNTICKED" ]; then
      WARNINGS="${WARNINGS}webui: commit(s) declare 'Closes: F-XX' but checklist still shows them open: ${UNTICKED}\n  Tick docs/roadmap/checklist.md (line containing '[ ] **F-XX**') with the commit SHA and re-run.\n"
    fi
  fi
fi

# ────────────────────────────────────────────────────────────────────
# Report
# ────────────────────────────────────────────────────────────────────
if [ -z "$WARNINGS" ]; then
  exit 0
fi

cat >&2 <<EOF
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
docs-sync reminder
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF
printf '%b' "$WARNINGS" >&2
cat >&2 <<EOF
See:
  docs/audits/webui-2026-04-27.md (WebUI audit)
  docs/plans/webui-phase-2-6.md (per-finding sketches)
  docs/roadmap/checklist.md (release gates + per-PR ticking)
  docs/spec/ docs/standards/ docs/decisions/ (domain refs)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

if [ "$STRICT" = "1" ]; then
  exit 1
fi
exit 0
