# ADR-038: SQL-first data access with sqlc + goose over GORM + Atlas provider

> Status: Accepted (2026-04-20)

## Context

Helling v0.1 stores only control-plane state in SQLite with a small schema. The prior plan centered on GORM models with Atlas GORM-provider diffs.

This introduced avoidable complexity for a small, reviewable schema:

- runtime ORM query generation and reflection overhead
- migration coupling to model-introspection workflow
- harder SQL review and explainability

## Decision

Adopt SQL-first persistence:

- Use `goose` for versioned SQL migrations
- Use `sqlc` to generate typed Go query methods from SQL
- Use `database/sql` with SQLite driver directly
- Treat SQL migration/query files as the primary schema and query contract

For v0.1 scope, this replaces GORM runtime ORM and Atlas GORM-provider migration flow.

## Consequences

**Easier:**

- Queries are explicit SQL and reviewable in PRs
- Compile-time checked generated query interfaces via sqlc
- Simpler migration lifecycle with forward-only SQL files
- Lower runtime abstraction overhead

**Harder:**

- More handwritten SQL required up front
- Teams must keep SQL quality high (indexes, explain plans, transaction boundaries)
- Relationship-heavy future domains may require additional conventions
