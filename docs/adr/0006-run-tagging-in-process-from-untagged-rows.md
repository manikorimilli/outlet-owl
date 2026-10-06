# ADR-0006: Run tagging in the Go server, driven by untagged reviews

- Status: Accepted
- Date: 2026-10-06
- Task: none (design before the first task)
- Deciders: the product owner, in session
- Area: messaging
- Reversibility: cheap: moving to a job table later adds rows and a claim step; the tagging code itself is unchanged

## Context

- Tagging starts automatically after each import (Q-022), sends reviews in batches of 20 (REQ-006), tags each review once (REQ-011) and retries missing or invalid results up to a limit (REQ-012 to REQ-014, Q-014).
- No message broker or other runtime service is allowed (PRD section 6, brief L80); the Go server is one long-running local process (ADR-0001).
- Volume: the seed holds 1,500 reviews (REQ-042), which is 75 batch calls. At an estimated 5 to 10 seconds per call (not measured), one batch at a time takes about 6 to 12 minutes.
- Every model call goes through the budgeted gateway (REQ-030 to REQ-032), so a duplicate run would also spend budget twice.

## What else was considered

| Option | Why not | Would suit |
| --- | --- | --- |
| Tagging run in the Go server over untagged rows (chosen) | Runs only while the server is up; one batch at a time makes the seed take minutes | One kind of background work whose pending state is already visible in the data |
| PostgreSQL job table with a worker in the Go server | Job rows, claims, stale-claim timeouts and cleanup for a single job type | Several kinds of background work, or parallel batches with per-job status |
| Tag synchronously inside the import request | The request blocks for minutes; a timeout or closed tab leaves work half done | Imports of a few dozen rows |
| A message broker (RabbitMQ, Redis) | A runtime service the PRD forbids | A product with no runtime limit |

## Decision

We will tag reviews in a run inside the Go server that takes every review with no stored tag result, sends them through the gateway in batches of 20, one batch at a time, and is started after each import and at server start, because the untagged rows are themselves the pending work, which needs no queue, survives restarts by construction and cannot be double-tagged when only one run may hold the lock. The product owner chose the recommended option.

How it meets the asynchronous-work checks:

1. Enqueue with the state change: inserting a review is the enqueue; it commits in the import transaction.
2. Where it runs: a goroutine in the long-running Go process.
3. A model call that times out: no result is stored, the review stays untagged and is retried; a duplicate paid call of about one cent may not reach the running total. Accepted by the product owner with this ADR.
4. Retry end: bounded retries within a run (Q-014); a review still failing stays untagged, is counted on screen, and is retried by the next run (next import or restart).
5. Rate: one batch in flight at a time.
6. Twice: a PostgreSQL advisory lock allows one run at a time, and one stored tag result per review means a repeated result writes nothing twice.

## Consequences

- No job table; "untagged" is derived from the reviews and their tag results.
- Seeding 1,500 reviews takes several minutes of model calls; the dashboard fills in as batches land.
- A running total that misses a timed-out call could under-count spend by about a cent per timeout; the USD 10 key cap absorbs it.
- Revisit if a second kind of background work appears (scheduled digests are a non-goal today) or if seeding must finish in under a minute.

## Commits us to

PostgreSQL advisory locks, Go goroutines
