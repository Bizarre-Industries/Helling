# Alpha Prerequisites

This file tracks minimum prerequisites before v0.1.0-alpha implementation work can be considered ready.

## Required Baseline

- Initial Go module scaffolding for core binaries
- API contract skeleton for Helling-specific endpoints
- Makefile targets for generation, lint, and test flow
- Web UI scaffold with build and dev commands
- CI workflow skeleton with basic validation gates

## Documentation Baseline

- Auth model aligned to ADR-024/030/031/032
- Console model aligned to ADR-010
- Kubernetes model aligned to ADR-005/033
- MicroVM scope explicitly deferred (ADR-006)

## Exit Criteria

- No contradictory references to superseded auth/query-param model
- No contradictory references to SPICE as default console
- No contradictory references to CAPN as v0.1 default path
- No contradictory references to microVM runtime in v0.1 scope
