# ADR-026: SHA-pin all third-party GitHub Actions

> Status: Accepted (2026-04-19)

## Context

Tag-based GitHub Action references can be force-moved upstream. This creates supply-chain risk.

## Decision

Pin every third-party action in workflows to a full commit SHA.

- Allowed: `uses: owner/action@<40-char-sha>`
- Not allowed: `uses: owner/action@vX` for third-party actions

## Consequences

- Reproducible CI behavior
- Reduced exposure to compromised/mutated action tags
- Slightly higher maintenance when upgrading pinned actions
