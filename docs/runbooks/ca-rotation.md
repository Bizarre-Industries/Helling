# Runbook: Internal CA Rotation

## Purpose

Rotate Helling internal CA material used for per-user Incus trust certificate workflows.

## Preconditions

- Change window approved.
- Control-plane backup completed and verified.
- Admin access to host and services.

## Procedure

1. Confirm service health before change.
2. Create fresh control-plane backup.
3. Generate new CA keypair according to secret-handling standards.
4. Update Helling CA material and mark old CA as retiring.
5. Re-issue user client certificates signed by new CA.
6. Register new certs in Incus trust store with required restrictions.
7. Remove old trust entries after grace window.
8. Verify Incus proxy access for representative admin/user accounts.

## Validation

- `/api/v1/health` healthy.
- Authenticated `/api/incus/*` calls succeed for expected users.
- No authz scope regressions.

## Rollback

- Restore pre-rotation control-plane backup.
- Reapply prior CA material and trust entries.
- Restart services and revalidate health.

## Related docs

- `docs/spec/internal-ca.md` — key material types, validity windows, and rotation contract
- `docs/spec/proxies.md` — mTLS client identity passed through hellingd to the Incus upstream
