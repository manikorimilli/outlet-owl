# Architecture decisions: OutletOwl

One row per decision, and every contradiction settled once so it is not
settled again, differently, in each file that runs into it.

The technology key log is docs/decisions.md (kept by `tech-decision`); this file indexes each ADR and records the conflicts the HLD settled.

## Decisions

| Id | Title | Area | Status | Reversibility |
| --- | --- | --- | --- | --- |
| ADR-0001 | Use Go for the server, with a separate TypeScript UI | backend language | Accepted | awkward: rewriting the API, gateway, seed and their tests |
| ADR-0002 | Use React with Vite for the UI, as a single-page app | frontend | Accepted | awkward: rewriting every screen; the API is unaffected |
| ADR-0003 | Use sqlc, pgx and goose for data access and migrations | orm and data access | Accepted | cheap to awkward: queries and migrations are plain SQL, so another query layer keeps the SQL |
| ADR-0004 | Use GitHub Actions for CI, with no live model calls | ci and delivery | Accepted | cheap: only the workflow file is GitHub-specific |
| ADR-0005 | Use REST with an OpenAPI spec between the UI and the server | api style | Accepted | cheap: dropping the spec only removes generated types |
| ADR-0006 | Run tagging in the Go server, driven by untagged reviews | messaging | Accepted | cheap: a job table adds rows and a claim step |
| ADR-0007 | Use a signed JWT in an HttpOnly cookie for sign-in | auth | Accepted | cheap: changes the sign-in handler and request check only |
| ADR-0008 | Tag reviews and draft replies with prompt-only calls to Claude Haiku 4.5 | llm approach | Accepted | cheap: a new approach is a new prompt version and parser |
| ADR-0009 | Allow one configured model in place of Claude Haiku 4.5 | llm approach | Accepted | cheap: unset MODEL_ID |

## Conflicts that were settled

### How a batch of 20 reviews stays usable under the 1000-token output cap

**Between:** REQ-006 (batches of 20), REQ-032 (max_tokens at most 1000), AC-US-01-004-4 (retry only the missing IDs), ADR-0006 (one batch at a time).

**Decision.** The model answers one compact line per review (review ID, theme codes, sentiment, urgent flag). An answer cut off by the cap keeps every complete line; the missing tail is retried as a smaller batch under the Q-014 retry limit.

**Why.** A single JSON array cut off mid-way parses as nothing, so all 20 IDs would be "missing" and the retry would be cut off again. Line-per-review keeps both of the brief's numbers and turns a truncation into an ordinary missing-ID retry.

**Settled by:** the product owner, in session, 2026-10-06.

**What now has to change to match:**

- HLD section 3 (tagging run) and section 8 (numbers that must agree): state the line format; done in v1.
- The tagging prompt v1 (prompts/tagging, `prompt-registry`): ask for one line per review with short theme codes.
- `low-level-design` for the tagging run: the line parser, the per-line validation and the shrinking retry batch.
- No ADR changes; no PRD change.

### How CI checks that the digest reaches the mail catcher

**Between:** AC-US-01-009-5 (one email reaches the MailHog inbox), ADR-0004 (CI with a PostgreSQL service container), PRD section 6 (runtime limited to PostgreSQL and MailHog).

**Decision.** CI runs MailHog as a second service container, and the digest test reads MailHog's inbox, as the criterion says.

**Why.** The criterion is then tested as written; CI mirrors the local Docker setup; the PRD's "nothing else" limits the runtime, not CI.

**Settled by:** the product owner, in session, 2026-10-06.

**What now has to change to match:**

- ADR-0004 is not edited. Its decision (GitHub Actions with service containers) covers one more container; its consequences line names PostgreSQL only. If a reviewer reads ADR-0004 as limiting CI to PostgreSQL, a superseding ADR-0008 records the addition.
- The CI workflow, when written in phase 0, adds the MailHog service container.
- HLD sections 6, 7 and 12 name MailHog in CI; done in v1.

### How the tagging run ends, and what its lock protects

**Between:** ADR-0006 (decision items 4 and 6: "retried by the next run", "advisory lock allows one run at a time"), AC-US-01-004-5, AC-US-01-002-6, HLD section 3, critic findings MAJOR 2 and MINOR 7.

**Decision.** One long-lived tagging goroutine fed by a signal channel holding one pending signal. Each signal runs one pass over a snapshot of the untagged ids; ids that still fail after 2 retries wait for the next pass, which starts at the next import or restart. The advisory lock is held on a dedicated connection for the whole pass and only keeps the seed and eval commands from tagging at the same time. An operator switch TAGGING_ENABLED stops tagging without touching the gateway mode.

**Why.** Re-selecting "every untagged review" inside one run loops on a review that always fails and can spend the budget; a lock taken through a pooled connection is re-entrant and allowed two runs; a start signal sent while a run was ending was lost.

**Settled by:** the product owner, in session, 2026-10-06 (applied with the critic's fixes).

**What now has to change to match:**

- ADR-0006 is not edited: "the next run" and "one run at a time" still hold; this entry records the mechanics. If a reviewer reads ADR-0006 as requiring a run to drain every untagged review, a superseding ADR-0008 records the one-pass rule.
- HLD sections 2, 3, 7, 8 and 12; done in v1.
- `low-level-design` for the tagging worker.

### Whether a re-imported review is a new review

**Between:** AC-US-01-002-1 ("each valid row becomes one stored review"), AC-US-01-002-3 and Q-023 (fix and re-import), tenet 8, HLD section 4 (an earlier v1 draft accepted duplicates), critic finding MINOR 10.

**Decision.** A review is unique on outlet, source, date, reviewer name and a hash of its text. A row matching an existing review is counted as "already imported" and not stored. Outlet names are unique ignoring case, and the CSV's outlet column matches by that name. Each review records the import that created it.

**Why.** Q-023 tells the admin to fix the file and import again; without a natural key that doubles every valid row, doubling trends and movers and paying to tag the copies.

**Settled by:** the product owner, in session, 2026-10-06 (applied with the critic's fixes).

**What now has to change to match:**

- Backlog AC-US-01-002-1 gains "rows already imported are counted, not stored" (through `backlog`); listed in HLD section 17.
- HLD sections 2, 4, 5 and 7; done in v1.

