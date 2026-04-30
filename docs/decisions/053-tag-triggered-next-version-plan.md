# ADR-053: Tag-triggered next-version plan generation

> Status: Accepted (2026-04-30)
>
> Relates to: ADR-025 (signed APT repo, tag-driven publish), ADR-046 (live-build ISO release), ADR-052 (Parallels primary dev VM).

## Context

Helling has eight version gates documented in `docs/roadmap/checklist.md` (v0.1.0-alpha → v1.0.0) and a per-version ADR table in `docs/roadmap/plan.md`. After each release tag ships, the maintainer needs to:

1. Snapshot the repo state (open checklist items, ADR backlog, audit findings).
2. Decide what's in scope for the next gate.
3. Author a tracking plan that contributors can pick up from.

Today this is improvised. The risk: the gap between "tag pushed" and "next plan exists" is where momentum dies.

A fully automated plan generator that uses a language model to synthesize new direction is out of scope per the project's no-AI-references rule. A deterministic generator that snapshots state and produces a plan skeleton is in scope and removes the busywork.

## Decision

Every signed Helling release tag (`v[0-9]+\.[0-9]+\.[0-9]+(-\w+)?`) triggers a deterministic "next-version plan" generation step.

Components:

- **`scripts/plan-next-version.sh`** — deterministic local generator. Reads `docs/roadmap/checklist.md`, identifies the lowest version gate with open items, lists those open items, lists recent ADRs in `docs/decisions/`, lists open audits in `docs/audits/`, lists prior `docs/plans/*.md` files, and writes a snapshot file at `docs/plans/v<next-version>-plan.md`. No language-model usage; pure git + grep + awk.
- **`Taskfile.yaml` `plan:next-version`** — wraps the script for manual runs.
- **Claude Code PostToolUse hook** — `.claude/settings.json` registers a `PostToolUse: Bash` hook that reads `${CLAUDE_TOOL_INPUT_command}`, detects tag-push patterns (`git push --tags`, `git push origin v*`, `git push.*refs/tags/v*`), and runs `scripts/claude-hooks/post-bash-version-shipped.sh`. The hook script regenerates the deterministic snapshot via `task plan:next-version`, then emits an `additionalContext` blob telling Claude to enter plan mode and produce the full tracking plan for the next gate. The agent does the analysis + plan synthesis; the hook is the trigger.

The deterministic generator runs on every tag and produces the same factual snapshot regardless of who pushed. The Claude Code hook is the trigger that re-engages the agent for synthesis after each ship.

## Consequences

Easier:

- Every release tag automatically re-engages the agent for next-version planning. No improvised handoff.
- The deterministic snapshot encodes project review structure (open checklist items, recent ADRs, audits, prior plans) so the agent's synthesis starts from a consistent base.
- The same script runs locally (`task plan:next-version`) and via the Claude Code hook.

Harder / costs:

- The generator must stay in sync with the checklist heading convention (`## v<version> Gate`). Schema drift would silently break the generator. Mitigation: the script asserts on the heading regex and exits non-zero if no gate is found.
- Plan synthesis (sequencing, scope) is still human + agent work — the generator only assembles raw facts. The Claude Code hook hands those facts to the agent on tag push and asks for a full plan.
- The Claude Code hook lives in `.claude/settings.json` (project-shared, committed). Contributors who don't run Claude Code still get the deterministic snapshot via the GHA-free local task; they just don't get the agent re-engagement.

## References

- `scripts/plan-next-version.sh` — deterministic snapshot generator.
- `scripts/claude-hooks/post-bash-version-shipped.sh` — Claude Code PostToolUse hook body.
- `.claude/settings.json` — registers the PostToolUse hook on `Bash` tool calls.
- `Taskfile.yaml` `plan:next-version` — local entry point.
- `docs/roadmap/checklist.md` — source of open items.
- `docs/roadmap/plan.md` — ADR table.
