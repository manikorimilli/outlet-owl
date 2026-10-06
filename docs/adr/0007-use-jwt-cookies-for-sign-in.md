# ADR-0007: Use a signed JWT in an HttpOnly cookie for sign-in

- Status: Accepted
- Date: 2026-10-06
- Task: none (design before the first task)
- Deciders: the product owner, in session
- Area: auth
- Reversibility: cheap: moving to server-side sessions changes the sign-in handler and the request check, not the screens

## Context

- Accounts are created by the seed or configuration; there is no user management screen (Q-003). The demo has about 6 accounts: one brand admin and one manager per outlet (US-02-006).
- Two roles: the brand admin sees all outlets; an outlet manager sees and acts on one assigned outlet; only managers approve replies (Q-002).
- The React UI is served by the Go binary from the same origin (ADR-0002, ADR-0005).
- No hosted identity provider or other runtime service is allowed (PRD section 6, brief L80).

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| Signed JWT in an HttpOnly cookie (chosen) | Cannot be revoked before it expires; a signing secret must be kept out of git and rotated if leaked | Several services checking sign-in without sharing a database |
| Server-side sessions in PostgreSQL (recommended in session) | One table and one lookup per request | A single server that wants immediate sign-out and no signing secret |
| Self-hosted Keycloak | A runtime service the PRD forbids | Single sign-on across many applications |
| HTTP Basic auth | No real sign-out; the browser keeps the credentials | Throwaway internal tools |

## Decision

We will sign users in with a password checked against a bcrypt hash and issue a signed JWT in an HttpOnly, SameSite=Strict cookie, because the product owner chose it in session over the recommended server-side sessions.

## Consequences

- A signing secret is needed at runtime, read from the environment, never committed, and changed if it leaks (changing it signs everyone out).
- Signing out clears the cookie; a copied token stays valid until it expires. The expiry is short, set in design.
- The token carries the user id only for identity; the server re-reads the user's role and assigned outlet from PostgreSQL on every request, so a changed assignment (Q-003) takes effect at once and authorisation never trusts stale claims.
- Role and outlet checks are enforced at the service on every request, not only in the UI (US-01-001, US-00-001, US-00-003).
- Revisit if immediate revocation becomes a requirement or a second service needs the identity.

## Commits us to

JWT (a well-reviewed Go JWT library, chosen at implementation), bcrypt (golang.org/x/crypto)
