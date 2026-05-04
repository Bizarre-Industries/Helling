#!/usr/bin/env bash
# Run the other local coding CLI as an MCP server with a recursion guard.

set -euo pipefail

depth="${HELLING_PEER_AGENT_DEPTH:-0}"
case "$depth" in
  '' | *[!0-9]*)
    echo "peer agent recursion depth is invalid" >&2
    exit 2
    ;;
esac
if [ "$depth" -ge 1 ]; then
  echo "peer agent recursion blocked" >&2
  exit 2
fi

tool="${1:-}"
if [ -z "$tool" ]; then
  echo "usage: scripts/run-peer-agent.sh claude|codex [args...]" >&2
  exit 2
fi
shift

chain="${HELLING_PEER_AGENT_CHAIN:-}"
case ",$chain," in
  *",$tool,"*)
    echo "peer agent recursion blocked for $tool" >&2
    exit 2
    ;;
esac
export HELLING_PEER_AGENT_DEPTH=$((depth + 1))
if [ -n "$chain" ]; then
  export HELLING_PEER_AGENT_CHAIN="$chain,$tool"
else
  export HELLING_PEER_AGENT_CHAIN="$tool"
fi

case "$tool" in
  claude)
    exe="$(command -v claude || true)"
    ;;
  codex)
    exe="$(command -v codex || true)"
    if [ -z "$exe" ] && [ -x /Applications/Codex.app/Contents/Resources/codex ]; then
      exe="/Applications/Codex.app/Contents/Resources/codex"
    fi
    ;;
  *)
    echo "unknown peer agent '$tool'. Expected claude or codex." >&2
    exit 2
    ;;
esac

if [ -z "$exe" ]; then
  echo "$tool CLI is not installed or not on PATH" >&2
  exit 127
fi

exec "$exe" "$@"
