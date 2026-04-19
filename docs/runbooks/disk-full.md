# Runbook: Disk Full / High Utilization

## Trigger

- Warning/critical threshold breach for storage utilization.
- Writes or backups failing due to insufficient free space.

## Immediate Actions

1. Confirm affected filesystem/pool.
2. Pause non-essential write-heavy operations.
3. Check recent growth sources (backups, logs, images, snapshots).

## Mitigation Steps

- Prune expired backups/snapshots per policy.
- Remove stale images/artifacts.
- Rotate/compress logs where applicable.
- Expand storage pool/capacity if available.

## Validation

- Utilization drops below critical threshold.
- Backup/schedule operations recover.
- Health and warning surfaces update accordingly.

## Follow-Up

- Tune retention and warning thresholds if needed.
- Add capacity planning action item.
