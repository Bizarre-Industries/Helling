# ADR-009: No Trivy — Use Grype

> Status: Accepted

## Context

March 2026: Trivy/Aqua Security supply chain attack. 76 of 77 Docker tags poisoned with credential stealer and self-propagating worm. StepSecurity Harden-Runner detected the attack across 12,000 repos by monitoring outbound C2 connections.

## Decision

Never use Trivy or any Aqua Security GitHub Actions. Use Grype for container vulnerability scanning. Pin all GitHub Actions to full SHA hashes (not version tags — tags can be force-pushed, which is exactly how the Trivy attack worked).

## Consequences

- Grype provides equivalent vulnerability scanning
- All CI actions pinned to SHA hashes (immune to tag-based attacks)
- Must verify any new CI action is not from Aqua Security
- StepSecurity Harden-Runner recommended for detecting future supply chain attacks
