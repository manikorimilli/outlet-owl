# Threat model: OutletOwl (whole system, main at 5c99352)

- Task: none
- Serves: US-00-001, US-00-002, US-00-003, US-01-002, US-02-001, ADR-0007, ADR-0008, ADR-0009
- HLD: docs/design/review-intelligence-hld.md; diagram: none (no C4 file; boundaries below come from docker-compose.yml, cmd/api/main.go and the HLD)
- Sensitive classes: auth, PII (reviewer names, staff emails), external input (CSV import, review text sent to a model)
- Author: unattributed, 2026-10-08, status Draft. Abuse-path pass: not run (the security-threat-model skill is not installed).

Deployment today: one machine, local demo only (PRD section 6). Section 5 lists what must change before any shared or internet deployment.

## 1. Assets

| Asset | Where it lives | Owner | Why an attacker wants it |
| --- | --- | --- | --- |
| Session token (HS256 JWT) | `outletowl_session` cookie, 8 h | backend role | act as an admin or a manager |
| JWT signing secret | `.env` `JWT_SECRET` | operator role | mint a token for any user id |
| OpenRouter API key | `.env` `OPENROUTER_API_KEY` | operator role | spend the brand's model budget, read usage |
| Staff accounts (emails, bcrypt hashes, roles, outlets) | `users.json`, table `users` | operator role | sign in, learn who manages which outlet |
| Reviews (reviewer names, texts) | tables `reviews`, `review_tags` | brand admin role | customer data, per-outlet complaints |
| Replies and approval record | table `replies` (`replied_by`, `replied_at`) | outlet manager role | post text in the brand's name, deny an approval |
| Model budget | schema `budget`, USD 8 stop | operator role | drain it so tagging and drafting stop |
| Digest mail | table `digests`, MailHog | brand admin role | read the weekly summary, spoof it |

## 2. Trust boundaries

| # | From | To | Protocol | Auth on the edge | Source |
| --- | --- | --- | --- | --- | --- |
| B1 | browser (any host that can reach the port) | api :8080 | HTTP, no TLS | session cookie; sign-in is public | cmd/api/main.go:107 |
| B2 | api | PostgreSQL | TCP on 127.0.0.1 | password from `DATABASE_URL` | docker-compose.yml:12 |
| B3 | api | OpenRouter | HTTPS | bearer API key | internal/gateway/openrouter.go:37 |
| B4 | api | MailHog | SMTP on 127.0.0.1, no auth | none | docker-compose.yml:21 |
| B5 | outlet manager role | brand admin role, other outlets | in-process | role and outlet re-read per request | internal/auth/service.go:46 |
| B6 | operator files (`.env`, `users.json`) | the process | file read at start | file system permissions | .gitignore:13 |
| B7 | review text (customers) | model prompt | in-process string | none; the text is untrusted | internal/tagging/message.go:41 |

## 3. Entry points

| # | Entry point | Kind | Auth required | Boundary |
| --- | --- | --- | --- | --- |
| E1 | POST /api/v1/auth/login | route | none | B1 |
| E2 | POST /api/v1/auth/logout, GET /api/v1/me | route | session | B1 |
| E3 | GET, POST /api/v1/outlets | route | session; POST brand admin | B1, B5 |
| E4 | POST /api/v1/imports (CSV upload) | upload | session, brand admin | B1, B5, B7 |
| E5 | GET /api/v1/reviews, /reviews/{id}, /themes, /dashboard/*, /tagging/status | route | session, outlet scoped | B1, B5 |
| E6 | POST /reviews/{id}/draft, PUT /reviews/{id}/reply, POST /reviews/{id}/replied | route | session, that outlet's manager | B1, B5, B7 |
| E7 | POST /api/v1/digests | route | session, brand admin, Idempotency-Key | B1, B4 |
| E8 | GET /* (the built UI from `web/dist`) | static files | none | B1 |
| E9 | tagging worker, `make seed`, `make eval`, `make tone-check` | in-process job, CLI | operator shell | B3, B6, B7 |
| E10 | `users.json` and `.env` read at start | config file | file system | B6 |

## 4. Threats (STRIDE)

| Id | Entry or boundary | Category | Threat | Likelihood | Impact | Mitigation | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| T-01 | E1 | Spoofing | password guessing against a known staff email, with no limit on attempts | M | H | new story: limit failed sign-ins per email and per client address | unmitigated |
| T-02 | E1 | Information disclosure | timing of sign-in tells which emails have accounts | L | L | an unknown email still runs bcrypt against a dummy hash, internal/auth/password.go:60 | mitigated |
| T-03 | E2, B1 | Spoofing | a forged token signed with another key or algorithm | L | H | only HS256 with the server secret verifies, internal/auth/token.go:69; secret of at least 32 bytes, internal/config/config.go:85 | mitigated |
| T-04 | E2, B1 | Spoofing | a stolen cookie read by page script | L | H | HttpOnly and SameSite=Strict cookie, internal/httpapi/session.go:30 | mitigated |
| T-05 | E2, B1 | Spoofing | a copied token still works after sign-out or role change, for up to 8 h | M | M | role and active flag re-read on every request, internal/auth/service.go:107; sign-out does not revoke, accepted in ADR-0007 | mitigated |
| T-06 | B1 | Information disclosure | the cookie travels in clear text over HTTP and has no Secure flag | M | H | new story: serve over HTTPS with a Secure cookie before any non-local deployment | planned |
| T-07 | B1 | Spoofing | another device on the same network reaches the API, since it listens on all interfaces | M | H | new story: bind the API to 127.0.0.1 by default and make the address a setting | unmitigated |
| T-08 | E3 to E7 | Tampering | cross-site request forgery: another site makes the browser send a write | L | H | Go CrossOriginProtection refuses cross-site writes, internal/middleware/crossorigin.go:29; SameSite=Strict, internal/httpapi/session.go:31 | mitigated |
| T-09 | E5, B5 | Elevation of privilege | a manager reads another outlet's review, list, dashboard or theme counts by id or filter | M | H | every query ANDs all_outlets OR outlet_id, db/queries/reviews.sql:18, db/queries/dashboard.sql:18; scope from the re-read user, internal/auth/service.go:46 | mitigated |
| T-10 | E6, B5 | Elevation of privilege | a manager drafts, edits or approves another outlet's reply | M | H | manager check before any write, internal/replies/service.go:144; writes bound to the outlet, db/queries/replies.sql:53 | mitigated |
| T-11 | E3, E4, E7, B5 | Elevation of privilege | a manager adds outlets, imports reviews or sends the digest | M | M | brand admin checks, internal/imports/service.go:75, internal/outlets/service.go:115, internal/httpapi/digest_handler.go:38 | mitigated |
| T-12 | E4 | Denial of service | a huge or malformed CSV exhausts memory or time | L | M | 5 MB file cap, internal/connector/csv.go:18; body cap, internal/httpapi/imports_handler.go:75; admin only | mitigated |
| T-13 | E4 | Tampering | CSV cells with NUL bytes, bad UTF-8 or formula prefixes poison stored data | L | L | NUL and invalid UTF-8 refused per cell, internal/connector/csv.go:173; data is never exported as a spreadsheet | mitigated |
| T-14 | E5 | Tampering | SQL injection through search, filters or the cursor | L | H | sqlc parameterised queries; LIKE escaped, internal/reviews/service.go:127; cursor validated, internal/reviews/cursor.go:38 | mitigated |
| T-15 | E5, E6, E8 | Tampering | stored XSS through a review text, reviewer name or reply | M | H | rendered as React text only, web/src/features/reply/ReviewPage.tsx:103; no innerHTML in web/src | mitigated |
| T-16 | E8 | Information disclosure | path traversal through the static file handler reads files outside web/dist | L | H | path cleaned under the web root, internal/httpapi/web.go:22 | mitigated |
| T-17 | E8 | Information disclosure | no security headers (nosniff, frame-ancestors, CSP), so the UI can be framed for clickjacking | L | M | new story: set X-Content-Type-Options, a frame-ancestors CSP and Referrer-Policy on every response | unmitigated |
| T-18 | B7, E6 | Tampering | prompt injection in a review steers the drafted reply | M | M | review text cannot close its tag, internal/replies/message.go:45; the manager reads and edits every draft before approving, US-00-003 | mitigated |
| T-19 | B7, E9 | Tampering | prompt injection in a review mis-tags itself or other reviews in the batch | M | M | review tags stripped, internal/tagging/message.go:41; answers validated per id in the batch, internal/tagging/parse.go:46 | mitigated |
| T-20 | E6, E9, B3 | Denial of service | repeated draft requests or imports drain the model budget | M | M | every call reserves a worst-case cost against the USD 8 stop first, internal/gateway/gateway.go:132 | mitigated |
| T-21 | B3 | Information disclosure | a free model (MODEL_ID ending in :free) may log or train on review texts and names | M | M | default stays Claude Haiku 4.5, ADR-0009; new story: warn at start when MODEL_ID names a free model | planned |
| T-22 | E10, B6 | Information disclosure | `.env` or `users.json` committed or shared, leaking the JWT secret, the API key or password hashes | L | H | both ignored by git, .gitignore:13 and .gitignore:18; hashes are bcrypt cost 12, internal/auth/usersfile.go:234 | mitigated |
| T-23 | E10 | Spoofing | the seed writes every demo account with the same known password | H | H | demo password only for local use; new story: refuse to seed when the environment is not local, or require SEED_PASSWORD | planned |
| T-24 | E7, B4 | Spoofing | a forged digest mail or header injection through the subject | L | L | plain text body and Q-encoded one-line subject, internal/digest/mail.go:54; sent only to the active admin, internal/digest/service.go:97 | mitigated |
| T-25 | E6 | Repudiation | a manager denies approving a reply | L | M | a replied row must carry replied_by and replied_at, db/migrations/00003_create_replies.sql:39; request log carries user_id, internal/middleware/requestlog.go:40 | mitigated |
| T-26 | E9 | Repudiation | model spend with no record of who or what called | L | L | one budget.model_calls row per attempt with purpose and model, db/migrations/budget/00001_create_model_calls.sql:23 | mitigated |
| T-27 | E1 to E8 | Information disclosure | internal errors or SQL text leak to the client | L | M | unknown errors answer a fixed 500 message, internal/httpapi/errors.go:126 | mitigated |
| T-28 | E1 to E8 | Denial of service | large JSON bodies | L | L | JSON bodies capped at 64 KiB, internal/httpapi/decode.go:33 | mitigated |
| T-29 | B2, B4 | Spoofing | another host on the network reaches PostgreSQL or MailHog | L | H | both published on 127.0.0.1 only, docker-compose.yml:12, docker-compose.yml:21 | mitigated |
| T-30 | E9 | Elevation of privilege | considered, none: the worker and CLIs run as the operator who already holds `.env`; no network entry reaches them | | | | |

## 5. Residual risks

| Threat | Risk accepted or planned | Owner (role) | Revisit |
| --- | --- | --- | --- |
| T-01 | accepted while local only; must ship before any shared deployment | backend lead role | 2026-11-01 |
| T-06 | planned before any non-local deployment | operator role | 2026-11-01 |
| T-07 | accepted while the demo machine is trusted; fix is small | backend lead role | 2026-11-01 |
| T-17 | accepted while local only | backend lead role | 2026-11-01 |
| T-21 | planned: free models only for synthetic demo data | product owner role | 2026-10-15 |
| T-23 | accepted for the demo seed; never seed a shared database | operator role | 2026-11-01 |

Before any shared or internet deployment: T-01, T-06, T-07, T-17 and T-23 must be closed, `JWT_SECRET` must come from a secret store, MailHog replaced by a real relay with auth, and MODEL_ID left empty or set to a paid model with a data agreement.

## 6. New stories needed

- Limit failed sign-ins per email and per client address (mitigates T-01), acceptance: the sixth wrong password within 15 minutes answers 429 and a correct one is still refused until the window ends.
- Serve over HTTPS with a Secure cookie before any non-local deployment (mitigates T-06), acceptance: the session cookie carries Secure and plain HTTP redirects to HTTPS.
- Bind the API to 127.0.0.1 by default and make the address a setting (mitigates T-07), acceptance: with defaults, a request from another host is refused at connect.
- Set X-Content-Type-Options, a frame-ancestors CSP and Referrer-Policy on every response (mitigates T-17), acceptance: every response carries the three headers and the UI cannot load in a frame.
- Warn at start when MODEL_ID names a free model (mitigates T-21), acceptance: a `:free` model id logs a warning naming the data risk.
- Refuse to seed when the environment is not local, or require SEED_PASSWORD (mitigates T-23), acceptance: a DATABASE_URL host other than localhost without SEED_PASSWORD stops the seed.

## 7. Counts

Assets 8, boundaries 7 (assumed 0; no C4 diagram, all read from files), entry points 10, threats 29 (mitigated 23, planned 3, unmitigated 3), plus T-30 considered, none. Gate: passed.
