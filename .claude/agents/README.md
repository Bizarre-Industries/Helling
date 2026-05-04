# `.claude/agents/` — Helling agent roster

Drives the council and self-critique pattern described in
[`CLAUDE.md`](../../CLAUDE.md). Read in this order:

1. **[`self-critique.md`](self-critique.md)** — fires before every
   commit. Single-shot, Opus-backed. Cheap. Finds the most likely
   failure mode in the diff. Default for non-escalated changes.
2. **The five council members** — fire in parallel, single-shot, Opus-backed.
   Trigger automatically on:
   - new ADR creation
   - new external dependency (`go.mod`, `web/package.json`, GH Actions, MCP)
   - breaking change to a service's public API or
     persisted-on-disk schema
   - code deletion >100 LOC in one commit
   - any change under `apps/hellingd/internal/auth/`, `api/openapi.yaml`, signing
     config, or `.github/workflows/release.yml`

   Members:
   - **[`council-architect.md`](council-architect.md)** — structural
     fit, ADR coverage, invariant adherence.
   - **[`council-security.md`](council-security.md)** — credential
     leakage, attack surface, signing.
   - **[`council-devil-advocate.md`](council-devil-advocate.md)** —
     argues the opposite decision was right.
   - **[`council-ux-critic.md`](council-ux-critic.md)** — only fires if
     the diff has a user-visible surface; otherwise sits this round
     out.
   - **[`council-perf-skeptic.md`](council-perf-skeptic.md)** —
     scaling, latency, blocking calls, memory.

3. **[`mechanical.md`](mechanical.md)** — Haiku-backed, cheap, fast.
   For routine rename/move/format/regex-replace ops. Doesn't make
   architectural decisions.

## Council deliberation flow

When a council fires:

1. The orchestrator (main loop) spawns all five council subagents in
   parallel with the same payload (diff + change description +
   pointer to active plan).
2. Each returns its structured verdict:

   ```text
   Decision: APPROVE | APPROVE-WITH-CONDITIONS | REJECT
   Rationale: ...
   Dissents: ...
   Risks: ...
   Conditions: ...
   ```

3. Orchestrator concatenates the verdicts and writes them to
   `.claude/council-logs/<YYYY-MM-DD>-<slug>.md`.
4. Orchestrator decides:
   - All five APPROVE → commit.
   - Any REJECT → halt, address dissents, re-run the rejecting
     member(s).
   - Any APPROVE-WITH-CONDITIONS → address the conditions, re-run if
     they were structural.

The log file is local runtime evidence and is not committed. Naming uses ISO
date plus a short slug. Example:
`.claude/council-logs/2026-05-04-adr-054-drop-jwt.md`.

## When the council theater alarm trips

If `Dissents: empty` shows up across all five members on three
substantive decisions in a row, the trigger list in `CLAUDE.md` is too
broad. Tune it tighter — fewer but heavier councils.

## Why no test-runner / linter agents

`make test`, `make fmt`, `gitleaks`, `make lint`, `vacuum lint` are
commands, not agents. They're invoked by the main loop
directly. Agents are for things that need _judgment_, not deterministic
tools.

## Why not agent teams (the heavier multi-Claude-instance pattern)

Agent teams (per
[Anthropic docs](https://code.claude.com/docs/en/best-practices))
shine when you need teammates to talk to each other mid-task. The
Helling council is single-shot per member and orchestrator-mediated —
no teammate-to-teammate chatter, ~5-7x cheaper. If we ever need
closed-loop builder/reviewer rounds, that's a v0.4+ decision and gets
its own ADR.
