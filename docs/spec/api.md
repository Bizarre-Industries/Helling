# API Specification

This document is the overview for Helling API surfaces and contract composition.

## Normative Source

The normative machine-readable contract for Helling-owned endpoints is:

- `api/openapi.yaml`

For v0.1, `api/openapi.yaml` is the spec-first source contract. Handler and
client code is generated from that file, and generated artifacts are never
hand-edited.
This document remains normative for envelope shape, error format, pagination contract, and versioning rules.

Markdown specs in this directory are companion contracts and operational rules.

## API Surfaces

1. Helling API: `/api/v1/*` (modeled in `api/openapi.yaml`)
2. Incus proxy: `/api/incus/*` (transparent proxy; native upstream format)
3. Podman proxy: `/api/podman/*` (transparent proxy; native upstream format)

## Endpoint Domains in OpenAPI

- Auth
- Users
- Schedules
- Webhooks
- Kubernetes
- System
- Firewall
- Audit
- Events
- Logs

Deferred domains in v0.1 remain out-of-contract in this OpenAPI file until promoted:

- BMC (target v0.4)
- Notifications (target v0.3)
- Stacks (target v0.3; Podman Compose stacks; see `docs/spec/containers.md`)
- Workspaces (target v0.5; see `docs/design/pages/workspaces.md`)

## Envelope Model

Helling-owned endpoints (`/api/v1/*`) use success/error envelopes defined in OpenAPI components.

- Success envelope: `data` + `meta.request_id`
- Error envelope: `error`, `code`, `action`, `doc_link`, `meta.request_id`

Companion references:

- `docs/spec/errors.md`
- `docs/spec/pagination.md`
- `docs/spec/validation.md`

Proxied endpoints (`/api/incus/*`, `/api/podman/*`) preserve native upstream formats.

## Authorization and Permission Contracts

Authorization behavior is specified in:

- `docs/spec/auth.md`
- `docs/spec/permissions.md`

In v0.1, raw Incus proxy routes are admin-only. Non-admin delegated-user proxying
waits for the ADR-024 per-user certificate transport.

## Event Contracts

Event catalog and delivery semantics are specified in:

- `docs/spec/events.md`

The OpenAPI path `/api/v1/events` defines the SSE transport endpoint while event type semantics live in the event catalog.

## WebSocket Console Note

Console and exec websocket flows under Incus proxy follow native Incus operation-secret protocol and are intentionally not rewritten by Helling.
