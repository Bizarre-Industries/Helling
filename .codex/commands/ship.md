---
description: Pre-push gate — format, lint, generated-drift, tests, security, then push HEAD
allowed-tools: Bash(make:*), Bash(git:*)
---

Run the full ship gate before pushing.

```bash
!`make fmt && make check && make check-generated && make security-fast`
```

If all green, push the current branch to `Helling HEAD`.

```bash
!`git push -- Helling HEAD`
```

Stop on first failure. Do NOT bypass with `--no-verify`.
