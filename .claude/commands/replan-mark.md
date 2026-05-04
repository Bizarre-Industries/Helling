---
description: Advance .last-shipped-tag after a v* tag's next-version plan is rewritten
allowed-tools: Bash(bash:*), Bash(.claude/hooks/*)
argument-hint: <tag>
---

Run after the next-version plan file has been rewritten and the first
execution-step commit has landed. Stops the SessionStart hook from
re-prompting about the same shipped tag.

```bash
!`bash .claude/hooks/mark-replanned.sh ${ARGUMENTS}`
```

Argument must be the tag name (e.g. `v0.1.0-alpha`).
