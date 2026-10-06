# System design tenets: OutletOwl

Rules this team holds itself to on this project. Each is here because
somebody could plausibly do the opposite, and a reviewer can point at a
breach.

## 1. One door to the model

**Every model call goes through the model gateway function; no other package builds an HTTP request to OpenRouter.**

The USD 8 stop, the 1000-token cap, the cost log and replay all live in the gateway (REQ-030 to REQ-032, ADR-0001). One call that goes around it spends money nobody counts and breaks replay in CI.

_A breach looks like:_ a merge request that adds a small OpenRouter client inside the reply handler "to get drafting working quickly", with its own model string and no cost-log row.

## 2. The budget lives in the database

**The running total is read from the model call log in PostgreSQL immediately before each live call, never from a value held in memory or in a cookie.**

The total must survive restarts and be shared by the tagging run, drafts and the commands (Q-015). A cached total lets a second process or a restart overspend.

_A breach looks like:_ a merge request that keeps `spentSoFar` in a package variable, loaded once at start, "to save a query per call".

## 3. Outlet scope is a query condition from the database

**Every query for outlet data carries the caller's outlet condition, taken from the user row re-read on this request, never from the JWT claims, a query parameter or the UI.**

Managers see only their own outlet (Q-002), and the token cannot be revoked for 8 hours (ADR-0007), so a claim in it can be stale.

_A breach looks like:_ a merge request whose review list handler reads `outlet_id` from the URL and passes it straight into the query, or reads the role from the token.

## 4. Results are matched by review ID, never by position

**Tag results are stored only against the review ID in the result line, after checking that ID belongs to the batch; the order of lines in the answer is ignored.**

Misaligned batch results are the brief's named biggest risk (REQ-012 to REQ-014).

_A breach looks like:_ a merge request that loops `for i, line := range lines { store(batch[i].ID, line) }`.

## 5. Tests never reach the network

**Tests and CI run the gateway in replay mode; a missing recording fails the test with its key, and nothing falls back to a live call.**

CI holds no key and makes no live calls (REQ-035, ADR-0004); a silent fallback would spend budget from a developer's machine without anyone choosing to.

_A breach looks like:_ a merge request where the replay layer, on a cache miss, "just calls the real API and saves it" when an environment variable happens to be set.

## 6. Untrusted text is only ever text

**Review text, reviewer names and model drafts are rendered as plain text in the UI and escaped in the digest email; no markup is built from them.**

Reviews come from outside and drafts come from a model (HLD section 9); both appear on the same origin as every staff session.

_A breach looks like:_ a merge request that uses `dangerouslySetInnerHTML` to keep line breaks in reviews, or marks review text as safe HTML in the digest template.

## 7. The OpenAPI spec is the contract

**A change to a request or response shape changes api/openapi.yaml in the same merge request, and the UI's API types are generated from it, never edited by hand.**

Go and TypeScript are two codebases (ADR-0001); the spec is what keeps them in step (ADR-0005).

_A breach looks like:_ a merge request that adds a field to the movers response in Go and types it by hand in the web app, with no change to the spec.

## 8. Writes with side effects are safe to repeat

**Every write endpoint with a side effect (import, draft, mark replied, digest) takes a client-made request id or is naturally idempotent, backed by a unique constraint, and a repeat returns the first result.**

Double clicks and retries would otherwise import a file twice, spend budget on a second draft or send a second digest.

_A breach looks like:_ a merge request for POST /api/digests that sends an email on every call, relying on the button being disabled after the first click.
