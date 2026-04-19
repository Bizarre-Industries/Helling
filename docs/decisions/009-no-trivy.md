# ADR-009: Suspend Aqua Security Tooling

> Status: Accepted

## Context

March 2026: Trivy/Aqua Security supply chain attack. 76 of 77 Docker tags poisoned with credential stealer and self-propagating worm. StepSecurity Harden-Runner detected the attack across 12,000 repos by monitoring outbound C2 connections.

## Decision

Suspend Trivy and other Aqua Security-hosted CI tooling in the Helling pipeline until trust and remediation criteria are explicitly re-evaluated.

Use Grype as the default container vulnerability scanner during the suspension window.

Permanent GitHub Action SHA pinning is tracked independently in ADR-026.

## Consequences

- Grype provides equivalent vulnerability scanning
- CI tooling policy separates temporary vendor suspension (this ADR) from permanent hardening policy (ADR-026)
- Must verify any new CI action is not from Aqua Security
- StepSecurity Harden-Runner recommended for detecting future supply chain attacks
