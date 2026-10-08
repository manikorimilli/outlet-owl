# OutletOwl

Review intelligence for one brand with several outlets: import reviews,
tag them by theme, sentiment and urgency through a budgeted model gateway,
see where complaints move, and draft replies that a manager approves.

- Go API (`cmd/api`, `internal/`) with PostgreSQL; React web app (`web/`)
  served by the same binary.
- Local demo only: PostgreSQL and MailHog run in Docker; model calls go
  through OpenRouter.

## Quick start

```bash
make setup
cp .env.example .env   # then set JWT_SECRET and free ports, see the guide
make db && make migrate
make seed SEED_ARGS="-tag=false"
make build
make dev               # API on http://127.0.0.1:8080
make web-dev           # in a second terminal: http://localhost:5173
```

Sign in as `ritika.rao@example.in` (brand admin) or `manager1@example.in`
to `manager5@example.in`, password `outletowl-demo`.

## Read next

- [Running OutletOwl locally](docs/runbooks/local-operations.md): setup,
  daily use, resetting data, model modes and fixes for common failures.
- [AGENTS.md](AGENTS.md): the working rules and every make target.
- [Product](docs/product/PRD.md), [design](docs/design/review-intelligence-hld.md),
  [decisions](docs/architecture/decisions.md),
  [threat model](docs/security/threat-model-outletowl.md).
