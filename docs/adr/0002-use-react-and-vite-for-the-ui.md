# ADR-0002: Use React with Vite for the UI, as a single-page app

- Status: Accepted
- Date: 2026-10-06
- Task: none (stack selection before the first task)
- Deciders: the product owner, in session
- Area: frontend
- Reversibility: awkward: moving to another framework means rewriting every screen; the Go API is unaffected

## Context

- Every screen is behind sign-in (Q-003); nothing needs search-engine indexing.
- The UI is dashboard heavy: rating and sentiment trends, a theme heatmap, movers, a filterable review list, reply editing (REQ-015 to REQ-024, REQ-044 to REQ-047).
- English and Devanagari Hindi text must render legibly (REQ-048, Q-009).
- The server is Go (ADR-0001). The runtime allows nothing beyond the app, PostgreSQL and MailHog (PRD section 6), so a UI framework that needs its own Node server at runtime would add a second backend process.

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| React + Vite SPA (chosen) | Routing and data fetching are assembled from libraries rather than built in | Signed-in dashboards served as static files |
| Next.js | Needs a Node server at runtime beside Go; server rendering adds nothing to a signed-in dashboard | Public pages that must be indexed or server-rendered |
| SvelteKit (static) | Smaller ecosystem for charts and complex tables | A team already working in Svelte |
| Vue + Vite | Smaller dashboard ecosystem than React | A team already working in Vue |

## Decision

We will build the UI as a React single-page app with Vite, built to static files that the Go server serves, because it adds no runtime process, has the widest choice of chart, heatmap and table libraries, and is covered by the installed `bearing-apps:react` skill. The product owner chose the recommended option.

## Consequences

- The built UI is static files; the Go binary serves them, so the runtime stays the Go binary, PostgreSQL and MailHog.
- Chart, table and routing libraries are chosen at design time (`screen-design`, `design-system`), not here.
- No UI framework or component library beyond React and Vite is selected by this ADR.
- Revisit if a public, search-indexed page becomes a requirement.

## Commits us to

React, Vite, TypeScript, Node (build time only)
