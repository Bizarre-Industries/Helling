---
description: Advance .last-shipped-tag after a v* tag's next-version plan is rewritten
allowed-tools: Bash(bash:*), Bash(.codex/hooks/*)
argument-hint: <tag>
---

Run after the next-version plan file has been rewritten and the first
execution-step commit has landed. Stops the SessionStart hook from
re-prompting about the same shipped tag.

Validate the argument as a release tag matching
`v<major>.<minor>.<patch>[-suffix]`, then run the hook with that literal tag
as one quoted argv value. Do not splice raw slash-command arguments into a
shell command.

Example:

```bash
!`bash .codex/hooks/mark-replanned.sh 'v0.1.0-alpha'`
```
