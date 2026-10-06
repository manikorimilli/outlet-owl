# ADR-0005: Use REST with an OpenAPI spec between the UI and the server

- Status: Accepted
- Date: 2026-10-06
- Task: none (stack selection before the first task)
- Deciders: the product owner, in session
- Area: api style
- Reversibility: cheap: the endpoints stay REST either way; dropping the spec only removes the generated types

## Context

- The Go server (ADR-0001) and the React UI (ADR-0002) are separate codebases; ADR-0001 names keeping the contract between them in step as the main cost of choosing Go.
- There is one browser client and no public API.
- About 12 to 15 endpoints are expected (an estimate, not measured): sign-in, outlets, CSV import, review list and search, review detail, draft, mark replied, trends, heatmap, movers, digest.

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| REST + OpenAPI with generated TypeScript types (chosen) | A spec file to maintain and a generate step | Two codebases in two languages sharing one contract |
| Plain REST, no spec | TypeScript types hand-copied from Go structs drift silently | A handful of endpoints in one language |
| GraphQL | Schema and server library for one client and about 15 endpoints | Many clients needing different shapes of one graph |
| gRPC or Connect | Browser support needs extra tooling | Service-to-service calls |

## Decision

We will expose a REST API described by one OpenAPI spec and generate the UI's TypeScript types from it, because both sides then build from one contract and a mismatch surfaces at build time, which addresses the cost ADR-0001 accepted. The product owner chose the recommended option.

## Consequences

- The OpenAPI spec is the source of truth for request and response shapes; `bearing:openapi-spec` writes and checks it.
- The UI's API types are generated from the spec and never edited by hand; the generator tools are chosen at implementation.
- CI (ADR-0004) fails when generated types are out of date with the spec.
- Revisit if a second kind of client (mobile, partner API) needs a different shape of the data.

## Commits us to

REST, OpenAPI 3
