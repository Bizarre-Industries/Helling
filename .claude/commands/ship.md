---
description: Pre-push gate — format, lint, generated-drift, tests, security, then push
allowed-tools: Bash(make:*), Bash(git:*)
argument-hint: [remote] [refspec]
---

Run the full ship gate before pushing.

```bash
!`make fmt && make check && make check-generated && make security-fast`
```

If all green, push. Default target: `Helling HEAD` (current branch).

```bash
!`git push ${ARGUMENTS:-Helling HEAD}`
```

Stop on first failure. Do NOT bypass with `--no-verify`.
