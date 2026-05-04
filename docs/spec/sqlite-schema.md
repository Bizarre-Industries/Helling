# SQLite Schema (v0.1)

> Status: Draft

<!-- markdownlint-disable MD060 -->

This document defines the Helling-owned SQLite schema for v0.1.

## Principles

- SQLite stores Helling control-plane state only.
- Incus and Podman remain source of truth for runtime resources.
- Secrets and private keys are encrypted at rest before persistence.
- Tables that model mutable state include timestamp columns for auditability.
- Schema is SQL-first and migration-driven (ADR-038).

## 1. Identity and Session Tables

### 1.1 users

| Column        | Type                 | Notes                                        |
| ------------- | -------------------- | -------------------------------------------- |
| id            | INTEGER PRIMARY KEY  | Autoincrementing local user id               |
| username      | TEXT UNIQUE NOT NULL | Login identity                               |
| password_hash | TEXT NOT NULL        | argon2id PHC for Helling-managed local users |
| created_at    | INTEGER NOT NULL     | Unix epoch seconds                           |
| is_admin      | INTEGER NOT NULL     | Boolean admin flag, 0 or 1                   |

Checks:

- `is_admin IN (0,1)`

Notes:

- `password_hash` is required for all v0.1 users. The first admin is created
  by `/auth/setup` with the one-time installer setup token.

### 1.2 sessions

| Column     | Type             | Notes                                  |
| ---------- | ---------------- | -------------------------------------- |
| id         | TEXT PRIMARY KEY | UUID                                   |
| user_id    | TEXT NOT NULL    | FK to users.id                         |
| token_hash | TEXT NOT NULL    | SHA-256 digest of opaque session token |
| user_agent | TEXT             | Client fingerprint for audit           |
| ip_address | TEXT             | Last known source IP                   |
| expires_at | INTEGER NOT NULL | Unix epoch seconds                     |
| revoked_at | INTEGER          | Null when active                       |
| created_at | INTEGER NOT NULL | Unix epoch seconds                     |

Indexes:

- `idx_sessions_user_id` on `(user_id)`
- `idx_sessions_expires_at` on `(expires_at)`

### 1.3 api_tokens

| Column       | Type             | Notes                    |
| ------------ | ---------------- | ------------------------ |
| id           | TEXT PRIMARY KEY | UUID                     |
| user_id      | TEXT NOT NULL    | FK to users.id           |
| name         | TEXT NOT NULL    | Human label              |
| token_hash   | TEXT NOT NULL    | SHA-256 digest           |
| scope        | TEXT NOT NULL    | `read`, `write`, `admin` |
| last_used_at | INTEGER          | Unix epoch seconds       |
| expires_at   | INTEGER NOT NULL | Unix epoch seconds       |
| revoked_at   | INTEGER          | Null when active         |
| created_at   | INTEGER NOT NULL | Unix epoch seconds       |

Checks:

- `scope IN ('read','write','admin')`

Indexes:

- `idx_api_tokens_user_id` on `(user_id)`
- `idx_api_tokens_expires_at` on `(expires_at)`

## 2. MFA and Auth Material

### 2.1 totp_secrets

| Column           | Type             | Notes                  |
| ---------------- | ---------------- | ---------------------- |
| user_id          | TEXT PRIMARY KEY | FK to users.id         |
| encrypted_secret | BLOB NOT NULL    | Encrypted with app key |
| enabled          | INTEGER NOT NULL | 0 or 1                 |
| created_at       | INTEGER NOT NULL | Unix epoch seconds     |
| updated_at       | INTEGER NOT NULL | Unix epoch seconds     |

Checks:

- `enabled IN (0,1)`

### 2.2 recovery_codes

| Column     | Type             | Notes              |
| ---------- | ---------------- | ------------------ |
| id         | TEXT PRIMARY KEY | UUID               |
| user_id    | TEXT NOT NULL    | FK to users.id     |
| code_hash  | TEXT NOT NULL    | Argon2id hash      |
| used_at    | INTEGER          | Null when unused   |
| created_at | INTEGER NOT NULL | Unix epoch seconds |

Indexes:

- `idx_recovery_codes_user_id` on `(user_id)`

### 2.3 auth_events

| Column        | Type             | Notes                                                          |
| ------------- | ---------------- | -------------------------------------------------------------- |
| id            | TEXT PRIMARY KEY | UUID                                                           |
| user_id       | TEXT             | Nullable for failed pre-auth events                            |
| event_type    | TEXT NOT NULL    | `login_ok`, `login_fail`, `token_revoked`, `cert_rotated`, etc |
| source_ip     | TEXT             | Request source                                                 |
| user_agent    | TEXT             | Request user agent                                             |
| metadata_json | TEXT             | Compact structured context                                     |
| created_at    | INTEGER NOT NULL | Unix epoch seconds                                             |

Indexes:

- `idx_auth_events_user_id` on `(user_id)`
- `idx_auth_events_created_at` on `(created_at)`

## 3. Incus Trust and CA Tables

### 3.1 helling_ca

| Column               | Type             | Notes                                  |
| -------------------- | ---------------- | -------------------------------------- |
| id                   | TEXT PRIMARY KEY | Singleton row key, value `default`     |
| cert_pem             | TEXT NOT NULL    | CA certificate PEM                     |
| encrypted_key_pem    | BLOB NOT NULL    | Encrypted CA private key PEM           |
| not_before           | INTEGER NOT NULL | Unix epoch seconds                     |
| not_after            | INTEGER NOT NULL | Unix epoch seconds                     |
| rotation_grace_until | INTEGER          | Null unless rolling CA rotation active |
| created_at           | INTEGER NOT NULL | Unix epoch seconds                     |
| updated_at           | INTEGER NOT NULL | Unix epoch seconds                     |

Checks:

- `id = 'default'`

### 3.2 incus_user_certs

| Column            | Type                 | Notes                        |
| ----------------- | -------------------- | ---------------------------- |
| user_id           | TEXT PRIMARY KEY     | FK to users.id               |
| cert_pem          | TEXT NOT NULL        | Public certificate           |
| encrypted_key_pem | BLOB NOT NULL        | Encrypted private key        |
| fingerprint       | TEXT UNIQUE NOT NULL | Incus trust fingerprint      |
| restricted        | INTEGER NOT NULL     | Must be 1 for user certs     |
| project_scope     | TEXT NOT NULL        | Assigned project/limit scope |
| expires_at        | INTEGER NOT NULL     | Unix epoch seconds           |
| revoked_at        | INTEGER              | Null when active             |
| created_at        | INTEGER NOT NULL     | Unix epoch seconds           |
| updated_at        | INTEGER NOT NULL     | Unix epoch seconds           |

Checks:

- `restricted IN (0,1)`

Indexes:

- `idx_incus_user_certs_fingerprint` on `(fingerprint)`
- `idx_incus_user_certs_expires_at` on `(expires_at)`

## 4. Webhooks and Delivery State

### 4.1 webhooks

| Column           | Type             | Notes                            |
| ---------------- | ---------------- | -------------------------------- |
| id               | TEXT PRIMARY KEY | UUID                             |
| name             | TEXT NOT NULL    | Human label                      |
| url              | TEXT NOT NULL    | HTTPS endpoint                   |
| events_json      | TEXT NOT NULL    | JSON array of event type filters |
| secret_encrypted | BLOB NOT NULL    | Encrypted HMAC secret            |
| enabled          | INTEGER NOT NULL | 0 or 1                           |
| last_delivery_at | INTEGER          | Last attempt timestamp           |
| created_by       | TEXT NOT NULL    | FK to users.id                   |
| created_at       | INTEGER NOT NULL | Unix epoch seconds               |
| updated_at       | INTEGER NOT NULL | Unix epoch seconds               |

Checks:

- `enabled IN (0,1)`

Indexes:

- `idx_webhooks_created_by` on `(created_by)`
- `idx_webhooks_enabled` on `(enabled)`

### 4.2 webhook_deliveries

| Column          | Type             | Notes                                                 |
| --------------- | ---------------- | ----------------------------------------------------- |
| id              | TEXT PRIMARY KEY | UUID                                                  |
| webhook_id      | TEXT NOT NULL    | FK to webhooks.id                                     |
| event_id        | TEXT NOT NULL    | Event envelope id                                     |
| event_type      | TEXT NOT NULL    | Event type                                            |
| attempt         | INTEGER NOT NULL | Attempt number (1..N)                                 |
| status          | TEXT NOT NULL    | `pending`, `success`, `failed`                        |
| http_status     | INTEGER          | Response status code if available                     |
| latency_ms      | INTEGER          | End-to-end latency for this attempt                   |
| next_retry_at   | INTEGER          | Null when no retry remains                            |
| error_text      | TEXT             | Failure summary (truncated)                           |
| response_sample | TEXT             | Response excerpt (truncated)                          |
| created_at      | INTEGER NOT NULL | Unix epoch seconds                                    |
| delivered_at    | INTEGER          | Set when terminal success or terminal failure reached |

Checks:

- `attempt >= 1`
- `status IN ('pending','success','failed')`

Indexes:

- `idx_webhook_deliveries_webhook_id_created_at` on `(webhook_id, created_at DESC)`
- `idx_webhook_deliveries_event_id` on `(event_id)`

Retention rule:

- Keep the latest 100 delivery rows per webhook (as specified in platform contract).

## 5. Kubernetes Control-Plane Metadata

### 5.1 kubernetes_clusters

| Column               | Type             | Notes                                                 |
| -------------------- | ---------------- | ----------------------------------------------------- |
| name                 | TEXT PRIMARY KEY | Cluster name                                          |
| state                | TEXT NOT NULL    | `creating`, `ready`, `upgrading`, `deleting`, `error` |
| k8s_version          | TEXT NOT NULL    | Kubernetes version                                    |
| pod_cidr             | TEXT             | Pod CIDR                                              |
| service_cidr         | TEXT             | Service CIDR                                          |
| control_plane_count  | INTEGER NOT NULL | Control-plane node count                              |
| worker_count         | INTEGER NOT NULL | Worker node count                                     |
| kubeconfig_encrypted | BLOB NOT NULL    | Encrypted kubeconfig payload                          |
| last_operation       | TEXT             | Last operation key                                    |
| last_error           | TEXT             | Last terminal or transient error                      |
| created_by           | TEXT NOT NULL    | FK to users.id                                        |
| created_at           | INTEGER NOT NULL | Unix epoch seconds                                    |
| updated_at           | INTEGER NOT NULL | Unix epoch seconds                                    |
| deleted_at           | INTEGER          | Soft-delete marker for async teardown flows           |

Checks:

- `state IN ('creating','ready','upgrading','deleting','error')`
- `control_plane_count >= 1`
- `worker_count >= 0`

Indexes:

- `idx_kubernetes_clusters_state` on `(state)`
- `idx_kubernetes_clusters_created_by` on `(created_by)`

### 5.2 kubernetes_nodes

| Column       | Type             | Notes                                          |
| ------------ | ---------------- | ---------------------------------------------- |
| id           | TEXT PRIMARY KEY | UUID                                           |
| cluster_name | TEXT NOT NULL    | FK to kubernetes_clusters.name                 |
| name         | TEXT NOT NULL    | Node name                                      |
| role         | TEXT NOT NULL    | `control-plane`, `worker`                      |
| status       | TEXT NOT NULL    | `provisioning`, `ready`, `notready`, `deleted` |
| instance_ref | TEXT             | Back-reference to Incus instance name          |
| ip_address   | TEXT             | Last known node IP                             |
| created_at   | INTEGER NOT NULL | Unix epoch seconds                             |
| updated_at   | INTEGER NOT NULL | Unix epoch seconds                             |

Checks:

- `role IN ('control-plane','worker')`
- `status IN ('provisioning','ready','notready','deleted')`

Indexes:

- `idx_kubernetes_nodes_cluster_name` on `(cluster_name)`
- `uq_kubernetes_nodes_cluster_name_name` unique on `(cluster_name, name)`

## 6. Firewall and Warning State

### 6.1 firewall_host_rules

| Column     | Type             | Notes                                      |
| ---------- | ---------------- | ------------------------------------------ |
| id         | TEXT PRIMARY KEY | UUID                                       |
| chain      | TEXT NOT NULL    | nft chain (for example `input`, `forward`) |
| position   | INTEGER NOT NULL | Stable order within chain                  |
| action     | TEXT NOT NULL    | `accept`, `drop`, `reject`                 |
| protocol   | TEXT NOT NULL    | `tcp`, `udp`, `icmp`, `any`                |
| src_cidr   | TEXT             | Source CIDR filter                         |
| dst_cidr   | TEXT             | Destination CIDR filter                    |
| src_port   | TEXT             | Source port/range                          |
| dst_port   | TEXT             | Destination port/range                     |
| comment    | TEXT             | Operator comment                           |
| enabled    | INTEGER NOT NULL | 0 or 1                                     |
| created_by | TEXT NOT NULL    | FK to users.id                             |
| created_at | INTEGER NOT NULL | Unix epoch seconds                         |
| updated_at | INTEGER NOT NULL | Unix epoch seconds                         |

Checks:

- `enabled IN (0,1)`
- `action IN ('accept','drop','reject')`
- `protocol IN ('tcp','udp','icmp','any')`

Indexes:

- `idx_firewall_host_rules_chain_position` on `(chain, position)`
- `idx_firewall_host_rules_enabled` on `(enabled)`

### 6.2 warnings

| Column          | Type             | Notes                                                      |
| --------------- | ---------------- | ---------------------------------------------------------- |
| id              | TEXT PRIMARY KEY | UUID                                                       |
| warning_key     | TEXT NOT NULL    | Deterministic dedupe key (category + subject)              |
| category        | TEXT NOT NULL    | `storage`, `smart`, `tls`, `instance`, `backup`, `cluster` |
| severity        | TEXT NOT NULL    | `info`, `warning`, `critical`                              |
| state           | TEXT NOT NULL    | `active`, `acknowledged`, `resolved`                       |
| subject         | TEXT             | Instance, pool, cert, or node reference                    |
| message         | TEXT NOT NULL    | User-facing summary                                        |
| details_json    | TEXT             | Structured warning context                                 |
| first_seen_at   | INTEGER NOT NULL | Unix epoch seconds                                         |
| last_seen_at    | INTEGER NOT NULL | Unix epoch seconds                                         |
| acknowledged_by | TEXT             | Nullable FK to users.id                                    |
| acknowledged_at | INTEGER          | Null unless acknowledged                                   |
| resolved_at     | INTEGER          | Null unless resolved                                       |

Checks:

- `severity IN ('info','warning','critical')`
- `state IN ('active','acknowledged','resolved')`

Indexes:

- `uq_warnings_warning_key` unique on `(warning_key)`
- `idx_warnings_state_severity` on `(state, severity)`

## 7. Foreign Key Policy

- `PRAGMA foreign_keys = ON` is mandatory for all connections.
- Parent-delete behavior:
  - `users` -> child auth/session/token tables: `ON DELETE CASCADE`
  - `users` -> warning acknowledgements and creator references: `ON DELETE SET NULL` where history must remain
  - `webhooks` -> `webhook_deliveries`: `ON DELETE CASCADE`
  - `kubernetes_clusters` -> `kubernetes_nodes`: `ON DELETE CASCADE`

## 8. SQLite PRAGMA Baseline

Runtime baseline for `hellingd` SQLite connections:

- `PRAGMA journal_mode = WAL`
- `PRAGMA synchronous = NORMAL`
- `PRAGMA foreign_keys = ON`
- `PRAGMA busy_timeout = 5000`
- `PRAGMA temp_store = MEMORY`

`PRAGMA optimize` may be executed during controlled maintenance windows.

## 9. Migration and Query Conventions (sqlc + goose)

- Migrations are forward-only SQL files managed by `goose`.
- Naming format: `<unix_ts>_<short_description>.sql`.
- Every migration includes both `-- +goose Up` and `-- +goose Down` sections; down sections are best-effort for local/dev rollback and are not relied on for production rollback.
- Destructive migrations require backup creation before apply.
- New columns must be backward-compatible for at least one release window.

sqlc conventions:

- Query SQL is the source of truth for generated data access methods.
- Mutations that touch multiple rows/entities use explicit SQL transactions in service layer.
- Secret-bearing columns (`*_encrypted`, key material, token hashes) are never selected by list/read queries used for API responses.

<!-- markdownlint-enable MD060 -->
