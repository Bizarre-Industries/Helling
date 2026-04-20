# Audit Specification

Normative contract for Helling's audit trail. All audit records are emitted to the systemd journal (ADR-019) as structured entries.

---

## 1. Field schema

The authoritative field list is in `docs/decisions/019-journal-over-sqlite-audit.md` §"Structured field schema (normative)". This spec references that schema — do not duplicate it. Every field listed there is indexed by journald and queryable via `journalctl HELLING_FIELD=value`.

Summary (see ADR-019 for types and required-ness):

- Identity: `HELLING_ACTOR`, `HELLING_ACTOR_ID`, `HELLING_ROLE`
- Action: `HELLING_ACTION`, `HELLING_OUTCOME`, `HELLING_POLICY_REASON`
- Request correlation: `HELLING_REQUEST_ID`, `HELLING_METHOD`, `HELLING_PATH`, `HELLING_STATUS`, `HELLING_DURATION_MS`
- Target: `HELLING_TARGET_TYPE`, `HELLING_TARGET_ID`, `HELLING_BEFORE`, `HELLING_AFTER`
- Transport: `HELLING_SOURCE_IP`, `HELLING_USER_AGENT`, `HELLING_JWT_ID`
- Journal defaults: `MESSAGE`, `PRIORITY`, `SYSLOG_IDENTIFIER=hellingd`

`HELLING_BEFORE`/`HELLING_AFTER` payloads are truncated at 4 KB each with a `"truncated":true` marker; if larger context is needed, emit a separate detail event.

## 2. Emission contract

All API mutations MUST emit an audit record. Auth events (login, logout, MFA, token issue/revoke) MUST emit an audit record. Policy-deny events MUST emit an audit record with `HELLING_OUTCOME=denied` and `HELLING_POLICY_REASON`.

Emission path uses `github.com/coreos/go-systemd/v22/journal` (ADR-018 narrow exception). On emit failure, the handler falls back to stderr (captured by systemd via `StandardError=journal`) and logs a non-fatal warning. The request MUST NOT fail on audit emission failure.

## 3. Query API

Read access is exposed via `/api/v1/audit/*` operations. Query implementation is a wrapper around `journalctl --output=json` executed as the `hellingd` user (ADR-050). The wrapper is responsible for:

- Translating Helling filter DTOs (`actor`, `action`, `outcome`, time range) into `journalctl` flags and `HELLING_FIELD=value` predicates
- Parsing the JSON-per-line output into Helling response envelopes
- Enforcing the pagination contract in `docs/spec/pagination.md`
- Enforcing the field redaction rules below

Operator access is via standard journal tooling (`journalctl -t hellingd`, `journalctl HELLING_ACTOR=suhail`) — no Helling-specific CLI is required to read audit from the host.

## 4. Authorization

The audit read API is restricted to the `admin` and `auditor` roles (ADR-032). The `user` role receives HTTP 403 on any `/api/v1/audit/*` read.

Audit writes are not user-facing — they are emitted by hellingd as a side-effect of authorized mutation operations.

## 5. Redaction

The following fields are redacted before audit emission:

- Passwords, API tokens, private keys, JWT secrets — never logged, never stored in `HELLING_BEFORE`/`HELLING_AFTER`
- TOTP codes — never logged
- Raw session cookies — never logged
- `HELLING_USER_AGENT` is truncated to 512 bytes (ADR-019)

Redaction is the emission-site's responsibility; the audit reader trusts what the journal contains.

## 6. Retention

Journal retention is controlled by `journald.conf` drop-in configuration shipped as part of the Helling OS image. Default retention: `SystemMaxUse=1G`, `SystemKeepFree=2G`, `MaxRetentionSec=90day`. Operators can adjust via the standard `journald.conf.d/` mechanism; Helling does not manage retention separately.

For long-term archival beyond 90 days, the recommended workflow is `helling audit export --format jsonl --since <time>` piped to an external log store or S3 bucket.

## 7. Export

`/api/v1/audit/export` produces JSON Lines output with one journal entry per line. Each line is the parsed JSON form of a single journal record (all `HELLING_*` fields plus `MESSAGE`, `PRIORITY`, `__REALTIME_TIMESTAMP`). Supports the same filters as `/api/v1/audit/query`.

The export endpoint streams — it MUST NOT buffer the full result set in memory. Clients receive `Content-Type: application/x-ndjson`.

## 8. Cross-references

- `docs/decisions/019-journal-over-sqlite-audit.md` — normative field schema + emission pattern
- `docs/decisions/018-shell-out-over-libraries.md` — journal emission exception rationale
- `docs/decisions/050-hellingd-non-root-user.md` — polkit / journal socket access under non-root
- `docs/spec/observability.md` — structured application logging (distinct from audit)
- `docs/design/pages/audit.md` — WebUI surface (table, before/after diff column)
- `docs/spec/pagination.md` — pagination contract
- `docs/standards/security.md` §4 — CVE response workflow (separate from audit emission)
