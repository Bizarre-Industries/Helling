# Error Codes

Normative machine-readable error code catalog for Helling-owned endpoints (`/api/v1/*`).

Format: `<DOMAIN>_<DETAIL>`

Domains: AUTH, USER, SCHEDULE, WEBHOOK, K8S, SYSTEM, FIREWALL, AUDIT, VALIDATION, RATE_LIMIT, UPSTREAM.

All error responses use the envelope contract documented in docs/spec/api.md and api/openapi.yaml.

## AUTH

| Code                         | HTTP | When                                           | Action                      |
| ---------------------------- | ---- | ---------------------------------------------- | --------------------------- |
| AUTH_INVALID_CREDENTIALS     | 401  | Local password authentication failed           | Check username and password |
| AUTH_INVALID_TOKEN           | 401  | Access token invalid or expired                | Re-authenticate             |
| AUTH_MFA_REQUIRED            | 202  | Login requires MFA completion                  | Call `/auth/mfa/complete`   |
| AUTH_TOTP_INVALID            | 401  | TOTP code invalid                              | Verify authenticator time   |
| AUTH_TOTP_LOCKED             | 423  | Too many MFA failures                          | Use recovery code           |
| AUTH_SETUP_ALREADY_COMPLETED | 409  | Setup endpoint called after first admin exists | Use login flow              |

## USER

| Code                 | HTTP | When                         | Action                   |
| -------------------- | ---- | ---------------------------- | ------------------------ |
| USER_NOT_FOUND       | 404  | User id missing              | Verify target id         |
| USER_USERNAME_EXISTS | 409  | Duplicate username on create | Choose a unique username |
| USER_ROLE_INVALID    | 400  | Role outside allowed enum    | Use admin or user        |
| USER_STATUS_INVALID  | 400  | Status outside allowed enum  | Use active or disabled   |

## SCHEDULE

| Code                    | HTTP | When                        | Action                              |
| ----------------------- | ---- | --------------------------- | ----------------------------------- |
| SCHEDULE_NOT_FOUND      | 404  | Schedule id missing         | Verify schedule id                  |
| SCHEDULE_CRON_INVALID   | 400  | Cron syntax rejected        | Fix cron expression                 |
| SCHEDULE_TARGET_INVALID | 400  | Unsupported target/resource | Verify target in UI/CLI             |
| SCHEDULE_TRIGGER_FAILED | 502  | Manual trigger failed       | Check systemd unit + upstream state |

## WEBHOOK

| Code                    | HTTP | When                         | Action                                   |
| ----------------------- | ---- | ---------------------------- | ---------------------------------------- |
| WEBHOOK_NOT_FOUND       | 404  | Webhook id missing           | Verify webhook id                        |
| WEBHOOK_URL_INVALID     | 400  | URL fails validation policy  | Use allowed HTTPS URL                    |
| WEBHOOK_EVENT_INVALID   | 400  | Unknown event type requested | Use catalog from docs/spec/events.md     |
| WEBHOOK_DELIVERY_FAILED | 502  | Delivery attempts exhausted  | Check endpoint health/signature handling |

## K8S

| Code                  | HTTP | When                           | Action                         |
| --------------------- | ---- | ------------------------------ | ------------------------------ |
| K8S_CLUSTER_NOT_FOUND | 404  | Cluster name missing           | Verify cluster name            |
| K8S_CREATE_INVALID    | 400  | Cluster create request invalid | Correct create payload         |
| K8S_OPERATION_FAILED  | 502  | Provision/scale/upgrade failed | Check Incus and bootstrap logs |

## SYSTEM

| Code                      | HTTP | When                         | Action                            |
| ------------------------- | ---- | ---------------------------- | --------------------------------- |
| SYSTEM_CONFIG_INVALID     | 400  | Config payload rejected      | Fix invalid keys/values           |
| SYSTEM_UPGRADE_FAILED     | 502  | Upgrade command failed       | Check package sources and journal |
| SYSTEM_DIAGNOSTICS_FAILED | 500  | Diagnostics internal failure | Retry and inspect logs            |

## FIREWALL

| Code                    | HTTP | When              | Action                        |
| ----------------------- | ---- | ----------------- | ----------------------------- |
| FIREWALL_RULE_NOT_FOUND | 404  | Rule id missing   | Verify rule id                |
| FIREWALL_RULE_INVALID   | 400  | Rule spec invalid | Correct protocol/ports/action |
| FIREWALL_APPLY_FAILED   | 502  | nft apply failed  | Check nft ruleset state       |

## AUDIT

| Code                        | HTTP | When                       | Action                |
| --------------------------- | ---- | -------------------------- | --------------------- |
| AUDIT_QUERY_INVALID         | 400  | Invalid audit query params | Correct filter values |
| AUDIT_EXPORT_INVALID_FORMAT | 400  | Unsupported export format  | Use csv or json       |

## VALIDATION

| Code                              | HTTP | When                          | Action                              |
| --------------------------------- | ---- | ----------------------------- | ----------------------------------- |
| VALIDATION_FAILED                 | 400  | One or more fields invalid    | Correct fields in `meta.validation` |
| VALIDATION_REQUIRED_FIELD_MISSING | 400  | Required field absent         | Provide required field              |
| VALIDATION_OUT_OF_RANGE           | 400  | Numeric value outside bounds  | Use allowed range                   |
| VALIDATION_PATTERN_MISMATCH       | 400  | String violates regex/pattern | Use allowed format                  |

## RATE LIMIT

| Code                | HTTP | When                   | Action                         |
| ------------------- | ---- | ---------------------- | ------------------------------ |
| RATE_LIMIT_EXCEEDED | 429  | Request limit exceeded | Retry after `Retry-After`      |
| AUTH_RATE_LIMITED   | 429  | Auth attempts exceeded | Wait for lock window to expire |

## UPSTREAM

| Code                        | HTTP | When                                 | Action                              |
| --------------------------- | ---- | ------------------------------------ | ----------------------------------- |
| INCUS_UPSTREAM_UNAVAILABLE  | 502  | Incus loopback transport unavailable | Verify Incus listener + trust/certs |
| PODMAN_UPSTREAM_UNAVAILABLE | 502  | Podman socket unavailable            | Verify podman socket activation     |
| UPSTREAM_TIMEOUT            | 504  | Upstream timeout                     | Retry and inspect service latency   |

## Doc Link Convention

`doc_link` value in error envelopes must point to:

`https://bizarre.industries/docs/errors/<CODE>`

Example: `https://bizarre.industries/docs/errors/AUTH_INVALID_TOKEN`
