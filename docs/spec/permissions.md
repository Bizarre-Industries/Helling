# Permission Matrix (v0.1)

Normative authorization contract for fixed roles from ADR-032.

Roles:

- `admin`
- `user`

Legend:

- `YES`: permitted
- `NO`: denied
- `SELF`: only own identity/resource scope
- `SCOPE`: constrained to assigned project/resource boundary

## Helling API (`/api/v1/*`)

| Endpoint Group                         | admin                 | user                         |
| -------------------------------------- | --------------------- | ---------------------------- |
| auth setup/status/login/logout         | YES                   | YES                          |
| auth totp management                   | SELF                  | SELF                         |
| auth token list/create/revoke          | SELF + admin override | SELF                         |
| users list/get                         | YES                   | SELF                         |
| users create/update/delete             | YES                   | NO                           |
| schedules list/get                     | YES                   | NO                           |
| schedules create/update/delete/run     | YES                   | NO                           |
| webhooks list/get                      | YES                   | NO                           |
| webhooks create/update/delete/test     | YES                   | NO                           |
| kubernetes list/get                    | YES                   | NO                           |
| kubernetes create/delete/scale/upgrade | YES                   | NO                           |
| kubernetes kubeconfig                  | YES                   | NO                           |
| system info/hardware/diagnostics       | YES                   | NO                           |
| system config read                     | YES                   | NO                           |
| system config write/upgrade            | YES                   | NO                           |
| firewall host list                     | YES                   | NO                           |
| firewall host create/delete            | YES                   | NO                           |
| audit query/export                     | YES                   | SELF (query own events only) |
| events SSE                             | YES                   | YES (filtered)               |
| health                                 | YES                   | YES                          |

## Incus Proxy (`/api/incus/*`)

v0.1 raw Incus proxy requests are admin-only and forward through the restricted Incus user socket. Caller-specific Incus client certificate identity (ADR-024 + ADR-036) is the post-v0.1 delegation path.

| Method Class                                | admin | user |
| ------------------------------------------- | ----- | ---- |
| Read (`GET`)                                | YES   | NO   |
| Mutation (`POST`, `PUT`, `PATCH`, `DELETE`) | YES   | NO   |

`SCOPE` for non-admin Incus proxy use returns when Incus trust restrictions are enforced by the certificate presented by hellingd.

## Podman Proxy (`/api/podman/*`)

Role gate is enforced in Helling middleware prior to proxying.

| Method Class                                | admin | user |
| ------------------------------------------- | ----- | ---- |
| Read (`GET`)                                | YES   | NO   |
| Mutation (`POST`, `PUT`, `PATCH`, `DELETE`) | YES   | NO   |

## Notes

- Endpoint-specific exceptions must be documented in api/openapi.yaml operation description.
- Authorization failures return `AUTH_INVALID_TOKEN`, `AUTH_INVALID_CREDENTIALS`, or domain-specific forbidden errors from docs/spec/errors.md.
