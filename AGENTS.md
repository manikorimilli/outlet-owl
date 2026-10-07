# AGENTS.md

The standard every AI coding agent follows in this repository. It loads into
every session, so it holds only what applies on every turn; the documents
under `docs/` carry the detail.

## What this repository is

OutletOwl, review intelligence for one multi-outlet brand. One repository
holds the documents and both apps (docs/architecture/repo-plan.json):

| Part | Where | Stack |
| --- | --- | --- |
| Server | root: `cmd/api`, `internal/`, `db/` | Go 1.26.8, net/http, pgx, sqlc, goose, slog (ADR-0001, ADR-0003) |
| Web app | `web/` | React 19, Vite 7, TypeScript; Vitest; built to static files the server serves (ADR-0002) |
| API contract | `api/openapi.yaml` | REST, OpenAPI 3.1; the UI's types are generated from it (ADR-0005) |
| Runtime services | `docker-compose.yml` | PostgreSQL 16 and MailHog, nothing else (PRD section 6) |
| Planning | `docs/` | PRD, backlog, ADRs, HLD, data model, screens, GenAI design |

The skeleton serves only `GET /api/v1/health`. Product features arrive story
by story from `docs/product/backlog.md`.

## Ground rules

Breaking one gets the change rejected, whatever else it does.

1. Never commit to `main`. Work on a branch named `<type>/<short-name>`
   (`docs/data-model`, `chore/repo-setup`); there are no task ids yet.
2. Never push, merge, tag or deploy. Prepare the command and hand it to the
   engineer, who pushes and fast-forwards `main` by hand.
3. Never commit a secret, key or `.env` file, and never read one into context.
   `.env.example` holds names and local demo values only.
4. Never rewrite history: no amend, rebase, force-push or `reset --hard`.
5. Never weaken a guardrail to go green: no lint suppression, no `any` or
   `@ts-ignore`, no skipped test, no lowered threshold.
6. Never add a dependency, model, service or feature nobody asked for;
   propose it with the reason. Choices of libraries follow an ADR.
7. Never invent an API field, table, variable or command; find it in
   `api/openapi.yaml`, `docs/design/data-model.md` or the code first.
8. Stay in scope. What you notice in passing goes under "Noticed", unfixed.
9. Report honestly: a check you did not run is "not run", never "passing".
10. No AI attribution in commits or documents. No em dashes; rewrite the sentence.

## Commands (every one is a Makefile target; `make help` lists them)

- First time: `make setup` (Go modules, web packages, git hooks), then
  `cp .env.example .env`, set `JWT_SECRET` in `.env` (`openssl rand -hex 32`;
  the server refuses to start without it), and `make db` (PostgreSQL and
  MailHog in Docker). An `.env` copied before phase 1 lacks `JWT_SECRET`,
  `USERS_FILE`, `BRAND_NAME` and `BRAND_TIMEZONE`: copy them from
  `.env.example`. `make dev` reads `.env` as shell, so quote a value with a
  space. When a port is taken, change `POSTGRES_PORT` and the port in
  `DATABASE_URL` together; `make migrate` and `make test-integration` use
  that `DATABASE_URL` unless one is given on the command line. `make migrate`
  also applies the budget set (`db/migrations/budget`), which has no Down and
  is never rolled back. `MODEL_GATEWAY_MODE` defaults to `replay`, which
  answers only from `testdata/recordings` and spends nothing; `live` and
  `record` need `OPENROUTER_API_KEY` and spend real budget.
- Accounts: `cp users.example.json users.json` (ignored by git), then replace
  each `password_hash` with the output of `make hash-password`. Start with the
  admin entry only: a manager entry naming an outlet that does not exist yet
  is skipped with a warning. Add the outlets as the admin, then the manager
  entries, and restart. `make migrate` before the first `make dev`; the
  server refuses to start without the tables or with a bad users file.
- Run: `make dev` (server on :8080), `make web-dev` (Vite on :5173, forwards /api).
- Gate: `make check` runs go-fmt-check, go-vet, go-lint, go-test,
  web-format-check, web-lint, web-api-types-check, web-typecheck and
  web-test, offline. A missing tool is recorded as skipped and fails the
  gate. web-api-types-check fails when `web/src/lib/api-types.ts` is not what
  `api/openapi.yaml` generates: run `make web-api-types` and commit both.
- Outside the gate: `make build`, `make test-integration` (needs `make db`
  and goose; each package gets its own database), `make vuln` (needs the
  network), `make migrate`, `make sqlc`, `make web-api-types`, `make fix`,
  `make doctor`.
- Toolchain: Go 1.26.8; sqlc v1.31.1, goose v3.28.0, golangci-lint v2.13.2 and
  govulncheck v1.8.0 installed with `go install` into `$(go env GOPATH)/bin`
  (the Makefile puts it on PATH); Node 24 (`web/.nvmrc`, run `nvm use` in
  `web/`) with pnpm 12.9.1 through corepack.

## Rules that come from the design

- Tenets in `docs/architecture/tenets.md` are binding: one door to the model
  (the gateway); the budget total is read from PostgreSQL before every live
  call; outlet scope is a query condition taken from the user row re-read on
  each request; tag results are matched by review id, never by position;
  tests never reach the network (replay only); review text, names and drafts
  are rendered as plain text; the OpenAPI spec changes in the same commit as
  the handler; writes with side effects are safe to repeat.
- Migrations are goose files in `db/migrations/`, append-only once applied,
  numbered as planned in `docs/design/data-model.md` section 8. The
  `budget` schema has its own migration set with no Down and is never
  truncated by the seed.
- A change to a table, column or response shape updates
  `docs/design/data-model.md` (or `api/openapi.yaml`) in the same branch.
- Prompt versions are immutable files; a change is a new version.
- `web/pnpm-workspace.yaml` allows build scripts for esbuild only; any other
  package that needs one is a decision for the engineer.

## Commits

Conventional subjects with no task id, summary at most 72 characters:
`docs: add data model, schema, data dictionary and ERD`. The commit-msg hook
checks this, plus no AI trailer and no em dash.

## Reporting

End every task in this shape: `## Changed` (path: what and why), `## Verified`
(the command and the tail of its output), `## Not done` (asked, not delivered,
why), `## Noticed` (seen in passing, not fixed).
