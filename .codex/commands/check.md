---
description: Run the full pre-merge gate (fmt, lint, race tests, generated drift)
allowed-tools: Bash(make:*)
---

Run Helling's full pre-merge verification gate. Mirrors what CI runs and
what `lefthook` runs on `git push`.

```bash
!`make check && make check-generated`
```

If anything fails, stop and surface the first failure verbatim. Do NOT
attempt fixes without explicit instruction.
