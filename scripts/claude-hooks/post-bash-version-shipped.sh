#!/usr/bin/env bash
# scripts/claude-hooks/post-bash-version-shipped.sh
#
# Claude Code PostToolUse hook (ADR-053).
#
# Fires after every Bash tool call. If the just-run command pushed a v* tag
# (matching `git push --tags` or `git push <remote> v*` / `git push <remote>
# refs/tags/v*`), this hook:
#   1. Runs `task plan:next-version` to regenerate the deterministic snapshot
#      at docs/plans/v<next>-plan.md.
#   2. Emits an additional-context blob telling Claude to enter plan mode and
#      author the full tracking plan for the next gate.
#
# When the command does not match, the hook is a no-op (exit 0, no output).
#
# Wiring: registered in .claude/settings.json under PostToolUse[matcher=Bash].
# Env input: CLAUDE_TOOL_INPUT_command (the Bash command that just ran).

set -euo pipefail

cmd="${CLAUDE_TOOL_INPUT_command:-}"

# Tag-push detection. Conservative — only matches push commands that include a
# tag reference, not arbitrary `git tag -s` (which is local-only).
case "$cmd" in
  *"git push"*"--tags"* | \
    *"git push"*"refs/tags/v"[0-9]* | \
    *"git push "*" v"[0-9]*)
    : # match
    ;;
  *)
    exit 0
    ;;
esac

# Resolve repo root from the hook's location (scripts/claude-hooks/.. = repo).
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

# Run the deterministic snapshot generator. Failures here should not block the
# agent; emit the additional context regardless so the agent at least knows a
# tag shipped.
plan_path=""
if command -v task >/dev/null 2>&1; then
  if out="$(task plan:next-version 2>/dev/null)"; then
    plan_path="$(printf '%s\n' "$out" | tail -n1)"
  fi
fi

cat <<EOF
Version tag shipped (ADR-053).
$([ -n "$plan_path" ] && printf 'Deterministic snapshot regenerated at: %s\n' "$plan_path")
Required next action: enter plan mode and author the tracking plan for the next version gate.
1. Read docs/roadmap/checklist.md to identify the next gate.
2. Read the deterministic snapshot above (open items, recent ADRs, open audits, prior plans).
3. Re-analyze repo + specs + plans for any state drift since the last plan.
4. Synthesize a phased implementation plan into the snapshot file (overwrite the skeleton).
5. Exit plan mode for user approval before any implementation work.
EOF
