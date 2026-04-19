# Phase 0 API-CLI-WebUI Parity Matrix (v0.1)

Purpose: track contract parity for Helling-owned API domains.

Gate rule: Phase 0 is not complete unless every implemented `/api/v1/*` operation has both CLI and WebUI support (or an explicit docs-approved exception).

Normative inputs:

- API contract: `api/openapi.yaml`
- CLI contract: `docs/spec/cli.md`
- WebUI contract: `docs/spec/webui-spec.md`

Last reviewed against OpenAPI baseline: commit `201a2c7` (2026-04-20).

## Domain Matrix

| Domain     | API Operations (OpenAPI)                              | CLI Coverage (spec)                                                             | WebUI Coverage (spec)                              | Parity Status                  |
| ---------- | ----------------------------------------------------- | ------------------------------------------------------------------------------- | -------------------------------------------------- | ------------------------------ |
| Auth       | setup, login, refresh, logout, mfa/totp, tokens       | `helling auth ...` listed                                                       | login/session flows + token-aware clients expected | In progress                    |
| Users      | list, create, get, update, delete                     | `helling user ...` listed (note: update/get parity needs exact command mapping) | Users page listed in nav                           | In progress                    |
| Schedules  | list, create, get, update, delete, run                | `helling schedule ...` listed                                                   | Schedules page implied in Helling features set     | In progress                    |
| Webhooks   | list, create, get, update, delete, test               | `helling webhook ...` listed                                                    | Webhooks page included in Helling features set     | In progress                    |
| Kubernetes | list, create, get, delete, scale, upgrade, kubeconfig | `helling k8s ...` listed                                                        | `/kubernetes` and `/kubernetes/:id` specified      | In progress                    |
| System     | info, hardware, config get/put, upgrade, diagnostics  | `helling system ...` listed                                                     | Settings and dashboard system surfaces specified   | In progress                    |
| Firewall   | host list/create/delete                               | `helling firewall ...` listed                                                   | `/firewall` page specified                         | In progress                    |
| Audit      | query, export                                         | no explicit `helling audit ...` commands yet in CLI spec                        | Audit page is part of Helling feature scope        | Gap: CLI                       |
| Events     | SSE stream                                            | no explicit events stream command in CLI spec                                   | task log/SSE updates in layout                     | Gap: CLI                       |
| Health     | health check                                          | no explicit health command in CLI spec                                          | operationally consumed by app shell (implicit)     | Gap: CLI + explicit UI surface |

## Mandatory Gap Closures for Phase 0 Exit

- Add explicit CLI command group for audit query/export parity.
- Add explicit CLI command for event stream (tail/follow) or document approved exception.
- Add explicit CLI command for health check or document approved exception.
- Ensure `users` CLI includes update/get behavior equivalent to API operations (or docs-approved exception).
- Add endpoint-to-command and endpoint-to-page mapping section once implementation paths are finalized.

## Exception Policy

Allowed only when all are true:

- Exception is documented in roadmap and spec.
- Exception includes rationale and target milestone.
- Exception has owner and closure criteria.

## Verification Checklist

- [ ] Every `/api/v1/*` operation mapped to CLI command or approved exception
- [ ] Every `/api/v1/*` operation mapped to WebUI page/action or approved exception
- [ ] Matrix updated when OpenAPI changes
- [ ] Matrix reviewed before Phase 0 completion gate
