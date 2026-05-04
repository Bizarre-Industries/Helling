#!/usr/bin/env bash
# never-guess.sh — PreToolUse hook fired on Bash + WebFetch. Doesn't
# block. Injects an additionalContext reminder when the proposed action
# involves an external tool whose behavior may have drifted since the
# LLM's training cutoff, or matches a "I think this works" pattern.
#
# Per CLAUDE.md / AGENTS.md: LLM training data drifts. Compilation passing is not
# correctness. This hook is a nudge, not a wall — the agent still acts,
# but with a reminder to verify upstream before relying on the action.
#
# Payload (stdin): JSON with tool_name, tool_input. Output (stdout):
# JSON with hookSpecificOutput.additionalContext when a reminder is
# warranted, otherwise nothing.

set -euo pipefail

# Read the hook payload. If `jq` isn't present, no-op (don't block the
# session over a missing dep — log the absence to stderr).
if ! command -v jq >/dev/null 2>&1; then
  echo "never-guess.sh: jq not found, skipping reminder injection" >&2
  exit 0
fi

if [ -t 0 ]; then
  exit 0
fi

payload="$(cat)"

tool_name="$(echo "$payload" | jq -r '.tool_name // empty')"
command_str="$(echo "$payload" | jq -r '.tool_input.command // .tool_input.url // empty')"

if [ -z "$command_str" ]; then
  exit 0
fi

# External tools whose CLI surface has historically drifted between
# Helling's relevant cycles. If a Bash invokes one of these, inject a
# verification reminder. Pattern is greedy on purpose — false positives
# cost a few extra tokens, false negatives cost a buggy commit.
declare -a watch_tools=(
  "incus"
  "incusd"
  "podman"
  "podman-system-service"
  "gh"
  "git-cliff"
  "cliff"
  "golangci-lint"
  "oapi-codegen"
  "openapi-ts"
  "vacuum"
  "goose"
  "lefthook"
  "bun"
  "task"
  "prlctl"
  "parallels"
  "reprepro"
  "live-build"
  "lb"
  "cosign"
  "syft"
  "sbom"
  "dch"
  "dput"
  "dpkg-buildpackage"
  "debuild"
  "shellcheck"
  "shfmt"
  "yamllint"
  "markdownlint-cli2"
  "prettier"
  "hellingd"
  "helling-cli"
  "caddy"
)

# Risky shell patterns regardless of tool.
declare -a watch_patterns=(
  "curl.*\|.*sh"
  "wget.*\|.*sh"
  "--force"
  "-f .* origin"
  "rm -rf"
  "git push.*--force"
  "git reset --hard"
  "git tag -d"
  "git branch -D"
  "sudo "
)

reminder=""

for t in "${watch_tools[@]}"; do
  if echo "$command_str" | grep -qE "(^|[^[:alnum:]_/-])${t}([^[:alnum:]_-]|$)"; then
    reminder="External tool '${t}' invoked. Its CLI surface may have drifted since the training cutoff. Before relying on output shape or flag behavior, verify against upstream: \`man ${t}\`, \`${t} --help\`, or the tool's GitHub repo at HEAD. If the behavior is critical (parsing output, depending on a flag, mutating state), run a smoke check first. Compilation/exit-0 is not correctness."
    break
  fi
done

# WebFetch-specific override: reminder is different — encourage breadth
# of sources rather than verifying flag behavior.
if [ "$tool_name" = "WebFetch" ]; then
  reminder="Single-source web fetch. If this is a load-bearing claim (decides architecture, security, or a release), cross-check against at least one other source before relying on it. Vendor blog posts can be aspirational; release notes and source code are authoritative."
fi

for p in "${watch_patterns[@]}"; do
  if echo "$command_str" | grep -qE -- "$p"; then
    if [ -n "$reminder" ]; then
      reminder="${reminder} Additionally, the command matches a high-risk shell pattern ('${p}'). Confirm this is intended; the disk-bricking deny-list catches only the most catastrophic cases -- '--force' / '-D' / 'reset --hard' destroy work without prompting."
    else
      reminder="Command matches a high-risk shell pattern ('${p}'). Confirm this is intended and reversible. The disk-bricking deny-list catches only catastrophic cases -- '--force' / '-D' / 'reset --hard' destroy work without prompting."
    fi
    break
  fi
done

if [ -z "$reminder" ]; then
  exit 0
fi

# Emit non-blocking reminder. permissionDecision omitted = allow per
# defaultMode.
jq -n --arg ctx "$reminder" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    additionalContext: $ctx
  }
}'
