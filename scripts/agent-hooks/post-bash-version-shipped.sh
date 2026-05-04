#!/usr/bin/env bash
# scripts/agent-hooks/post-bash-version-shipped.sh
#
# Claude Code / Codex PostToolUse hook (ADR-053).
#
# Fires after every Bash tool call. If the just-run command pushed a v* tag
# (matching `git push --tags` or `git push <remote> v*` / `git push <remote>
# refs/tags/v*`), this hook:
#   1. Runs `task plan:next-version` to regenerate the deterministic snapshot
#      at docs/plans/v<next>-plan.md.
#   2. Emits an additional-context blob telling the agent to enter plan mode and
#      author the full tracking plan for the next gate.
#
# When the command does not match, the hook is a no-op (exit 0, no output).
#
# Wiring: registered in .claude/settings.json and .codex/hooks.json.
# Env input: CLAUDE_TOOL_INPUT_command or CODEX_TOOL_INPUT_command.
# Stdin fallback: JSON hook payload with tool_input.command.

set -euo pipefail

cmd="${CODEX_TOOL_INPUT_command:-${CLAUDE_TOOL_INPUT_command:-}}"
if [ -z "$cmd" ] && command -v jq >/dev/null 2>&1 && [ ! -t 0 ]; then
  payload="$(cat)"
  if [ -n "$payload" ]; then
    cmd="$(printf '%s' "$payload" | jq -r '.tool_input.command // .tool_input.cmd // empty' 2>/dev/null || true)"
  fi
fi

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

# Resolve repo root from the hook's location (scripts/agent-hooks/.. = repo).
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

shipped_tag="$(printf '%s\n' "$cmd" | sed -nE 's/.*(refs\/tags\/)?(v[0-9]+[.][0-9]+[.][0-9]+(-[A-Za-z0-9._-]+)?).*/\2/p' | tail -n1)"
if [ -z "$shipped_tag" ]; then
  shipped_tag="$(git tag --list 'v*' --sort=-v:refname | head -n1 || true)"
fi
if [[ ! "$shipped_tag" =~ ^v[0-9]+[.][0-9]+[.][0-9]+(-[A-Za-z0-9._-]+)?$ ]]; then
  exit 0
fi
if ! git verify-tag "$shipped_tag" >/dev/null 2>&1; then
  exit 0
fi

# Run the deterministic snapshot generator. Failures here should not block the
# agent; emit the additional context regardless so the agent at least knows a
# tag shipped.
plan_path=""
if command -v task >/dev/null 2>&1; then
  if out="$(HELLING_SHIPPED_TAG="$shipped_tag" task plan:next-version 2>/dev/null)"; then
    plan_path="$(printf '%s\n' "$out" | tail -n1)"
  fi
fi

context="Version tag shipped (ADR-053)."
if [ -n "$plan_path" ]; then
  context="${context}
Deterministic snapshot regenerated at: ${plan_path}"
fi
context="${context}
Required next action: enter plan mode and author the tracking plan for the next version gate.
1. Read docs/roadmap/checklist.md to identify the next gate.
2. Read the deterministic snapshot above (open items, recent ADRs, open audits, prior plans).
3. Re-analyze repo + specs + plans for any state drift since the last plan.
4. Synthesize a phased implementation plan into the snapshot file (overwrite the skeleton).
5. Exit plan mode for user approval before any implementation work."

HOOK_CONTEXT="$context" python3 - <<'PY'
import json
import os

print(json.dumps({
    "hookSpecificOutput": {
        "hookEventName": "PostToolUse",
        "additionalContext": os.environ["HOOK_CONTEXT"],
    }
}))
PY
