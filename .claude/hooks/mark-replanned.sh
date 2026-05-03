#!/usr/bin/env bash
# mark-replanned.sh <tag> — called by Claude after the next-version
# plan file has been rewritten and the first execution-step commit has
# landed. Advances .claude/.last-shipped-tag so the SessionStart +
# Stop hook stops re-prompting.

set -euo pipefail

tag="${1:?usage: mark-replanned.sh <tag>}"
repo="${CLAUDE_PROJECT_DIR:-$(pwd)}"

mkdir -p "$repo/.claude"
echo "$tag" >"$repo/.claude/.last-shipped-tag"
echo "marked replanned at $tag"
