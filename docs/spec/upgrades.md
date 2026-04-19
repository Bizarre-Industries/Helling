# Upgrades Specification

Upgrade behavior for Helling management components in v0.1.

## Scope

- In scope: `hellingd`, `helling-proxy`, `helling` package upgrades.
- Out of scope: runtime workload migration orchestration for Incus/Podman workloads.

## Source and Packaging

- Packages are distributed through a signed APT repository.
- Standard upgrade path:

```bash
apt update && apt install --only-upgrade helling helling-proxy hellingd
```

## Release and Version Rules

- Release cadence: on-demand for v0.1 patch/minor updates.
- Version skipping: supported for forward-only upgrades as long as migration prerequisites are met.
- Unsupported direct jumps must return a preflight failure with required intermediate version guidance.

## Upgrade Sequence

1. Preflight checks:
   - repository reachability
   - package signature verification
   - disk space and service health
2. Create SQLite backup snapshot before package replacement.
3. Stop management plane services (`helling-proxy`, `hellingd`).
4. Install upgraded packages.
5. Start `hellingd` and run startup migrations.
6. Run health checks.
7. Start `helling-proxy`.
8. Mark upgrade successful and emit audit/event records.

## Migration Behavior

- Schema migrations run on daemon start and are forward-only.
- Migration failure is terminal for the upgrade attempt and triggers rollback path.
- Application code must handle compatible older data shape where feasible.

## Rollback

Rollback is package and database coordinated:

1. Reinstall previous package versions.
2. Restore pre-upgrade SQLite backup.
3. Restart management services.
4. Re-run health checks.

Rollback trigger conditions:

- migration failure
- post-upgrade health check failure
- startup crash loop in management plane

## Downtime and Workload Impact

- Target management-plane downtime: under 2 minutes in normal path.
- Running Incus VMs and containers are expected to remain unaffected during management-plane upgrade.

## Health Check Criteria

Upgrade success requires:

- `hellingd` healthy and responsive
- `helling-proxy` healthy and serving dashboard/API
- authentication path functional
- Incus and Podman proxy paths responsive

On failure, upgrade status is marked failed and rollback guidance is returned.
