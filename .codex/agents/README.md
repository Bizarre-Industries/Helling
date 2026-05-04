# `.codex/agents/` — Helling Codex Agent Roster

These Codex agent definitions mirror the Claude Code council roles in
`.claude/agents/`.

Use them only when the trigger list in `AGENTS.md` requires review:

- `council-architect`
- `council-security`
- `council-perf-skeptic`
- `council-devil-advocate`
- `council-ux-critic`
- `self-critique`
- `mechanical`

When changing trigger rules or role instructions, update both the Claude and
Codex versions and run:

```sh
task check:agents
```
