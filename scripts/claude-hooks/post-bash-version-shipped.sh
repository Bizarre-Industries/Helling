#!/usr/bin/env bash
set -euo pipefail

repo="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "$0")/../.." && pwd)}"
exec bash "$repo/scripts/agent-hooks/post-bash-version-shipped.sh" "$@"
