# ADR-040: net/http ServeMux over chi for v0.1 routing baseline

> Status: Accepted (2026-04-20)

## Context

Helling routing needs are method + path matching with standard middleware chaining and minimal dependency footprint.

Go's standard library ServeMux supports method-aware patterns and path values, covering the required v0.1 handler surface.

## Decision

Use standard library `net/http` ServeMux as the default router baseline for Helling v0.1.

- Prefer stdlib route patterns and `r.PathValue(...)`
- Keep middleware as standard `func(http.Handler) http.Handler`
- Avoid external router dependency unless future requirements exceed stdlib capabilities

## Consequences

**Easier:**

- One less runtime dependency to maintain
- Keeps routing behavior within stdlib primitives
- Aligns with minimal dependency objective

**Harder:**

- Some ergonomic niceties from third-party routers are unavailable
- Complex route grouping patterns may require explicit composition
