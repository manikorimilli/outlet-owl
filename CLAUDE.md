@AGENTS.md

# CLAUDE.md

Claude Code specifics for this repository; the standard is AGENTS.md above.
Loaded into every session: keep this file short.

```
Repository:   OutletOwl (Go server at the root, React web app in web/)
Stack:        Go 1.26.8 + pgx + sqlc + goose; React 19 + Vite 7 + TypeScript
Databases:    PostgreSQL 16 (Docker); MailHog for the digest
Entrypoint:   cmd/api/main.go; web/src/main.tsx
Run, test:    make dev, make web-dev; gate: make check
Git host:     GitHub (manikorimilli/outlet-owl)
Tracker:      none
Trunk:        main (the engineer fast-forwards and pushes it)
```

Plan mode for anything over three files or touching a schema, contract or
auth. On a third failed approach, stop and say so. Bound shell output with
`| tail -40`.

Every skill step asks before it runs the next one; never start the next
Bearing skill without the engineer's go-ahead.

## Things the agent gets wrong in this repository

Add a line each time a mistake repeats; delete lines that stop applying.
