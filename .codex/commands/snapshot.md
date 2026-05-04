---
description: Print the current docs/plans/checklist snapshot (also fires on SessionStart)
allowed-tools: Bash(bash:*), Bash(scripts/*)
---

Render the same context blob the SessionStart hook injects: release-gate
snapshot, WebUI audit status, recent ADRs, working-tree dirtiness.

```bash
!`bash scripts/docs-snapshot.sh`
```

Use this to re-orient on the active milestone without restarting the session.
