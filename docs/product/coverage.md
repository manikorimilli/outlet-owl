# Coverage: PRD statements to stories

PRD: docs/product/PRD.md   Backlog: docs/product/backlog.md   Built: 2026-10-06

No application code exists yet, so no statement was checked against code; the repository holds only the README and docs/product.

## Matrix

| REQ | Statement (short) | Judgement | Why | Covered by | AC ids |
| --- | --- | --- | --- | --- | --- |
| REQ-001 | The system lets a brand admin add an outlet | story | A brand admin can do something new: add an outlet. | US-01-001 | AC-US-01-001-1, AC-US-01-001-2 |
| REQ-002 | The system lets a brand admin import reviews from a CSV file | story | Importing reviews is a capability the admin uses on its own. | US-01-002 | AC-US-01-002-1, AC-US-01-002-3, AC-US-01-002-6 |
| REQ-003 | The CSV import reads, for each review, the fields outlet, source,... | criterion-of US-01-002 | It names the fields the import reads, a condition on the import. | US-01-002 | AC-US-01-002-2, AC-US-01-002-3 |
| REQ-004 | The system imports reviews through a connector interface, with CSV... | criterion-of US-01-002 | It says how the import is built, not something a user does on its own. | US-01-002 | AC-US-01-002-4 |
| REQ-005 | The system defines a Google connector as an interface only, with... | criterion-of US-01-002 | An interface with no implementation gives the user nothing new; it is checked as part of the import design. | US-01-002 | AC-US-01-002-5 |
| REQ-006 | The system tags reviews by theme with the model, sending reviews... | story | Tagging is the new capability every dashboard view reads. | US-01-003 | AC-US-01-003-1, AC-US-01-003-2, AC-US-01-003-7 |
| REQ-007 | The system takes the themes from a configurable list | criterion-of US-01-003 | It limits which themes a tag can carry. | US-01-003 | AC-US-01-003-2 |
| REQ-008 | The initial theme list is food, wait time, staff, cleanliness and... | criterion-of US-01-003 | It sets the starting values of the theme list. | US-01-003 | AC-US-01-003-3 |
| REQ-009 | The system records a sentiment for each tagged review | criterion-of US-01-003 | Sentiment is one more field of the same tagging call. | US-01-003, US-02-004 | AC-US-01-003-4, AC-US-02-004-6 |
| REQ-010 | The system flags a review as urgent when it concerns food safety,... | criterion-of US-01-003 | The urgent flag is one more field of the same tagging call. | US-01-003 | AC-US-01-003-5 |
| REQ-011 | The system stores each successful tag result and never sends an... | criterion-of US-01-003 | Tagging once is a rule on the tagging run, not a user action. | US-01-003 | AC-US-01-003-6 |
| REQ-012 | The model's batch tagging output identifies each result by its... | criterion-of US-01-004 | The ID in the output is how alignment is guaranteed. | US-01-004 | AC-US-01-004-1 |
| REQ-013 | The system validates the batch tagging output per review ID | story | Per-ID validation is the requested fallback for the brief's biggest risk and ships as its own guard. | US-01-004 | AC-US-01-004-2, AC-US-01-004-3 |
| REQ-014 | After a batch, the system retries tagging only for the review IDs... | criterion-of US-01-004 | Retrying only missing IDs is the action taken after validation. | US-01-004 | AC-US-01-004-4, AC-US-01-004-5 |
| REQ-015 | The dashboard shows the rating trend over time for each outlet | story | A new dashboard view; the outlet manager's restricted view is US-00-001. | US-01-005, US-00-001 | AC-US-01-005-1, AC-US-00-001-1, AC-US-00-001-2 |
| REQ-016 | The dashboard shows the sentiment trend over time for each outlet | story | Shown beside the rating trend in the same comparison view. | US-01-005, US-00-001 | AC-US-01-005-2, AC-US-00-001-2 |
| REQ-017 | The dashboard shows a heatmap of themes by outlet | story | A separate dashboard view with its own counting rule. | US-01-006, US-00-001 | AC-US-01-006-1, AC-US-01-006-2, AC-US-00-001-2 |
| REQ-018 | The dashboard shows the biggest movers week over week | story | Movers are their own calculation, reused by the digest. | US-01-007, US-00-001 | AC-US-01-007-1, AC-US-01-007-2, AC-US-01-007-3, AC-US-00-001-2 |
| REQ-019 | The dashboard includes a review list that can be searched | story | Search is a new capability on the review list. | US-01-008, US-00-001 | AC-US-01-008-1, AC-US-01-008-2, AC-US-00-001-3, AC-US-00-001-4 |
| REQ-020 | The system drafts a reply to each review with the model, in the... | story | Drafting a reply is something new the manager gets. | US-00-002 | AC-US-00-002-1, AC-US-00-002-2, AC-US-00-002-3, AC-US-00-002-7 |
| REQ-021 | The drafted reply is written in the review's language, English or... | criterion-of US-00-002 | Language is a condition on the draft. | US-00-002 | AC-US-00-002-4, AC-US-00-002-5 |
| REQ-022 | The system lets an outlet manager edit a draft reply | story | Editing the draft is a manager action. | US-00-003 | AC-US-00-003-1 |
| REQ-023 | The system lets an outlet manager mark a reply as replied | story | Marking replied is a manager action, done in the same screen as editing. | US-00-003 | AC-US-00-003-2 |
| REQ-024 | A reply is not marked replied until an outlet manager has approved it | criterion-of US-00-003 | Approval is a rule on marking replied. | US-00-003 | AC-US-00-003-2, AC-US-00-003-3, AC-US-00-003-4 |
| REQ-025 | The system produces a weekly digest that lists the biggest movers | story | The digest is a new output for the brand admin. | US-01-009 | AC-US-01-009-2 |
| REQ-026 | The weekly digest lists the urgent reviews | criterion-of US-01-009 | A section of the digest. | US-01-009 | AC-US-01-009-4 |
| REQ-027 | The weekly digest names the outlet and the issue that moved most | criterion-of US-01-009 | The headline line of the digest. | US-01-009 | AC-US-01-009-3, AC-US-01-009-6 |
| REQ-028 | The system generates the digest on demand | story | On-demand generation is how the admin gets the digest; merged with REQ-025 into one story. | US-01-009 | AC-US-01-009-1 |
| REQ-029 | The system sends the digest by email to the local mail catcher | criterion-of US-01-009 | The delivery route of the digest. | US-01-009 | AC-US-01-009-5 |
| REQ-030 | The system records the tokens used and the cost of every model call | criterion-of US-02-001 | Logging is a duty of the gateway. | US-02-001 | AC-US-02-001-1, AC-US-02-001-2, AC-US-02-001-7 |
| REQ-031 | The system refuses any model call once the running total cost of... | story | The budget stop is the gateway's reason to exist and ships with it. | US-02-001 | AC-US-02-001-4, AC-US-02-001-5, AC-US-02-001-6 |
| REQ-032 | The system sets max_tokens to at most 1000 on every model call | criterion-of US-02-001 | A limit on every request the gateway builds. | US-02-001 | AC-US-02-001-3 |
| REQ-033 | The system versions the prompts it sends to the model | story | Prompt versioning is a capability the developer uses on its own. | US-02-002 | AC-US-02-002-1, AC-US-02-002-2, AC-US-02-002-3 |
| REQ-034 | The automated tests replay recorded model responses instead of... | story | Replay testing is a capability the developer uses on every change. | US-02-003 | AC-US-02-003-1, AC-US-02-003-2 |
| REQ-035 | The CI pipeline makes no live model calls | criterion-of US-02-003 | No live calls in CI is the condition replay must meet. | US-02-003 | AC-US-02-003-3 |
| REQ-036 | The project includes an evaluation that measures theme accuracy on... | story | The classification evaluation is its own deliverable. | US-02-004 | AC-US-02-004-1, AC-US-02-004-5 |
| REQ-037 | The evaluation measures urgent-flag accuracy on the same 100... | story | Measured in the same evaluation run as REQ-036. | US-02-004 | AC-US-02-004-2, AC-US-02-004-5 |
| REQ-038 | The classification evaluation passes or fails on urgent recall | criterion-of US-02-004 | The gate is a rule on the evaluation result; its pass mark is 90% urgent recall (Q-017, given 2026-10-06). | US-02-004 | AC-US-02-004-3, AC-US-02-004-4 |
| REQ-039 | The project includes a tone check of 30 reply drafts | story | The tone check is its own deliverable. | US-02-005 | AC-US-02-005-1, AC-US-02-005-2, AC-US-02-005-3 |
| REQ-040 | The repository includes a seed script that creates the demo data | story | The seed script is what the developer runs. | US-02-006 | AC-US-02-006-1, AC-US-02-006-5 |
| REQ-041 | The seed script creates 5 outlets | criterion-of US-02-006 | A count the seed must produce. | US-02-006 | AC-US-02-006-2 |
| REQ-042 | The seed script creates 1,500 reviews spread across 6 months | criterion-of US-02-006 | A count and date range the seed must produce. | US-02-006 | AC-US-02-006-3 |
| REQ-043 | The seed data includes a planted wait-time spike at one outlet | criterion-of US-02-006 | A pattern the seed must contain. | US-02-006 | AC-US-02-006-4 |
| REQ-044 | The interface presents a clear information hierarchy, readable... | non-functional | A visual quality every dashboard view carries as a criterion. | US-01-005, US-01-006, US-01-007, US-01-008 | AC-US-01-005-4, AC-US-01-006-3, AC-US-01-007-4, AC-US-01-008-5 |
| REQ-045 | The interface makes outlet comparisons easy to find | criterion-of US-01-005 | Where the outlet comparison is reached. | US-01-005 | AC-US-01-005-3 |
| REQ-046 | The interface makes urgent reviews easy to find | criterion-of US-01-008 | Where urgent reviews are reached. | US-01-008 | AC-US-01-008-3 |
| REQ-047 | The interface makes reply actions easy to find | criterion-of US-00-003 | Where the reply actions sit. | US-00-003 | AC-US-00-003-5 |
| REQ-048 | The interface displays English and Hindi text legibly | non-functional | Legible Hindi applies to every screen that shows review or reply text. | US-01-008, US-00-002 | AC-US-01-008-4, AC-US-00-002-6 |
| REQ-049 | The system applies each tenant's own colours to the interface | story | Stretch only: EP-08, not in the MVP. | US-01-010 | AC-US-01-010-1, AC-US-01-010-2 |

## Gaps

| REQ | Why uncovered | Proposed action |
| --- | --- | --- |
| none | Every live REQ has at least one acceptance criterion. | none |

Not a REQ gap, but worth your decision: sign-in is not a PRD statement. It is carried as an `inferred:` criterion in US-00-001 (AC-US-00-001-1) because roles need it; Q-003 (confirmed 2026-10-06) says users sign in with seeded or configured accounts. If you want it as a requirement, `prd` adds it as a new REQ.

## Orphan stories

| Story | Reason it exists | Action |
| --- | --- | --- |
| none | Every story covers at least one REQ. | none |

Inferred persona: "Developer (group 02 operator)" carries US-02-001 to US-02-006. The PRD names only the brand admin and outlet manager; the gateway, prompt, test, evaluation and seed statements have no user role. Accept it or name another role.

## Counts

stories-coverage: 49 REQ from docs/product/PRD.md (0 withdrawn), 49 covered, 0 out of scope, 0 gaps, 19 stories, 87 AC, 0 orphans, 0 problems
Verdict: covered
