#!/usr/bin/env bash
# mark-replanned.sh <tag> — called by Claude/Codex after the next-version
# plan file has been rewritten and the first execution-step commit has
# landed. Advances both agent marker files so the SessionStart +
# Stop hook stops re-prompting.

set -euo pipefail

tag="${1:?usage: mark-replanned.sh <tag>}"
repo="${CODEX_PROJECT_DIR:-${CLAUDE_PROJECT_DIR:-$(pwd)}}"

if [[ ! "$tag" =~ ^v[0-9]+[.][0-9]+[.][0-9]+(-[A-Za-z0-9._-]+)?$ ]]; then
  echo "invalid release tag: $tag" >&2
  exit 2
fi

mkdir -p "$repo/.claude" "$repo/.codex"
echo "$tag" >"$repo/.claude/.last-shipped-tag"
echo "$tag" >"$repo/.codex/.last-shipped-tag"
echo "marked replanned at $tag"
