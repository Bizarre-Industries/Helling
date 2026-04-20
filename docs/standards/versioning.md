# Versioning Standard

Defines product and API versioning policy.

## Product Versioning

- Product versions follow roadmap milestones from v0.1 toward v1.0.
- Breaking operational behavior requires explicit release notes and migration guidance.

## API Versioning

- Helling-owned API uses URI major versioning (`/api/v1`) per ADR-041.
- Breaking API changes require major version path update.
- Non-breaking changes should remain additive when possible.

## Definition of Done for v1.0

- Contract stability for v1 API.
- Tooling and quality gates fully enforced.
- Core roadmap commitments complete.
