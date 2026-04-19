# Runbook: Upgrade Rollback

## Trigger Conditions

- Migration failure
- Post-upgrade health check failure
- Management-plane crash loop after package update

## Procedure

1. Capture current failure diagnostics (`journalctl`, service status).
2. Stop management services (`hellingd`, `caddy`).
3. Reinstall previous known-good package versions.
4. Restore pre-upgrade SQLite backup snapshot.
5. Start `hellingd` and verify migration state.
6. Start Caddy edge service.
7. Run health checks and proxy-path validation.

## Validation Checklist

- `hellingd` active
- Caddy active
- `/api/v1/health` returns success
- Auth flow works
- `/api/incus/*` and `/api/podman/*` proxy paths respond

## Post-Incident

- Record timeline and root cause.
- Create follow-up issue before retrying upgrade.
