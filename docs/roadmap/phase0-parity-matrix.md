# Phase 0 API-CLI-WebUI Parity Matrix (v0.1)

Purpose: track contract parity for Helling-owned API domains.

Gate rule: Phase 0 is not complete unless every implemented `/api/v1/*` operation has both CLI and WebUI support (or an explicit docs-approved exception).

Normative inputs:

- API contract: `api/openapi.yaml`
- CLI contract: `docs/spec/cli.md`
- WebUI contract: `docs/spec/webui-spec.md`

Last reviewed against OpenAPI baseline: commit `201a2c7` (2026-04-20). Matrix refreshed 2026-04-20 after CLI-parity restoration (commit `36b1bff`) added `helling user get/update`, `helling audit query/export`, `helling events tail`, `helling system health`, and `helling system upgrade --rollback`.

## Domain Matrix

| Domain     | API Operations (OpenAPI)                              | CLI Coverage (spec)                                              | WebUI Coverage (spec)                              | Parity Status |
| ---------- | ----------------------------------------------------- | ---------------------------------------------------------------- | -------------------------------------------------- | ------------- |
| Auth       | setup, login, refresh, logout, mfa/totp, tokens       | `helling auth ...` listed                                        | login/session flows + token-aware clients expected | ✅ Covered    |
| Users      | list, create, get, update, delete                     | `helling user list/create/get/update/delete`                     | `/users` page with ProTable + fixed-role read-only | ✅ Covered    |
| Schedules  | list, create, get, update, delete, run                | `helling schedule ...` listed                                    | `/schedules` page specified                        | ✅ Covered    |
| Webhooks   | list, create, get, update, delete, test               | `helling webhook ...` listed                                     | Webhooks page in Helling feature set               | ✅ Covered    |
| Kubernetes | list, create, get, delete, scale, upgrade, kubeconfig | `helling k8s ...` listed                                         | `/kubernetes` and `/kubernetes/:id` specified      | ✅ Covered    |
| System     | info, hardware, config get/put, upgrade, diagnostics  | `helling system ...` including `health` and `upgrade --rollback` | Settings + dashboard system surfaces specified     | ✅ Covered    |
| Firewall   | host list/create/delete                               | `helling firewall ...` listed                                    | `/firewall` page specified                         | ✅ Covered    |
| Audit      | query, export                                         | `helling audit query`, `helling audit export`                    | `/audit` page with filters + CSV export            | ✅ Covered    |
| Events     | SSE stream                                            | `helling events tail` (SSE follow)                               | task log / SSE consumed by app shell layout        | ✅ Covered    |
| Health     | health check                                          | `helling system health`                                          | consumed internally by app shell for status banner | ✅ Covered    |
| Logs       | query (journal-backed)                                | `helling logs ...` listed                                        | `/logs` page specified                             | ✅ Covered    |

## Status Notes

All domains have spec-level parity as of 2026-04-20. The matrix currently shows **zero** Phase 0 parity gaps at the spec level. BMC and Notifications domains are tracked separately in `phase0-parity-exceptions.yaml` with target versions v0.4.0 and v0.3.0 respectively — those are explicit scope deferrals, not gaps.

## Exception Policy

Allowed only when all are true:

- Exception is documented in `phase0-parity-exceptions.yaml` with operation_id, `missing` surface, rationale, and target version.
- Exception has an owner and closure criteria.
- Exception target version is a real planned release (v0.3.0, v0.4.0, v0.5.0, v0.6.0).

## Verification Checklist

- [ ] Every `/api/v1/*` operation mapped to CLI command or approved exception
- [ ] Every `/api/v1/*` operation mapped to WebUI page/action or approved exception
- [ ] Matrix updated when OpenAPI changes
- [ ] Matrix reviewed before Phase 0 completion gate
