# ADR-0010: Deploy the public demo as one container on Azure Container Apps

- Status: Accepted
- Date: 2026-10-13
- Task: none
- Deciders: the product owner, 2026-10-13
- Area: deployment
- Reversibility: cheap: the same image runs anywhere; delete the Azure resource group to undo

## Context

- The PRD and HLD describe a local product: the Go binary serves the web app on `http://localhost` (ADR-0002), the session cookie is HttpOnly and SameSite=Strict with no Secure flag (ADR-0007), and `CrossOriginProtection` refuses writes from another origin.
- The product owner wants a public demo link on Azure, using their Azure credits, with both the web app and the API there.
- Serving the web app from a second origin (a static host) would make the browser treat the API as another site: the Strict cookie would not be sent and every write would be refused, unless a proxy forwarded `/api` or the cookie and CORS rules were loosened.

## Decision

- **One container serves the web app and the API from one HTTPS address**, exactly as locally (ADR-0002). The image (`Dockerfile`) builds the web app with Node and the server with Go, and the server serves `web/dist` from the image. No proxy, no CORS, and the cookie and cross-site rules stay as they are.
- **Azure Container Apps** (Central India) runs the image from GitHub Container Registry, built by the `image` workflow on each merge to main, with **exactly one replica** (min 1, max 1): the tagging worker and the sign-in limiter live in the process. PostgreSQL is Azure Database for PostgreSQL Flexible Server. The Container Apps ingress serves HTTPS only.
- **Production settings:** `COOKIE_SECURE=true` (Secure cookie and HSTS), `HOST=0.0.0.0` (set in the image), `MODEL_GATEWAY_MODE=live` with a free model.
- **Demo access on purpose:** every account keeps the password `outletowl-demo`, and the image is built with `DEMO_PREFILL=true` so the sign-in form is prefilled and says it is a demo. The data is the synthetic seed only. A build without the flag carries neither the account nor the note.
- **Hardening that ships first:** five wrong passwords per email in 15 minutes lock it (T-01), every response carries security headers (T-17), Go 1.26.9 (no known standard library vulnerabilities).

## Consequences

- Anyone with the link can sign in as the brand admin and change the demo data, use AI drafting and send the digest to the configured address. The budget stop (USD 8) and free models (USD 0) bound the spend; restoring the database dump resets the data.
- Free models may log prompts (T-21): the demo must never hold real reviews.
- The sign-in limit is per process; a second replica would keep its own count. Scaling out needs a shared store first.
- There is no SMTP relay on Azure for now: the digest fails with `mail_unavailable` until an SMTP service is configured.
- A new image does not deploy itself: the operator creates a new revision (docs/runbooks/deploy-azure.md).
- Rejected: the web app on Vercel with `/api` forwarded to Azure (two products to run, a trusted-origin setting for older browsers); two origins with SameSite=None and CORS (weakens ADR-0007); Render (the owner's credits are on Azure).
