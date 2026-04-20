# Testing Standard

Quality and test policy for Helling.

## Coverage Floors

- Go handlers: 80% minimum
- Go services: 90% minimum
- Go overall: 80% minimum
- Frontend components: 60% minimum
- Frontend hooks: 80% minimum

## Test Types

- Unit: required for changed logic
- Integration: required for multi-component behavior
- E2E: required for release-critical paths
- Fuzz/bench: required for critical parsing/hot paths where relevant

## Rules

- Tests must be deterministic.
- Avoid network and external side effects in unit tests.
- Use race detector in CI for Go tests where supported.
- Prefer table-driven tests for multi-case behavior.

## CI Gate

Test failures block merge.
