#!/usr/bin/env bash
set -euo pipefail

repo="${CODEX_PROJECT_DIR:-${CLAUDE_PROJECT_DIR:-$(pwd)}}"
exec bash "$repo/scripts/agent-hooks/never-guess.sh" "$@"
