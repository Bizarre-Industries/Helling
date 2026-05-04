#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

require_file() {
  [ -f "$REPO_ROOT/$1" ] || fail "missing $1"
}

require_executable() {
  [ -x "$REPO_ROOT/$1" ] || fail "$1 must be executable"
}

check_json() {
  local file="$1"
  if command -v jq >/dev/null 2>&1; then
    jq empty "$REPO_ROOT/$file" >/dev/null || fail "$file is not valid JSON"
  else
    python3 -m json.tool "$REPO_ROOT/$file" >/dev/null || fail "$file is not valid JSON"
  fi
}

check_toml_dir() {
  local dir="$1"
  python3 - "$REPO_ROOT/$dir" <<'PY'
import pathlib
import sys

try:
    import tomllib
except ModuleNotFoundError as exc:
    raise SystemExit(f"tomllib missing: {exc}")

root = pathlib.Path(sys.argv[1])
for path in sorted(root.glob("*.toml")):
    with path.open("rb") as f:
        tomllib.load(f)
PY
}

check_mcp_json() {
  python3 - "$REPO_ROOT/.mcp.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    cfg = json.load(f)
servers = cfg.get("mcpServers", {})
for key in ("openaiDeveloperDocs", "context7", "codex-peer"):
    if key not in servers:
        raise SystemExit(f".mcp.json missing MCP server {key}")
context7_args = servers["context7"].get("args", [])
if "@upstash/context7-mcp@2.2.3" not in context7_args:
    raise SystemExit(".mcp.json must pin @upstash/context7-mcp@2.2.3")
PY
}

require_file AGENTS.md
require_file CLAUDE.md
require_file .mcp.json
require_file .codex/config.toml
require_file .claude/settings.json
require_file .codex/hooks.json
require_file .claude/agents/README.md
require_file .codex/agents/README.md
require_file scripts/run-peer-agent.sh
require_file scripts/agent-hooks/mark-replanned.sh
require_file scripts/agent-hooks/never-guess.sh
require_file scripts/agent-hooks/replan-on-tag.sh
require_file scripts/agent-hooks/post-bash-version-shipped.sh
require_executable scripts/run-peer-agent.sh
require_executable .claude/hooks/mark-replanned.sh
require_executable .claude/hooks/never-guess.sh
require_executable .claude/hooks/replan-on-tag.sh
require_executable .codex/hooks/mark-replanned.sh
require_executable .codex/hooks/never-guess.sh
require_executable .codex/hooks/replan-on-tag.sh

check_json .mcp.json
check_json .claude/settings.json
check_json .codex/hooks.json
check_mcp_json
python3 - "$REPO_ROOT/.claude/settings.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    cfg = json.load(f)
disabled = set(cfg.get("disabledMcpjsonServers", []))
for key in ("context7", "codex-peer"):
    if key not in disabled:
        raise SystemExit(f".claude/settings.json must keep {key} opt-in via disabledMcpjsonServers")
PY
python3 - "$REPO_ROOT/.codex/config.toml" <<'PY'
import sys
import tomllib

with open(sys.argv[1], "rb") as f:
    cfg = tomllib.load(f)
for key in ("openaiDeveloperDocs", "context7", "claude-code"):
    if key not in cfg.get("mcp_servers", {}):
        raise SystemExit(f".codex/config.toml missing MCP server {key}")
context7_args = cfg["mcp_servers"]["context7"].get("args", [])
if "@upstash/context7-mcp@2.2.3" not in context7_args:
    raise SystemExit(".codex/config.toml must pin @upstash/context7-mcp@2.2.3")
for key in ("context7", "claude-code"):
    if cfg["mcp_servers"][key].get("enabled") is not False:
        raise SystemExit(f".codex/config.toml must keep {key} disabled by default")
PY
check_toml_dir .codex/agents

for script in \
  scripts/run-peer-agent.sh \
  scripts/agent-hooks/mark-replanned.sh \
  scripts/agent-hooks/never-guess.sh \
  scripts/agent-hooks/replan-on-tag.sh \
  scripts/agent-hooks/post-bash-version-shipped.sh \
  scripts/claude-hooks/post-bash-version-shipped.sh \
  .claude/hooks/mark-replanned.sh \
  .claude/hooks/never-guess.sh \
  .claude/hooks/replan-on-tag.sh \
  .codex/hooks/mark-replanned.sh \
  .codex/hooks/never-guess.sh \
  .codex/hooks/replan-on-tag.sh; do
  bash -n "$REPO_ROOT/$script" || fail "$script has shell syntax errors"
done

claude_commands="$(cd "$REPO_ROOT/.claude/commands" && find . -maxdepth 1 -type f -name '*.md' -print | sort)"
codex_commands="$(cd "$REPO_ROOT/.codex/commands" && find . -maxdepth 1 -type f -name '*.md' -print | sort)"
if [ "$claude_commands" != "$codex_commands" ]; then
  fail ".claude/commands and .codex/commands have different command files"
fi

if rg -n '\$\{ARGUMENTS\}|\$ARGUMENTS' "$REPO_ROOT/.claude/commands" "$REPO_ROOT/.codex/commands" >/dev/null; then
  rg -n '\$\{ARGUMENTS\}|\$ARGUMENTS' "$REPO_ROOT/.claude/commands" "$REPO_ROOT/.codex/commands" >&2
  fail "slash commands must not splice raw ARGUMENTS into shell commands"
fi

if rg -n '<claude-mem-context>|</claude-mem-context>|<environment_context>|</environment_context>|<subagent_notification>' "$REPO_ROOT/AGENTS.md" "$REPO_ROOT/CLAUDE.md" >/dev/null; then
  rg -n '<claude-mem-context>|</claude-mem-context>|<environment_context>|</environment_context>|<subagent_notification>' "$REPO_ROOT/AGENTS.md" "$REPO_ROOT/CLAUDE.md" >&2
  fail "durable agent manuals include transient session context"
fi

if [ -d "$REPO_ROOT/.claude/agents" ]; then
  claude_agents="$(cd "$REPO_ROOT/.claude/agents" && find . -maxdepth 1 -type f -name '*.md' ! -name README.md -print | sed 's/[.]md$//' | sort)"
  codex_agents="$(cd "$REPO_ROOT/.codex/agents" && find . -maxdepth 1 -type f -name '*.toml' -print | sed 's/[.]toml$//' | sort)"
  if [ "$claude_agents" != "$codex_agents" ]; then
    fail ".claude/agents and .codex/agents define different agent names"
  fi
fi

if rg -n '\.Codex|scripts/Codex-hooks' "$REPO_ROOT/AGENTS.md" "$REPO_ROOT/CLAUDE.md" "$REPO_ROOT/.codex" >/dev/null; then
  rg -n '\.Codex|scripts/Codex-hooks' "$REPO_ROOT/AGENTS.md" "$REPO_ROOT/CLAUDE.md" "$REPO_ROOT/.codex" >&2
  fail "stale uppercase Codex references remain in active agent config"
fi

printf '✓ agent tooling config is coherent\n'
