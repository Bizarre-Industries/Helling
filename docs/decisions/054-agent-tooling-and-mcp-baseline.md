# ADR-054: Agent tooling and MCP baseline

> Status: Accepted (2026-05-04)
>
> Relates to: ADR-050, ADR-053.

## Context

Helling uses both Claude Code and Codex for repository automation. The prior
setup had project hooks and commands split across agent-specific directories,
which caused drift in release-plan hooks, command behavior, council-agent
instructions, and MCP server availability.

The repository also needs a clear policy for agent-side MCP execution. MCP
servers can execute local code or proxy privileged agent actions, so the
baseline must be explicit, reviewable, and pinned where executable packages are
used.

## Decision

Project-shared agent tooling is committed in both agent-native shapes:

- Claude Code: `.claude/settings.json`, `.claude/commands/`, `.claude/hooks/`,
  `.claude/skills/`, and `.claude/agents/`.
- Codex: `.codex/config.toml`, `.codex/hooks.json`, `.codex/commands/`,
  `.codex/hooks/`, and `.codex/agents/`.
- Shared hook implementation lives in `scripts/agent-hooks/`; agent-specific
  hook files are wrappers.
- `scripts/sync-agent-tooling.sh` is the local coherence gate and is wired into
  `task check:agents`.

The default MCP baseline is:

- `openaiDeveloperDocs` over the official OpenAI developer-docs MCP endpoint.
- `context7` through `npx -y @upstash/context7-mcp@2.2.3`; executable MCP npm
  packages must be pinned to an exact version in project config.
- `codex-peer` for Claude Code, running `codex mcp-server` through
  `scripts/run-peer-agent.sh`.
- `claude-code` for Codex, running `claude mcp serve` through
  `scripts/run-peer-agent.sh`.

The peer launcher carries a recursion guard and validates the requested peer
tool. It is for local agent collaboration only; product code must never call
it.

ADR-050 is also clarified: `hellingd` is not installed as `incus-admin`.
Privileged Incus administration remains a future privileged-helper design, not
default daemon group membership.

## Consequences

Easier:

- Claude Code and Codex start from the same project commands, hooks, council
  roles, and MCP assumptions.
- Hook drift is reduced because shell logic lives in one shared directory.
- Agent MCP supply-chain risk is visible in committed config and checked by
  `task check:agents`.

Harder / costs:

- Updating agent commands or council roles requires changing both formats and
  running `task check:agents`.
- The Context7 MCP version must be bumped deliberately when upstream changes
  are needed, and it stays disabled by default until its process lifecycle is
  reverified.
- Peer-agent MCP is intentionally opt-in. The launcher remains for manual use,
  but default MCP registration is disabled until start/initialize/list/cancel
  and disconnect behavior has smoke coverage.

## Alternatives Considered

- **Keep agent config local-only.** Rejected because this repo already depends
  on repeatable hooks, slash commands, and council roles for release work. Local
  drift caused the broken hook state that this ADR fixes.
- **Enable Context7 by default.** Rejected for now. Current library docs are
  useful during OpenAPI, frontend, and agent-tooling work, but executable MCPs
  must not be default-on while an upstream process-lifecycle regression is under
  review. Keep the exact pin and require a deliberate local enablement.
- **Default peer-agent MCP bridge.** Rejected for now. Claude Code and Codex both
  expose MCP server modes intended for local tool composition, but the bridge is
  default-disabled until lifecycle smoke coverage proves process exit and cancel
  behavior. Use `scripts/run-peer-agent.sh` manually for sensitive reviews.

## References

- `.mcp.json`
- `.codex/config.toml`
- `.claude/settings.json`
- `.codex/hooks.json`
- `scripts/agent-hooks/`
- `scripts/run-peer-agent.sh`
- `scripts/sync-agent-tooling.sh`
