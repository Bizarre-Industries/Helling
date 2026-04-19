# Backup Format Specification

<!-- markdownlint-disable MD029 MD032 -->

Backup and restore format contract for Helling v0.1.

## Scope

- Helling control-plane state backups (SQLite + metadata)
- Workload backups delegated to Incus export/snapshot mechanisms

## Backup Types

1. Control-plane backup

- Includes SQLite state and required metadata to restore Helling control plane.

2. Workload backup

- Uses Incus-native backup/export semantics for instances/volumes.
- Helling stores schedule and verification metadata, not a custom VM disk format.

## Control-Plane Backup Layout

Recommended archive layout:

- `metadata.json` (required)
- `helling.db` (required)
- `checksums.txt` (required)

`metadata.json` minimum fields:

- `format_version`
- `created_at`
- `helling_version`
- `schema_version`
- `encryption` (none|age)
- `source_host`

## Encryption

- Optional encryption uses age.
- If encrypted, metadata must indicate age mode and recipient metadata.
- Private identity material is never embedded unencrypted in archives.

## Integrity

- Backups MUST include content checksums.
- Restore MUST fail fast on checksum mismatch.

## Restore Rules

- Restore requires compatible application/schema version or documented migration path.
- Control-plane restore sequence:
  1. stop management services
  2. verify backup integrity
  3. restore DB
  4. start services
  5. run health checks

## Compatibility Contract

- Backup format uses explicit `format_version`.
- Breaking format changes require version bump and migration tooling.
- Forward compatibility is not guaranteed without migration support.

## Verification

- Scheduled restore verification should periodically validate recoverability.
- Verification outcomes must emit warning/event state when failing.
