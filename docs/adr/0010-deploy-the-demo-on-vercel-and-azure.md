# ADR-0010: Deploy the public demo on Vercel and Azure Container Apps

- Status: Accepted
- Date: 2026-10-13
- Task: none
- Deciders: the product owner, 2026-10-13
- Area: deployment
- Reversibility: cheap: the same image runs anywhere; delete the Azure resource group and the Vercel project to undo

## Context

- The PRD and HLD describe a local product: the Go binary serves the web app on `http://localhost` (ADR-0002), the session cookie is HttpOnly and SameSite=Strict with no Secure flag (ADR-0007), and `CrossOriginProtection` refuses writes from another origin.
- The product owner wants a public demo link. They chose Vercel for the web app and Azure for the server, using their Azure credits.
- Served from two origins, the browser would treat the API as another site: the Strict cookie would not be sent and every write would be refused. Making that work means a SameSite=None cookie and CORS, which weakens ADR-0007.

## Decision

- **Vercel serves the web app and forwards `/api/*` to the Go server** with a rewrite (`web/vercel.json`). The browser only ever talks to the Vercel origin, so the cookie stays first-party and Strict, and `Sec-Fetch-Site: same-origin` passes the write check. No CORS.
- **The Go server runs on Azure Container Apps** (Central India) from an image built by GitHub Actions and stored in GitHub Container Registry, with **exactly one replica** (min 1, max 1): the tagging worker and the sign-in limiter live in the process. PostgreSQL is Azure Database for PostgreSQL Flexible Server.
- **Production settings:** `COOKIE_SECURE=true` (Secure cookie and HSTS; both hops are HTTPS), `TRUSTED_ORIGINS` set to the Vercel address for browsers that send no Sec-Fetch-Site through the proxy, `HOST=0.0.0.0`, `MODEL_GATEWAY_MODE=live`.
- **Demo access on purpose:** every account keeps the password `outletowl-demo` and the sign-in form is prefilled when the web build sets `VITE_DEMO_PREFILL=true`, with a note on the page saying it is a demo. The data is the synthetic seed only.
- **Hardening that ships first:** five wrong passwords per email in 15 minutes lock it (T-01) and every response carries security headers (T-17).

## Consequences

- Anyone with the link can sign in as the brand admin and change the demo data, use AI drafting and send the digest to the configured address. The budget stop (USD 8) and free models (USD 0) bound the spend; `make seed` against the Azure database resets the data.
- Free models may log prompts (T-21): the demo must never hold real reviews.
- The sign-in limit is per process; a second replica would keep its own count. Scaling out needs a shared store first.
- There is no SMTP relay on Azure for now: the digest fails with `mail_unavailable` until an SMTP service is configured.
- Each merge to main builds a new image; the Container App picks it up when the operator updates the revision (docs/runbooks/deploy-azure.md).
- Rejected: two origins with SameSite=None and CORS (weakens ADR-0007); Render (simpler, but the owner's credits are on Azure).
