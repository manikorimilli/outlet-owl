# Decision log

One row per technology decision. The ADR holds the full reasoning; this
table is the index. `tech-decision` maintains it.

Settled by the PRD and not recorded as decisions: PostgreSQL in Docker (database), MailHog in Docker (digest email), OpenRouter with Claude Haiku 4.5 (LLM provider, Q-016), nothing else at runtime (PRD section 6).

| Date | Key | Choice | Recommended | Why it was chosen | ADR | Status |
| --- | --- | --- | --- | --- | --- | --- |
| 2026-10-06 | backend language | Go, with a separate TypeScript UI | TypeScript full stack | product owner chose Go; accepts two toolchains and an API contract in exchange for a single server binary | ADR-0001 | Accepted |
| 2026-10-06 | frontend | React + Vite SPA, served by the Go binary | React + Vite SPA | signed-in dashboard, no SEO; static build adds no runtime process | ADR-0002 | Accepted |
| 2026-10-06 | orm and data access | sqlc + pgx + goose | sqlc + pgx + goose | aggregate-heavy SQL checked at build time; matches the Go house rules | ADR-0003 | Accepted |
| 2026-10-06 | ci and delivery | GitHub Actions, no OpenRouter key in CI | GitHub Actions | CI of the existing git host; PostgreSQL as a service container | ADR-0004 | Accepted |
| 2026-10-06 | api style | REST with an OpenAPI spec, TypeScript types generated | REST + OpenAPI | one contract for the Go server and the React UI; offsets the two-language cost of ADR-0001 | ADR-0005 | Accepted |
