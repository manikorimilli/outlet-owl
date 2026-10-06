# ADR-0003: Use sqlc, pgx and goose for data access and migrations

- Status: Accepted
- Date: 2026-10-06
- Task: none (stack selection before the first task)
- Deciders: the product owner, in session
- Area: orm and data access
- Reversibility: cheap to awkward: queries are plain SQL files, so moving to another query layer keeps the SQL; migrations are plain SQL too

## Context

- PostgreSQL in Docker is fixed by the PRD (section 6, brief L78).
- The server is Go (ADR-0001).
- Data is small (5 seed outlets, 1,500 seed reviews, REQ-041, REQ-042) and the core reads are aggregates: weekly rating and sentiment trends (REQ-015, REQ-016), the theme heatmap (REQ-017), movers ranked by the change in negative reviews per outlet and theme (REQ-018, Q-004), filtered search (REQ-019, Q-020), and the running model cost total (REQ-031, Q-015).
- The installed `bearing-backend:go` house rules name `pgx`, `sqlc` and `goose`, and `bearing:db-migration` and `bearing:database` follow them.

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| sqlc + pgx + goose (chosen) | SQL is written by hand and a generate step runs after each query change | Aggregate-heavy reads checked against the schema at build time |
| GORM + goose | Hides SQL; the aggregates would be raw SQL anyway; query-per-row risk; outside the house rules | Mostly simple CRUD screens and a team that prefers an ORM |
| Raw pgx repositories | No compile-time check that queries match the schema; more boilerplate | Very few queries |
| sqlx + golang-migrate | The house rules name sqlc, goose and pgx instead | A repository already using them |

## Decision

We will write queries as plain SQL compiled to typed Go with sqlc, run them through pgx, and manage the schema with goose SQL migrations, because the core reads are aggregates that read best as SQL, sqlc checks them against the schema at build time, and this matches the house rules the later skills apply. The product owner chose the recommended option.

## Consequences

- Layout per the house rules: `db/migrations/` (goose, numbered SQL with Up and Down), `db/queries/` (sqlc input), generated code under `internal/store/`, never edited by hand.
- sqlc and goose must be installed before implementation; neither is installed yet.
- Search over review text uses PostgreSQL itself (no other service is allowed); the exact technique (for example `ILIKE` or `pg_trgm`) is set in design.
- Revisit if most new work turns into simple CRUD screens where hand-written SQL slows delivery.

## Commits us to

PostgreSQL, pgx v5, sqlc, goose
