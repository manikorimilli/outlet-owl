# ADR-0001: Use Go for the server, with a separate TypeScript UI

- Status: Accepted
- Date: 2026-10-06
- Task: none (stack selection before the first task)
- Deciders: the product owner, in session
- Area: backend language
- Reversibility: awkward: switching language later means rewriting the API, the model gateway, the seed script and their tests

## Context

- The runtime is fixed by the PRD (section 6, brief L76 to L80): PostgreSQL in Docker, MailHog in Docker, an OpenRouter key capped at USD 10, and nothing else. The product runs locally; no cloud.
- The AI work is two prompt types sent over HTTP to OpenRouter (batch tagging, reply drafting), a budget-capped gateway (REQ-030 to REQ-032), replayed recordings in tests (REQ-034, REQ-035) and simple scoring for evaluations (REQ-036 to REQ-039). No machine-learning library is needed.
- The UI is dashboard heavy (REQ-015 to REQ-019, REQ-044 to REQ-048) and is built in TypeScript whichever server language is chosen.
- The repository has no code yet. Go is not installed on the development machine as of 2026-10-06 (Node 20 and Python 3.12 are).

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| Go API + TypeScript UI (chosen) | Two languages and toolchains (Go for the server, Node only to build the UI) and an API contract to keep in step; fewer ready-made LLM client libraries, so the OpenRouter call is plain `net/http` | A team that wants one small, fast server binary that also serves the built UI |
| TypeScript full stack (recommended in session) | Python-style data tooling unavailable for evaluations (not needed at this size) | A team that wants one language, one toolchain and one test runner end to end |
| Python (FastAPI) API + TypeScript UI | Two languages, two test setups and an API contract | A team stronger in Python or planning Python-only evaluation tooling |
| Java or Kotlin (Spring) API + TypeScript UI | Heaviest setup for a medium, local-only project | An existing JVM estate |

## Decision

We will write the server in Go, with the UI as a separate TypeScript application, because the product owner chose it in session over the recommended TypeScript full stack. The server runtime is a single Go binary, which keeps the "nothing else" runtime to that binary, PostgreSQL and MailHog.

## Consequences

- Two toolchains in development and CI: Go for the server, Node for building and testing the UI. Node is a build-time tool only.
- The API between the Go server and the UI is a contract both sides must keep in step; its style is a separate decision.
- The OpenRouter call, batch validation per review ID and recording/replay live in Go behind one gateway function.
- Go 1.25 or newer must be installed before implementation; not installed yet.
- The `bearing-backend:go` house rules apply: standard library `net/http`, `pgx`, `sqlc`, `goose`, `slog`, table-driven and `httptest` tests.
- Revisit if the evaluation work grows into data-science tooling that only exists in Python, or if keeping the API contract in step becomes the main source of defects.

## Commits us to

Go 1.25 or newer, Go standard library net/http, slog
