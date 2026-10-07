# Migrations

goose SQL migrations (ADR-0003), numbered and append-only once applied. The
plan is in docs/design/data-model.md section 8; the first file is
`00001_create_outlets_and_users.sql`. The budget schema
(`budget.model_calls`) has its own set in `budget/`, with its own version
table `goose_budget_version` and no Down (HLD section 4): `make migrate`
applies it after the domain set and `make migrate-down` never touches it,
so no back-out drops the running total.
