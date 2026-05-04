---
description: Regenerate Go server, Go client, and TypeScript client from api/openapi.yaml
allowed-tools: Bash(make:*)
---

Regenerate code from `api/openapi.yaml`. Spec is single source of truth
(per `CLAUDE.md`); generated artifacts must stay in sync.

```bash
!`make generate`
```

Then verify drift is absent:

```bash
!`make check-generated`
```

If drift exists after generate, commit the regenerated files in the same PR
as the spec change.
