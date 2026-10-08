# Running OutletOwl locally

How to set up, start, reset and fix the local demo. Every command is a
Makefile target or a shell line run from the repository root. Last checked
against main at 32ddfc3 on 2026-10-08.

## What runs where

| Part | Address | Started by |
| --- | --- | --- |
| API and the built UI | http://127.0.0.1:8080 (`HOST`, `PORT`) | `make dev` |
| UI with hot reload | http://localhost:5173, forwards `/api` to 127.0.0.1:8080 | `make web-dev` |
| PostgreSQL 16 | 127.0.0.1, `POSTGRES_PORT` | `make db` |
| MailHog (digest mail) | SMTP on `MAILHOG_SMTP_PORT`, inbox on http://127.0.0.1:`MAILHOG_UI_PORT` | `make db` |
| Model gateway | OpenRouter, only in `record` or `live` mode | the API and the CLIs |

## First setup

1. `make setup` (Go modules, web packages, git hooks).
2. `cp .env.example .env`, then edit `.env`:
   - `JWT_SECRET`: at least 32 characters; `openssl rand -hex 32` makes one.
   - Ports: if another project already uses 5432, 1025 or 8025, set
     `POSTGRES_PORT`, `MAILHOG_SMTP_PORT`, `MAILHOG_UI_PORT`, the port inside
     `DATABASE_URL` and `SMTP_ADDR` to free ones (for example 55432, 51025
     and 58025). Change `POSTGRES_PORT` and `DATABASE_URL` together.
3. `make db` (PostgreSQL and MailHog in Docker).
4. `make migrate` (both migration sets).
5. `make seed SEED_ARGS="-tag=false"` (5 outlets, 1,500 reviews, the demo
   accounts in `users.json`; no model call).
6. `make build` once, so `web/dist` exists for the API to serve.

## Every day

- Start: `make db`, then `make dev` in one terminal and `make web-dev` in
  another. Open http://localhost:5173. Under `make web-dev` the sign-in form
  is filled with the demo admin.
- Accounts (password `outletowl-demo` unless `SEED_PASSWORD` was set):
  `ritika.rao@example.in` is the brand admin; `manager1@example.in` to
  `manager5@example.in` manage one outlet each (`manager2` is Koramangala).
- Stop: Ctrl+C in both terminals, then `make db-down` (data kept).

## Reset the demo data

- `make seed`: truncates the domain tables (never the budget schema),
  rewrites `users.json` and seeds again. It overwrites both, so keep a copy
  of a hand-edited `users.json`.
- `SEED_ARGS="-tag=false"`: no tagging; the reviews stay untagged.
- `SEED_ARGS="-reviews=500"`: a smaller set (at least 145), about 25 tagging
  calls instead of 75. The Koramangala wait-time spike stays.
- `make db-reset`: drops the whole database volume, budget history included
  (destructive), then run `make migrate` and `make seed` again.

## Model modes and cost

- `MODEL_GATEWAY_MODE=replay` (the default) answers only from
  `testdata/recordings` and spends nothing. A missing recording means no tag
  and "drafting unavailable".
- `record` and `live` call OpenRouter and need `OPENROUTER_API_KEY`.
  `record` also saves each answer so later runs can replay it.
- `MODEL_ID` picks the model; empty is Claude Haiku 4.5 (ADR-0009). A
  `:free` model costs nothing but has rate limits, and its provider may log
  the prompts: use it only with the synthetic demo data.
- The `budget` log line at start shows the running total against the USD 8
  stop. It counts a worst-case price for calls that failed (a 429 still
  counts its reservation), so it can be above what OpenRouter billed.

## When something goes wrong

| What you see | Cause | Fix |
| --- | --- | --- |
| `bind: address already in use` on start | another server already holds the port, often an older `make dev` in another terminal | `ss -ltnp \| grep ':8080 '` names the process; press Ctrl+C in its terminal, then start again |
| `make db` fails with `port is already allocated` | another project's container uses that host port | set the free ports in `.env` (First setup, step 2), `docker compose down`, then `make db` |
| `make migrate` or `make seed`: `password authentication failed` on port 5432 | `DATABASE_URL` still points at another PostgreSQL | fix the port in `DATABASE_URL` to match `POSTGRES_PORT` |
| `JWT_SECRET is required` or `must be at least 32 bytes` | the value is empty or short in `.env` | set a 32-character or longer value |
| Log shows `OpenRouter answered 429` and tagging stops | the free tier's limit: 20 requests a minute, and 50 a day on an account that has bought less than USD 10 of credit (1,000 a day after) | wait for the limit to reset, use `-reviews=500`, or empty `MODEL_ID` to use the paid default |
| Reviews say "Not tagged yet" | no model answer yet (replay without recordings, a 429, or the budget stop) | fix the cause above, then restart `make dev`: the worker tags untagged reviews at start and after each import |
| The review page says "Drafting is unavailable" | the draft call failed (same causes) | the manager writes the reply by hand and marks it replied; drafting works again once the model answers |
| "The model is still drafting this reply" on save | another tab or request holds the drafting claim | wait a few seconds and press "Reload the reply"; a claim older than 60 s is taken over |
| The digest says the mail catcher did not answer | MailHog is not running or `SMTP_ADDR` names the wrong port | `make db`, and match `SMTP_ADDR` to `MAILHOG_SMTP_PORT` |
| Another device cannot open the app | the server listens on 127.0.0.1 by design (threat model T-07) | set `HOST=0.0.0.0` only if you mean to open it to the network |

## Checks

- `make check`: the offline gate (format, vet, lint, Go and web tests, API
  types). It needs no database.
- `make test-integration`: needs `make db` and goose; each package gets its
  own database.
- `make eval`: needs the 100 labelled reviews (phase 7 LLD) and model access;
  exits 1 below 90% urgent recall.
