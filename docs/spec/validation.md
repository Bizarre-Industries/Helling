# Input Validation Contract

Normative validation rules at API boundaries for `/api/v1/*`.

Validation failure response uses `VALIDATION_FAILED` envelope with field details in `meta.validation`.

## Common Field Rules

### Name (generic resource names)

- Pattern: `^[a-z][a-z0-9-]*[a-z0-9]$`
- Length: 1..63

### Identifier

- ULID-like ids: `^[0-9A-HJKMNP-TV-Z]{26}$` where applicable
- Path params must be non-empty UTF-8 strings even when id format is not ULID

### Username

- Pattern: `^[a-z_][a-z0-9_-]{0,31}$`
- Length: 1..32

### Password

- Length: 8..1024
- No composition requirements beyond length at the Helling layer

### URL

- Webhook URL must be absolute URI
- Allowed scheme: `https`
- `http` is rejected unless explicitly enabled by future config gate

### Duration

- Go-style duration strings
- Min 1s, max 720h

### Cron

- Five-field cron expression
- Timezone baseline is UTC

### Role/status enums

- Role label: `admin | user`
- User status: `active | disabled`
- Token scope: `read | write | admin`

## Endpoint-Specific Examples

- `POST /api/v1/users`: username + password required; `is_admin` defaults false.
- `POST /api/v1/schedules`: type, target, cron required and validated.
- `POST /api/v1/webhooks`: name, url, events required; each event must exist in docs/spec/events.md catalog.

## Error Response Shape

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_FAILED",
  "action": "Correct invalid fields and retry",
  "doc_link": "https://bizarre.industries/docs/errors/VALIDATION_FAILED",
  "meta": {
    "request_id": "req_01JZ...",
    "validation": [
      {
        "field": "name",
        "code": "VALIDATION_PATTERN_MISMATCH",
        "message": "must match name pattern"
      }
    ]
  }
}
```

## Contract Notes

- Validation runs before authorization-sensitive side effects.
- Unknown fields in strict payloads may be rejected to prevent silent misconfiguration.
- Endpoint contracts in api/openapi.yaml are authoritative for required fields and basic type checks.
