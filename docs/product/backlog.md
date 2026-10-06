# Backlog: Review Intelligence for Multi-Outlet Businesses

PRD: docs/product/PRD.md   Questions: docs/product/questions.md   Built: 2026-10-06

Status of decisions: all 23 questions in docs/product/questions.md were answered by you in session on 2026-10-06, and each criterion that rests on one cites it as confirmed. No story waits on a question. Still yours to give, not needed to build: the evaluation pass marks (Q-017, Q-018, Q-019). Criteria marked `inferred:` (sign-in in US-00-001, the spike placement note in US-02-006) are not PRD statements.

Scope of this file: stories and acceptance criteria only. Tasks, user flows and a tracker import are not written (not requested). Points read TBD until `estimate` runs.

MVP: EP-01 to EP-07. Stretch, not in the MVP: EP-08.

## Story index

| Story | Epic | Title | Persona | Priority | Points | Covers | Depends on | Resolve before build |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-01-001 | EP-01 | Add an outlet | Brand admin | Must | TBD | REQ-001 | none | none |
| US-01-002 | EP-01 | Import reviews from a CSV file through a connector | Brand admin | Must | TBD | REQ-002, REQ-003, REQ-004, REQ-005 | US-01-001 | none |
| US-01-003 | EP-02 | Tag reviews by theme, sentiment and urgency | Brand admin | Must | TBD | REQ-006, REQ-007, REQ-008, REQ-009, REQ-010, REQ-011 | US-01-002, US-02-001 | none |
| US-01-004 | EP-02 | Keep batch tags matched to the right review | Brand admin | Must | TBD | REQ-012, REQ-013, REQ-014 | US-01-003 | none |
| US-01-005 | EP-03 | Compare rating and sentiment trends across outlets | Brand admin | Must | TBD | REQ-015, REQ-016, REQ-044, REQ-045 | US-01-003 | none |
| US-01-006 | EP-03 | See a theme heatmap by outlet | Brand admin | Must | TBD | REQ-017, REQ-044 | US-01-003 | none |
| US-01-007 | EP-03 | See the biggest movers week over week | Brand admin | Must | TBD | REQ-018, REQ-044 | US-01-003 | none |
| US-01-008 | EP-03 | Search the review list and find urgent reviews | Brand admin | Must | TBD | REQ-019, REQ-044, REQ-046, REQ-048 | US-01-003 | none |
| US-00-001 | EP-03 | Sign in and see my outlet's dashboard and reviews | Outlet manager | Must | TBD | REQ-015, REQ-016, REQ-017, REQ-018, REQ-019 | US-01-005, US-01-006, US-01-007, US-01-008 | none |
| US-00-002 | EP-04 | Get a drafted reply in the brand's tone and the review's language | Outlet manager | Must | TBD | REQ-020, REQ-021, REQ-048 | US-00-001, US-02-001, US-02-002 | none |
| US-00-003 | EP-04 | Edit, approve and mark a reply as replied | Outlet manager | Must | TBD | REQ-022, REQ-023, REQ-024, REQ-047 | US-00-002 | none |
| US-01-009 | EP-05 | Generate the weekly digest and send it to the mail catcher | Brand admin | Must | TBD | REQ-025, REQ-026, REQ-027, REQ-028, REQ-029 | US-01-007 | none |
| US-02-001 | EP-06 | Route every model call through one budgeted gateway | Developer | Must | TBD | REQ-030, REQ-031, REQ-032 | none | none |
| US-02-002 | EP-06 | Keep model prompts under version | Developer | Must | TBD | REQ-033 | US-02-001 | none |
| US-02-003 | EP-06 | Run tests on recorded model responses | Developer | Must | TBD | REQ-034, REQ-035 | US-02-001 | none |
| US-02-004 | EP-06 | Evaluate theme and urgent-flag accuracy on 100 labelled reviews | Developer | Must | TBD | REQ-009, REQ-036, REQ-037, REQ-038 | US-01-003, US-02-003 | none |
| US-02-005 | EP-06 | Check the tone of 30 reply drafts | Developer | Must | TBD | REQ-039 | US-00-002 | none |
| US-02-006 | EP-07 | Seed the demo with 5 outlets and 1,500 reviews | Developer | Must | TBD | REQ-040, REQ-041, REQ-042, REQ-043 | US-01-001 | none |
| US-01-010 | EP-08 | Show each tenant's own colours (stretch) | Brand admin | Could | TBD | REQ-049 | US-01-005 | none |

## Hours by discipline

tasks: not written (stories only)

## EP-01 Set up outlets and bring in reviews

Goal: a brand admin has every outlet in the system and its reviews imported from a CSV file.
Covers: REQ-001, REQ-002, REQ-003, REQ-004, REQ-005

### US-01-001 Add an outlet

Epic: EP-01   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-001   Judgement: story

**Narrative.** As a brand admin, I want to add an outlet, so that its reviews can be imported and compared with the other outlets.

**Why it matters.** B1: an outlet that is not in the system cannot appear in any comparison or digest; every other story starts here.

**From the PRD.**
- REQ-001: "The system lets a brand admin add an outlet."

**Preconditions.**
- The user is signed in as the brand admin, with an account created by the seed or configuration (Q-003, confirmed 2026-10-06).

**Acceptance criteria.**

- AC-US-01-001-1. Given a signed-in brand admin, when they add an outlet with a name, then the outlet appears in the outlet list and can be chosen as the outlet of imported reviews.
  Covers: REQ-001
- AC-US-01-001-2. Given a signed-in outlet manager, when they try to add an outlet, then the system refuses the action, both on screen and at the service.
  Covers: REQ-001

**Not in this story.**
- Creating or assigning outlet manager accounts; they come from the seed or configuration (Q-003, confirmed 2026-10-06).
- Several brands in one installation (Q-001, confirmed 2026-10-06: one brand per installation).

**Depends on.**
- none

**Assumptions.**
- One brand per installation, so outlets belong to that brand (Q-001, confirmed 2026-10-06).
- Outlet managers are assigned to outlets by the seed or configuration, not in this screen; a newly added outlet has no manager until the configuration assigns one (Q-003, confirmed 2026-10-06).

**Resolve before build.** none (Q-001, Q-003 confirmed 2026-10-06)

### US-01-002 Import reviews from a CSV file through a connector

Epic: EP-01   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-002, REQ-003, REQ-004, REQ-005   Judgement: merged from REQ-002 (story), REQ-003, REQ-004 and REQ-005 (criteria of it)

**Narrative.** As a brand admin, I want to import reviews from a CSV file, so that every outlet's reviews are in one place for tagging and replies.

**Why it matters.** B3: sentiment and themes can only be tracked for reviews the system holds; CSV is the only source in scope.

**From the PRD.**
- REQ-002: "The system lets a brand admin import reviews from a CSV file."
- REQ-003: "The CSV import reads, for each review, the fields outlet, source, date, rating, text and reviewer name."
- REQ-004: "The system imports reviews through a connector interface, with CSV as one connector, so that other review sources can be added later as further connectors."
- REQ-005: "The system defines a Google connector as an interface only, with no working implementation behind it."

**Preconditions.**
- At least one outlet exists (US-01-001).

**Acceptance criteria.**

- AC-US-01-002-1. Given a CSV file whose rows name existing outlets, when the brand admin imports it, then each valid row becomes one stored review and the admin sees how many reviews were imported.
  Covers: REQ-002
- AC-US-01-002-2. Given a CSV row with outlet, source, date, rating, text and reviewer name, when it is imported, then the stored review holds all six values unchanged.
  Covers: REQ-003
- AC-US-01-002-3. Given a CSV file where some rows name an unknown outlet or miss a required field, when it is imported, then the valid rows are imported and each rejected row is reported with its row number and reason (Q-023, confirmed 2026-10-06).
  Covers: REQ-002, REQ-003
- AC-US-01-002-4. Given the import code, when the CSV import runs, then it runs through the connector interface, and a second connector can be added by implementing that interface without changing the import flow.
  Covers: REQ-004
- AC-US-01-002-5. Given the Google connector, when the code base is inspected or the connector is invoked, then it exists as an interface only, fetches nothing and makes no network call.
  Covers: REQ-005
- AC-US-01-002-6. Given a completed import, when it finishes, then the new reviews are queued for tagging (US-01-003) without a separate admin action (Q-022, confirmed 2026-10-06).
  Covers: REQ-002

**Not in this story.**
- Live connectors that fetch reviews (PRD non-goal).
- A working Google connector (PRD non-goal; REQ-005 keeps the interface only).
- Tagging itself (US-01-003).

**Depends on.**
- US-01-001: imported rows must name an outlet that exists.

**Assumptions.**
- Rows with errors are skipped and reported, valid rows are kept (Q-023, confirmed 2026-10-06).
- Tagging starts automatically after an import (Q-022, confirmed 2026-10-06).

**Resolve before build.** none (Q-022, Q-023 confirmed 2026-10-06)

## EP-02 Understand every review automatically

Goal: each imported review carries its themes, sentiment and urgent flag, tagged once and matched to the right review.
Covers: REQ-006, REQ-007, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012, REQ-013, REQ-014

### US-01-003 Tag reviews by theme, sentiment and urgency

Epic: EP-02   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-006, REQ-007, REQ-008, REQ-009, REQ-010, REQ-011   Judgement: merged from REQ-006 (story) and REQ-007 to REQ-011 (criteria of it)

**Narrative.** As a brand admin, I want every imported review tagged by theme, sentiment and urgency, so that I can see what customers talk about at each outlet and catch urgent reviews.

**Why it matters.** B3: the trends, heatmap, movers and digest all read these tags; without them no outlet can be compared. B1 depends on it too.

**From the PRD.**
- REQ-006: "The system tags reviews by theme with the model, sending reviews to the model in batches of 20."
- REQ-007: "The system takes the themes from a configurable list."
- REQ-008: "The initial theme list is food, wait time, staff, cleanliness and price."
- REQ-009: "The system records a sentiment for each tagged review."
- REQ-010: "The system flags a review as urgent when it concerns food safety, harassment or a legal threat."
- REQ-011: "The system stores each successful tag result and never sends an already tagged review to the model for tagging again."

**Preconditions.**
- Untagged reviews exist (US-01-002); model calls go through the gateway (US-02-001).

**Acceptance criteria.**

- AC-US-01-003-1. Given 45 untagged reviews, when tagging runs, then the model receives 3 requests holding 20, 20 and 5 reviews, each through the gateway.
  Covers: REQ-006
- AC-US-01-003-2. Given a theme list in configuration, when a review is tagged, then it carries zero, one or several themes, all from that list (developer-edited list: Q-012, confirmed 2026-10-06; several themes per review: Q-021, confirmed 2026-10-06).
  Covers: REQ-007, REQ-006
- AC-US-01-003-3. Given a fresh installation, when the theme list is read, then it holds exactly food, wait time, staff, cleanliness and price.
  Covers: REQ-008
- AC-US-01-003-4. Given a tagged review, when it is stored, then it carries one sentiment label: positive, neutral or negative (Q-013, confirmed 2026-10-06).
  Covers: REQ-009
- AC-US-01-003-5. Given recorded reviews that mention food safety, harassment or a legal threat, when they are tagged, then each is flagged urgent with each reason it shows (food_safety, harassment, legal_threat; several allowed, none repeated), and a review mentioning none of these is not flagged and has no reason (product owner, 2026-10-06).
  Covers: REQ-010
- AC-US-01-003-6. Given a review already tagged, when tagging runs again, then that review is not sent to the model and its stored tags, with the prompt version that produced them, are unchanged (Q-011, confirmed 2026-10-06).
  Covers: REQ-011
- AC-US-01-003-7. Given a review in Hindi (Devanagari or Latin script), when it is tagged, then it receives themes, sentiment and urgent flag like an English review (Q-009, confirmed 2026-10-06).
  Covers: REQ-006

**Not in this story.**
- Recovering misaligned or missing batch results (US-01-004).
- Per-brand themes (PRD non-goal).
- An admin screen to edit the theme list (Q-012, confirmed 2026-10-06: developer-edited configuration).
- Re-tagging reviews when the tagging prompt changes (Q-011, confirmed 2026-10-06: no re-tagging) or the theme list changes (Q-012, confirmed 2026-10-06: existing tags are kept).

**Depends on.**
- US-01-002: provides the reviews.
- US-02-001: every model call goes through the gateway.

**Assumptions.**
- Zero, one or several themes per review (Q-021, confirmed 2026-10-06).
- Sentiment is one label per review (Q-013, confirmed 2026-10-06).
- The theme list is one developer-edited configuration; a new theme applies only to reviews tagged after the change (Q-012, confirmed 2026-10-06).
- Hindi reviews, in Devanagari or Latin script, are tagged too (Q-009, confirmed 2026-10-06).
- Tagging starts automatically after each import (Q-022, confirmed 2026-10-06).

**Resolve before build.** none (Q-009, Q-011 to Q-013, Q-021, Q-022 confirmed 2026-10-06)

### US-01-004 Keep batch tags matched to the right review

Epic: EP-02   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-012, REQ-013, REQ-014   Judgement: merged from REQ-013 (story) and REQ-012, REQ-014 (criteria of it)

**Narrative.** As a brand admin, I want every tag stored against the review it belongs to, so that a misaligned batch never puts one review's urgent flag on another.

**Why it matters.** B3: the brief names misaligned batch results as the biggest risk; a wrong tag corrupts every trend and the urgent list.

**From the PRD.**
- REQ-012: "The model's batch tagging output identifies each result by its review ID."
- REQ-013: "The system validates the batch tagging output per review ID."
- REQ-014: "After a batch, the system retries tagging only for the review IDs missing from the output."

**Preconditions.**
- Tagging runs in batches (US-01-003).

**Acceptance criteria.**

- AC-US-01-004-1. Given a batch of 20 reviews, when the model answers, then every result in the output carries a review ID from that batch, and results are stored by that ID, never by position.
  Covers: REQ-012
- AC-US-01-004-2. Given a recorded output with a result for an ID not in the batch, when it is validated, then that result is discarded and nothing is stored for it.
  Covers: REQ-013
- AC-US-01-004-3. Given a recorded output with a result naming a theme not in the list or missing its sentiment or urgent flag, when it is validated, then that result is discarded and its review ID is treated as missing (Q-014, confirmed 2026-10-06).
  Covers: REQ-013
- AC-US-01-004-4. Given a recorded output that omits 3 of the 20 IDs, when the batch is processed, then the 17 valid results are stored and the retry request contains exactly the 3 missing IDs.
  Covers: REQ-014
- AC-US-01-004-5. Given a review that is still missing after the retry limit set in design, when tagging ends, then the review stays untagged and is picked up by the next tagging run (Q-014, confirmed 2026-10-06).
  Covers: REQ-014

**Not in this story.**
- The retry limit value (set in design; Q-014 confirmed only that a limit exists).

**Depends on.**
- US-01-003: provides the batch tagging this story guards.

**Assumptions.**
- A present but invalid result counts as missing and is retried with the missing IDs (Q-014, confirmed 2026-10-06).

**Resolve before build.** none (Q-014 confirmed 2026-10-06)

## EP-03 Compare outlets and spot what changed

Goal: a brand admin compares outlets over time and finds the outlet and theme that changed most; an outlet manager sees the same views for their own outlet.
Covers: REQ-015, REQ-016, REQ-017, REQ-018, REQ-019, REQ-044, REQ-045, REQ-046, REQ-048

### US-01-005 Compare rating and sentiment trends across outlets

Epic: EP-03   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-015, REQ-016, REQ-044, REQ-045   Judgement: merged from REQ-015, REQ-016 (story), REQ-045 (criterion) and REQ-044 (non-functional)

**Narrative.** As a brand admin, I want rating and sentiment trends for each outlet side by side, so that I can see which outlet is slipping.

**Why it matters.** B3: this is where sentiment per outlet over time becomes visible. B1 builds on it.

**From the PRD.**
- REQ-015: "The dashboard shows the rating trend over time for each outlet."
- REQ-016: "The dashboard shows the sentiment trend over time for each outlet."
- REQ-044: "The interface presents a clear information hierarchy, readable charts and tables, consistent typography and spacing, and restrained colours."
- REQ-045: "The interface makes outlet comparisons easy to find."

**Preconditions.**
- Tagged reviews exist (US-01-003); the user is signed in as the brand admin.

**Acceptance criteria.**

- AC-US-01-005-1. Given reviews across several weeks, when the brand admin opens the dashboard, then each outlet shows its average rating per week as a trend line.
  Covers: REQ-015
- AC-US-01-005-2. Given tagged reviews across several weeks, when the brand admin opens the dashboard, then each outlet shows its sentiment per week as a trend line, built from the positive, neutral and negative labels (Q-013, confirmed 2026-10-06).
  Covers: REQ-016
- AC-US-01-005-3. Given the brand admin on any screen, when they look for the outlet comparison, then it is reachable in one step from the main navigation (exact acceptance set in design).
  Covers: REQ-045
- AC-US-01-005-4. Given the trend charts, when they are reviewed against the design system, then they follow its hierarchy, type scale, spacing and colour tokens with no animation of the charts themselves; only the shared load fade-in and hover lift apply, and both are off under reduced motion (acceptance set in design; motion decided by the product owner on 2026-10-06).
  Covers: REQ-044

**Not in this story.**
- Tenant colours (US-01-010, stretch).
- Manager-only view of one outlet (US-00-001).

**Depends on.**
- US-01-003: sentiment comes from the tags.

**Assumptions.**
- The brand admin sees all outlets (Q-002, confirmed 2026-10-06).
- Sentiment is one label per review (Q-013, confirmed 2026-10-06); the chart form (weekly share or average) is set in design.

**Resolve before build.** none (Q-002, Q-013 confirmed 2026-10-06)

### US-01-006 See a theme heatmap by outlet

Epic: EP-03   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-017, REQ-044   Judgement: story (REQ-017), carrying REQ-044 (non-functional)

**Narrative.** As a brand admin, I want a heatmap of themes by outlet, so that I can see at a glance which outlet has trouble with which theme.

**Why it matters.** B1: the heatmap points at the outlet and the issue before the weekly movers confirm it.

**From the PRD.**
- REQ-017: "The dashboard shows a heatmap of themes by outlet."
- REQ-044: "The interface presents a clear information hierarchy, readable charts and tables, consistent typography and spacing, and restrained colours."

**Preconditions.**
- Tagged reviews exist (US-01-003).

**Acceptance criteria.**

- AC-US-01-006-1. Given tagged reviews for 5 outlets and 5 themes, when the brand admin opens the heatmap, then it shows one row per outlet and one column per theme in the configured list.
  Covers: REQ-017
- AC-US-01-006-2. Given a review tagged with two themes, when the heatmap counts it, then it counts once under each theme (Q-021, confirmed 2026-10-06); the cell measure is set in design.
  Covers: REQ-017
- AC-US-01-006-3. Given the heatmap, when it is reviewed against the design system, then its colour scale is restrained, labelled and readable without colour alone (acceptance set in design).
  Covers: REQ-044

**Not in this story.**
- Per-theme sentiment in each cell (Q-013, confirmed 2026-10-06: one sentiment per review).

**Depends on.**
- US-01-003: provides the themes.

**Assumptions.**
- Several themes per review (Q-021, confirmed 2026-10-06).
- One sentiment per review (Q-013, confirmed 2026-10-06).

**Resolve before build.** none (Q-013, Q-021 confirmed 2026-10-06)

### US-01-007 See the biggest movers week over week

Epic: EP-03   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-018, REQ-044   Judgement: story (REQ-018), carrying REQ-044 (non-functional)

**Narrative.** As a brand admin, I want to see what moved most this week compared with last week, so that I act on the outlet and issue that changed.

**Why it matters.** B1: this is the "outlet and issue that moved most" the brief is built around.

**From the PRD.**
- REQ-018: "The dashboard shows the biggest movers week over week."
- REQ-044: "The interface presents a clear information hierarchy, readable charts and tables, consistent typography and spacing, and restrained colours."

**Preconditions.**
- At least two complete weeks of tagged reviews exist.

**Acceptance criteria.**

- AC-US-01-007-1. Given tagged reviews for the latest complete week and the week before, when the brand admin opens the movers view, then it lists outlet and theme pairs ranked by the change in the number of negative reviews tagged with that theme between the two weeks; weeks run Monday to Sunday (Q-004, Q-005, confirmed 2026-10-06).
  Covers: REQ-018
- AC-US-01-007-2. Given data where the planted wait-time spike falls in the latest complete week, when the movers are computed, then that outlet and wait time are ranked first.
  Covers: REQ-018
- AC-US-01-007-3. Given each mover, when it is shown, then it displays both weeks' counts and the change, so the ranking can be checked by hand.
  Covers: REQ-018
- AC-US-01-007-4. Given the movers view, when it is reviewed against the design system, then the top mover is the most prominent item on the screen (acceptance set in design).
  Covers: REQ-044

**Not in this story.**
- The weekly digest email (US-01-009).

**Depends on.**
- US-01-003: movers are computed from tags.

**Assumptions.**
- A mover is an outlet and theme pair measured by the absolute change in the number of negative reviews for that theme; "issue" means theme (Q-004, confirmed 2026-10-06).
- Weeks run Monday to Sunday and the latest complete week is compared with the week before (Q-005, confirmed 2026-10-06).

**Resolve before build.** none (Q-004, Q-005 confirmed 2026-10-06)

### US-01-008 Search the review list and find urgent reviews

Epic: EP-03   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-019, REQ-044, REQ-046, REQ-048   Judgement: story (REQ-019), with REQ-046 (criterion), REQ-044 and REQ-048 (non-functional)

**Narrative.** As a brand admin, I want to search the reviews and go straight to the urgent ones, so that a food-safety or legal complaint never waits.

**Why it matters.** B2: urgent reviews need a reply first; B3: search lets the admin read what sits behind a trend.

**From the PRD.**
- REQ-019: "The dashboard includes a review list that can be searched."
- REQ-044: "The interface presents a clear information hierarchy, readable charts and tables, consistent typography and spacing, and restrained colours."
- REQ-046: "The interface makes urgent reviews easy to find."
- REQ-048: "The interface displays English and Hindi text legibly."

**Preconditions.**
- Tagged reviews exist (US-01-003).

**Acceptance criteria.**

- AC-US-01-008-1. Given reviews whose text contains "wait", when the brand admin searches for "wait", then the list shows only reviews whose text or reviewer name contains it.
  Covers: REQ-019
- AC-US-01-008-2. Given the review list, when the brand admin filters by outlet, theme, sentiment, urgent or reply status, then only matching reviews are shown (Q-020, confirmed 2026-10-06).
  Covers: REQ-019
- AC-US-01-008-3. Given urgent reviews exist, when the brand admin opens the dashboard, then the number of urgent reviews is visible and leads to the list filtered to urgent in one step.
  Covers: REQ-046
- AC-US-01-008-4. Given a review written in Hindi (Devanagari), when it is shown in the list, then it renders with a font that supports Devanagari, with no missing glyphs; Hinglish in Latin script renders like English text (Q-009, confirmed 2026-10-06).
  Covers: REQ-048
- AC-US-01-008-5. Given the review list, when it is reviewed against the design system, then it follows the table and typography rules (acceptance set in design).
  Covers: REQ-044

**Not in this story.**
- Replying from the list (US-00-002, US-00-003).

**Depends on.**
- US-01-003: filters on themes, sentiment and urgent read the tags.

**Assumptions.**
- Search covers review text and reviewer name, with filters for outlet, theme, sentiment, urgent and reply status (Q-020, confirmed 2026-10-06).

**Resolve before build.** none (Q-009, Q-020 confirmed 2026-10-06)

### US-00-001 Sign in and see my outlet's dashboard and reviews

Epic: EP-03   Priority: Must   Points: TBD (estimate)
Persona: Outlet manager, group 00   Ticket: unassigned
Covers: REQ-015, REQ-016, REQ-017, REQ-018, REQ-019   Judgement: the outlet manager's view of the stories that carry REQ-015 to REQ-019; the sign-in part is inferred: from Q-003 (confirmed 2026-10-06), not a PRD statement

**Narrative.** As an outlet manager, I want to sign in and see my outlet's trends and reviews, so that I know what my customers say without seeing other outlets.

**Why it matters.** B2: the manager needs their reviews in front of them to reply; B3: they see their own outlet's trend.

**From the PRD.**
- REQ-015: "The dashboard shows the rating trend over time for each outlet."
- REQ-016: "The dashboard shows the sentiment trend over time for each outlet."
- REQ-017: "The dashboard shows a heatmap of themes by outlet."
- REQ-018: "The dashboard shows the biggest movers week over week."
- REQ-019: "The dashboard includes a review list that can be searched."

**Preconditions.**
- A manager account assigned to an outlet exists, created by the seed or configuration (Q-003, confirmed 2026-10-06).

**Acceptance criteria.**

- AC-US-00-001-1. Given an outlet manager account, when the manager signs in, then they reach their outlet's dashboard, and a wrong password is refused (inferred: sign-in is not a PRD statement; it follows from Q-003, confirmed 2026-10-06).
  Covers: REQ-015
- AC-US-00-001-2. Given a signed-in outlet manager, when they open the dashboard, then the rating trend, sentiment trend, heatmap row and movers shown are for their assigned outlet only (Q-002, confirmed 2026-10-06).
  Covers: REQ-015, REQ-016, REQ-017, REQ-018
- AC-US-00-001-3. Given a signed-in outlet manager, when they search the review list, then results come only from their assigned outlet (Q-002, Q-020 confirmed 2026-10-06).
  Covers: REQ-019
- AC-US-00-001-4. Given a signed-in outlet manager, when they request another outlet's reviews directly at the service, then the service refuses the request.
  Covers: REQ-019

**Not in this story.**
- User management screens (none: Q-003, confirmed 2026-10-06).
- Comparing several outlets (US-01-005, brand admin).

**Depends on.**
- US-01-005, US-01-006, US-01-007, US-01-008: the views this story restricts to one outlet.

**Assumptions.**
- One outlet per manager, managers see only their outlet (Q-002, confirmed 2026-10-06).
- Accounts come from the seed or configuration and users sign in with them (Q-003, confirmed 2026-10-06).

**Resolve before build.** none (Q-002, Q-003, Q-020 confirmed 2026-10-06)

## EP-04 Reply to every review

Goal: an outlet manager replies to every review with an on-brand draft they edit and approve; nothing is posted automatically.
Covers: REQ-020, REQ-021, REQ-022, REQ-023, REQ-024, REQ-047, REQ-048

### US-00-002 Get a drafted reply in the brand's tone and the review's language

Epic: EP-04   Priority: Must   Points: TBD (estimate)
Persona: Outlet manager, group 00   Ticket: unassigned
Covers: REQ-020, REQ-021, REQ-048   Judgement: story (REQ-020), with REQ-021 (criterion) and REQ-048 (non-functional)

**Narrative.** As an outlet manager, I want a reply drafted for each review in the brand's tone and the review's language, so that I can answer every review without writing from scratch.

**Why it matters.** B2: drafting is what lets a manager reply to every review without a dedicated person.

**From the PRD.**
- REQ-020: "The system drafts a reply to each review with the model, in the brand's tone."
- REQ-021: "The drafted reply is written in the review's language, English or Hindi."
- REQ-048: "The interface displays English and Hindi text legibly."

**Preconditions.**
- The manager is signed in (US-00-001); the reply prompt exists (US-02-002).

**Acceptance criteria.**

- AC-US-00-002-1. Given a review with no draft, when the outlet manager opens it, then a draft reply is generated through the gateway, stored, and shown (Q-007, confirmed 2026-10-06).
  Covers: REQ-020
- AC-US-00-002-2. Given a review that already has a stored draft, when the manager opens it again, then the stored draft is shown and no model call is made.
  Covers: REQ-020
- AC-US-00-002-3. Given the recorded response for an English review, when the draft is produced, then the reply prompt used is the current versioned reply prompt carrying the brand's tone, and its version is stored with the draft (Q-010, Q-011 confirmed 2026-10-06).
  Covers: REQ-020
- AC-US-00-002-4. Given a review in Devanagari Hindi, when the draft is produced, then the draft is in Devanagari Hindi; given an English review, the draft is in English.
  Covers: REQ-021
- AC-US-00-002-5. Given a review in Hindi written in Latin script (Hinglish), when the draft is produced, then the draft is in Hinglish; given a review in any language other than English or Hindi, the draft is in English (Q-009, confirmed 2026-10-06).
  Covers: REQ-021
- AC-US-00-002-6. Given a Hindi draft, when it is shown, then it renders legibly with a Devanagari-capable font.
  Covers: REQ-048
- AC-US-00-002-7. Given the gateway has refused calls because the budget is spent, when a draft is requested, then the manager sees a message that drafting is unavailable and can still write a reply by hand.
  Covers: REQ-020

**Not in this story.**
- Posting the reply to Google or anywhere else (PRD non-goal).
- Drafting every review automatically on import (Q-007, confirmed 2026-10-06: on demand only).
- A tone settings screen (Q-010, confirmed 2026-10-06: tone lives in the versioned reply prompt).
- Editing and approving the draft (US-00-003).

**Depends on.**
- US-00-001: the manager sees their reviews.
- US-02-001: drafting goes through the gateway.
- US-02-002: the versioned reply prompt.

**Assumptions.**
- One brand, one tone (Q-001, confirmed 2026-10-06).
- Drafts are made on demand when a manager opens a review, and stored (Q-007, confirmed 2026-10-06).
- Reply in the review's language and script: English to English, Devanagari Hindi to Devanagari Hindi, Hinglish to Hinglish, any other language to English (Q-009, confirmed 2026-10-06).
- The reply prompt is versioned in the repository and each draft records its version (Q-011, confirmed 2026-10-06).
- Tone lives in the versioned reply prompt (Q-010, confirmed 2026-10-06).

**Resolve before build.** none (Q-001, Q-007, Q-009, Q-010, Q-011 confirmed 2026-10-06)

### US-00-003 Edit, approve and mark a reply as replied

Epic: EP-04   Priority: Must   Points: TBD (estimate)
Persona: Outlet manager, group 00   Ticket: unassigned
Covers: REQ-022, REQ-023, REQ-024, REQ-047   Judgement: merged from REQ-022, REQ-023 (story), REQ-024 (criterion) and REQ-047 (criterion)

**Narrative.** As an outlet manager, I want to edit the draft, approve it and mark the review replied, so that every reply is one I have checked.

**Why it matters.** B2: manager approval is the gate on every reply; "replied" is how the brand sees that every review got an answer.

**From the PRD.**
- REQ-022: "The system lets an outlet manager edit a draft reply."
- REQ-023: "The system lets an outlet manager mark a reply as replied."
- REQ-024: "A reply is not marked replied until an outlet manager has approved it."
- REQ-047: "The interface makes reply actions easy to find."

**Preconditions.**
- A draft exists for the review (US-00-002).

**Acceptance criteria.**

- AC-US-00-003-1. Given a draft, when the outlet manager edits its text and saves, then the edited text is stored and shown next time.
  Covers: REQ-022
- AC-US-00-003-2. Given a draft, when the outlet manager marks it replied, then the review shows as replied with the final text, the manager and the time; marking replied is the approval (Q-008, confirmed 2026-10-06).
  Covers: REQ-023, REQ-024
- AC-US-00-003-3. Given a review, when anyone other than its outlet's manager, the brand admin included, tries to mark it replied, then the service refuses it (Q-002, confirmed 2026-10-06).
  Covers: REQ-024
- AC-US-00-003-4. Given a review whose draft was never approved by a manager, when it is read through any screen or the service, then its status is not replied.
  Covers: REQ-024
- AC-US-00-003-5. Given a review open on screen, when the manager looks for the edit and mark-replied actions, then both are visible without scrolling (acceptance set in design).
  Covers: REQ-047

**Not in this story.**
- Posting the reply to Google (PRD non-goal; the manager posts it by hand).
- A separate "approved" state before "replied" (Q-008, confirmed 2026-10-06: one action).
- Regenerating a draft (not in the PRD).

**Depends on.**
- US-00-002: provides the draft.

**Assumptions.**
- Approve and mark replied are one action (Q-008, confirmed 2026-10-06).
- Only outlet managers approve replies; the brand admin does not (Q-002, confirmed 2026-10-06).

**Resolve before build.** none (Q-002, Q-008 confirmed 2026-10-06)

## EP-05 Get the weekly digest

Goal: the brand admin gets, on demand, a digest naming the outlet and issue that moved most, with the urgent reviews, in the local mail catcher.
Covers: REQ-025, REQ-026, REQ-027, REQ-028, REQ-029

### US-01-009 Generate the weekly digest and send it to the mail catcher

Epic: EP-05   Priority: Must   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-025, REQ-026, REQ-027, REQ-028, REQ-029   Judgement: merged from REQ-025, REQ-028 (story) and REQ-026, REQ-027, REQ-029 (criteria of it)

**Narrative.** As a brand admin, I want to generate the weekly digest when I choose, so that I get the movers and urgent reviews in one email.

**Why it matters.** B1: the digest names the outlet and the issue that moved most.

**From the PRD.**
- REQ-025: "The system produces a weekly digest that lists the biggest movers."
- REQ-026: "The weekly digest lists the urgent reviews."
- REQ-027: "The weekly digest names the outlet and the issue that moved most."
- REQ-028: "The system generates the digest on demand."
- REQ-029: "The system sends the digest by email to the local mail catcher."

**Preconditions.**
- Movers can be computed (US-01-007); MailHog is running.

**Acceptance criteria.**

- AC-US-01-009-1. Given a signed-in brand admin, when they ask for the digest, then it is generated at that moment, and nothing generates or sends it on a schedule (Q-005, confirmed 2026-10-06).
  Covers: REQ-028
- AC-US-01-009-2. Given the latest complete Monday to Sunday week and the week before, when the digest is generated, then it lists the movers ranked as in US-01-007 (measure: Q-004; period: Q-005; both confirmed 2026-10-06).
  Covers: REQ-025
- AC-US-01-009-3. Given the digest week, when the digest is generated, then its first line names the outlet and the theme that moved most, with both weeks' counts.
  Covers: REQ-027
- AC-US-01-009-4. Given urgent reviews dated in the digest week, when the digest is generated, then it lists each with its urgent reasons, outlet, date and text, and lists none from other weeks (urgent reasons: product owner, 2026-10-06).
  Covers: REQ-026
- AC-US-01-009-5. Given MailHog running locally, when the digest is generated, then one email reaches the MailHog inbox addressed to the brand admin, and no other mail server is contacted (Q-006, confirmed 2026-10-06).
  Covers: REQ-029
- AC-US-01-009-6. Given data where the planted wait-time spike falls in the latest complete week, when the digest is generated, then it names that outlet and wait time as the issue that moved most.
  Covers: REQ-027

**Not in this story.**
- Scheduled weekly sending (Q-005, confirmed 2026-10-06: on demand only).
- Choosing which week the digest covers (Q-005, confirmed 2026-10-06: always the latest complete week).
- Digest open rate tracking (not in the PRD; PRD section 10 advises against it).
- Digests to outlet managers (Q-006, confirmed 2026-10-06: brand admin only).

**Depends on.**
- US-01-007: the digest reuses the movers calculation.

**Assumptions.**
- Digest covers the latest complete Monday to Sunday week and the brand admin generates it (Q-005, confirmed 2026-10-06).
- Mover measure: count change in negative reviews per outlet and theme (Q-004, confirmed 2026-10-06).
- Recipient is the brand admin only (Q-006, confirmed 2026-10-06).

**Resolve before build.** none (Q-004, Q-005, Q-006 confirmed 2026-10-06)

## EP-06 Keep model spend and quality under control

Goal: every model call is capped, logged and budgeted, prompts are versioned, tests run without live calls, and tagging and replies are measured.
Covers: REQ-030, REQ-031, REQ-032, REQ-033, REQ-034, REQ-035, REQ-036, REQ-037, REQ-038, REQ-039

### US-02-001 Route every model call through one budgeted gateway

Epic: EP-06   Priority: Must   Points: TBD (estimate)
Persona: Developer (inferred:, group 02 operator)   Ticket: unassigned
Covers: REQ-030, REQ-031, REQ-032   Judgement: merged from REQ-031 (story) and REQ-030, REQ-032 (criteria of it)

**Narrative.** As the developer running the product, I want every model call to go through one gateway that caps tokens, logs cost and stops at the budget, so that spend never runs past what the brief allows.

**Why it matters.** B2 and B3: tagging and drafting both depend on model calls, and the OpenRouter key is capped at USD 10; a run that spends it stops the product.

**From the PRD.**
- REQ-030: "The system records the tokens used and the cost of every model call."
- REQ-031: "The system refuses any model call once the running total cost of model calls passes USD 8."
- REQ-032: "The system sets max_tokens to at most 1000 on every model call."

**Preconditions.**
- An OpenRouter key and PostgreSQL are available.

**Acceptance criteria.**

- AC-US-02-001-1. Given the code base, when it is searched for calls to OpenRouter, then exactly one gateway function makes them and every tagging, drafting and evaluation call goes through it.
  Covers: REQ-030
- AC-US-02-001-2. Given any model call, when it completes, then a log row records the model, input tokens, output tokens, cost in USD and the time.
  Covers: REQ-030
- AC-US-02-001-3. Given any model call, when the request is built, then max_tokens is set and is at most 1000, including when a caller asks for more.
  Covers: REQ-032
- AC-US-02-001-4. Given a recorded running total of USD 8.01, when any model call is attempted, then the gateway refuses it, sends nothing to OpenRouter and returns a budget error (Q-015, confirmed 2026-10-06).
  Covers: REQ-031
- AC-US-02-001-5. Given a running total of USD 7.99 or exactly USD 8.00, when a model call is attempted, then the gateway sends it; the call that takes the total above USD 8 is allowed, and the next one is refused (Q-015, confirmed 2026-10-06).
  Covers: REQ-031
- AC-US-02-001-6. Given the application restarts, when the gateway starts, then the running total is read from PostgreSQL and is not reset.
  Covers: REQ-031
- AC-US-02-001-7. Given the gateway configuration, when a call is sent, then the model is Claude Haiku 4.5 under OpenRouter's own identifier, checked against OpenRouter's model list before the first live call, and when that model is unavailable the call fails with no fallback model (Q-016, confirmed 2026-10-06).
  Covers: REQ-030

**Not in this story.**
- Model tiers, fallbacks or other providers (Q-016, confirmed 2026-10-06: none).
- Prompt versioning (US-02-002).

**Depends on.**
- none

**Assumptions.**
- "Passes USD 8" means the recorded total is above USD 8; the total counts every model call, evaluation and recording runs included, and survives restarts (Q-015, confirmed 2026-10-06).
- The model identifier is OpenRouter's own for Claude Haiku 4.5 (likely "anthropic/claude-haiku-4.5", not yet verified), checked before the first live call (Q-016, confirmed 2026-10-06).
- Drafts are made on demand, which keeps spend tied to use (Q-007, confirmed 2026-10-06).

**Resolve before build.** none (Q-007, Q-015, Q-016 confirmed 2026-10-06); verify the identifier string before the first live call

### US-02-002 Keep model prompts under version

Epic: EP-06   Priority: Must   Points: TBD (estimate)
Persona: Developer (inferred:, group 02 operator)   Ticket: unassigned
Covers: REQ-033   Judgement: story

**Narrative.** As the developer, I want the tagging and reply prompts kept as numbered versions, so that a change in tone or tagging can be traced to the prompt that caused it.

**Why it matters.** B2: the brand's tone lives in the reply prompt; B3: tags stay comparable when they record the prompt that made them.

**From the PRD.**
- REQ-033: "The system versions the prompts it sends to the model."

**Preconditions.**
- The gateway exists (US-02-001).

**Acceptance criteria.**

- AC-US-02-002-1. Given the repository, when the prompts are listed, then the tagging prompt and the reply prompt each exist as numbered versions, and one version of each is marked current (Q-011, confirmed 2026-10-06).
  Covers: REQ-033
- AC-US-02-002-2. Given a stored tag or draft, when it is read, then it names the prompt version that produced it (Q-011, confirmed 2026-10-06).
  Covers: REQ-033
- AC-US-02-002-3. Given the reply prompt, when it is read, then it holds the brand's tone, and changing the tone means adding a new version (Q-001, Q-010 confirmed 2026-10-06: one brand, one tone, kept in the reply prompt).
  Covers: REQ-033

**Not in this story.**
- Creating or switching prompt versions in the app (Q-011, confirmed 2026-10-06: repository only).
- Re-tagging reviews when the tagging prompt changes (REQ-011 tags each review once).

**Depends on.**
- US-02-001: prompts are sent through the gateway.

**Assumptions.**
- One brand, so one tone (Q-001, confirmed 2026-10-06).
- Tone lives in the reply prompt (Q-010, confirmed 2026-10-06).
- Versions live in the repository and are recorded on outputs; a new tagging version does not re-tag old reviews (Q-011, confirmed 2026-10-06).

**Resolve before build.** none (Q-001, Q-010, Q-011 confirmed 2026-10-06)

### US-02-003 Run tests on recorded model responses

Epic: EP-06   Priority: Must   Points: TBD (estimate)
Persona: Developer (inferred:, group 02 operator)   Ticket: unassigned
Covers: REQ-034, REQ-035   Judgement: merged from REQ-034 (story) and REQ-035 (criterion of it)

**Narrative.** As the developer, I want the tests to replay recorded model responses, so that tests are repeatable and CI never spends the model budget.

**Why it matters.** B2 and B3: tagging and drafting can be tested on every change without touching the USD 10 key.

**From the PRD.**
- REQ-034: "The automated tests replay recorded model responses instead of calling the model."
- REQ-035: "The CI pipeline makes no live model calls."

**Preconditions.**
- Calls go through one gateway (US-02-001), which is where responses are recorded and replayed.

**Acceptance criteria.**

- AC-US-02-003-1. Given the test suite, when it runs, then every model response comes from a recorded file and the running total does not change.
  Covers: REQ-034
- AC-US-02-003-2. Given a test that needs a response with no recording, when it runs in replay mode, then it fails with a message naming the missing recording instead of calling the model.
  Covers: REQ-034
- AC-US-02-003-3. Given the CI pipeline, when it runs with no OpenRouter key and network calls to OpenRouter blocked, then the whole suite passes.
  Covers: REQ-035

**Not in this story.**
- Choosing the CI service (later, at technology selection).
- Live evaluation runs (US-02-004, US-02-005).

**Depends on.**
- US-02-001: replay is done at the gateway.

**Assumptions.**
- none beyond the PRD.

**Resolve before build.** none

### US-02-004 Evaluate theme and urgent-flag accuracy on 100 labelled reviews

Epic: EP-06   Priority: Must   Points: TBD (estimate)
Persona: Developer (inferred:, group 02 operator)   Ticket: unassigned
Covers: REQ-009, REQ-036, REQ-037, REQ-038   Judgement: merged from REQ-036, REQ-037 (story) and REQ-038 (criterion); REQ-009 added for the sentiment report (product owner, 2026-10-06)

**Narrative.** As the developer, I want an evaluation on 100 labelled reviews, so that I know how often themes and urgent flags are right before trusting the tags.

**Why it matters.** B3: trends are only as good as the tags; a missed urgent review is the costliest error, so recall is the gate.

**From the PRD.**
- REQ-036: "The project includes an evaluation that measures theme accuracy on 100 labelled reviews."
- REQ-037: "The evaluation measures urgent-flag accuracy on the same 100 labelled reviews."
- REQ-038: "The classification evaluation passes or fails on urgent recall."
- REQ-009: "The system records a sentiment for each tagged review." (measured here, report only)

**Preconditions.**
- Tagging works (US-01-003); recording and replay work (US-02-003).

**Acceptance criteria.**

- AC-US-02-004-1. Given 100 real public reviews with reviewer names removed, collected and labelled by the product owner with themes, one sentiment and urgent reasons, at least 20 of them urgent (at least 5 per reason) and at least 20 in Hindi or Hinglish (product owner, 2026-10-06), when the evaluation runs, then it reports theme precision and recall per theme (Q-018, Q-021 confirmed 2026-10-06).
  Covers: REQ-036
- AC-US-02-004-2. Given the same 100 reviews, when the evaluation runs, then it reports urgent recall and urgent precision, and lists each urgent review the model missed.
  Covers: REQ-037
- AC-US-02-004-3. Given urgent recall on the 100 reviews is 90% or more, when the evaluation runs, then it reports the urgent gate as passed (Q-017, pass mark 90% given by you on 2026-10-06).
  Covers: REQ-038
- AC-US-02-004-4. Given urgent recall below 90% on a manual live run or on the recorded results replayed in CI, when the evaluation runs, then it reports the urgent gate as failed, lists the missed urgent reviews and exits with failure (Q-017, pass mark 90% given by you on 2026-10-06).
  Covers: REQ-038
- AC-US-02-004-5. Given a live evaluation run, when it ends, then its report names the tagging prompt version and the model, and its cost is in the gateway's running total.
  Covers: REQ-036, REQ-037
- AC-US-02-004-6. Given the same 100 reviews, when the evaluation runs, then it reports sentiment accuracy, and precision and recall for negative, with no pass or fail verdict (product owner, 2026-10-06: report only; movers and the digest count negative reviews).
  Covers: REQ-009

**Not in this story.**
- Any pass mark for theme accuracy (Q-018, decided by you on 2026-10-06: report only, no pass mark).
- Reply tone checks (US-02-005).

**Depends on.**
- US-01-003: the tagging under test.
- US-02-003: the evaluation can be replayed from recordings.

**Assumptions.**
- Themes are scored by precision and recall per theme, with no pass mark (Q-018, confirmed 2026-10-06).
- The evaluation runs by hand with live calls and CI replays the recorded results; the urgent recall pass mark is 90% (Q-017, given by you on 2026-10-06).
- The report names the tagging prompt version, which each stored tag records (Q-011, confirmed 2026-10-06).
- Several themes per review (Q-021, confirmed 2026-10-06).

**Resolve before build.** none (Q-017, Q-018, Q-021 confirmed 2026-10-06; Q-017 pass mark 90% given)

### US-02-005 Check the tone of 30 reply drafts

Epic: EP-06   Priority: Must   Points: TBD (estimate)
Persona: Developer (inferred:, group 02 operator)   Ticket: unassigned
Covers: REQ-039   Judgement: story

**Narrative.** As the developer, I want 30 reply drafts checked for tone, so that I know the replies sound like the brand before managers rely on them.

**Why it matters.** B2: replies a manager has to rewrite every time do not save the dedicated person.

**From the PRD.**
- REQ-039: "The project includes a tone check of 30 reply drafts."

**Preconditions.**
- Drafting works (US-00-002).

**Acceptance criteria.**

- AC-US-02-005-1. Given 30 reviews chosen for the check, English and Hindi among them, when drafts are produced, then all 30 drafts are saved with the reply prompt version that made them.
  Covers: REQ-039
- AC-US-02-005-2. Given the 30 drafts and a written tone rubric, when a person scores them, then each draft has a score per rubric item and the totals are reported (Q-019, confirmed 2026-10-06).
  Covers: REQ-039
- AC-US-02-005-3. Given the 30 drafts are scored, when the check is reported, then it shows the scores per draft and per rubric item with no pass or fail verdict (Q-019, decided by you on 2026-10-06: report only).
  Covers: REQ-039

**Not in this story.**
- A model judge (Q-019, confirmed 2026-10-06: a person scores).

**Depends on.**
- US-00-002: produces the drafts.

**Assumptions.**
- A person scores the drafts against a written rubric; report only, no pass mark (Q-019, decided by you on 2026-10-06).
- Drafts record the prompt version (Q-011, confirmed 2026-10-06).

**Resolve before build.** none (Q-011, Q-019 confirmed 2026-10-06)

## EP-07 Demo with realistic seed data

Goal: a fresh installation shows 5 outlets, 6 months of reviews and a visible wait-time spike.
Covers: REQ-040, REQ-041, REQ-042, REQ-043

### US-02-006 Seed the demo with 5 outlets and 1,500 reviews

Epic: EP-07   Priority: Must   Points: TBD (estimate)
Persona: Developer (inferred:, group 02 operator)   Ticket: unassigned
Covers: REQ-040, REQ-041, REQ-042, REQ-043   Judgement: merged from REQ-040 (story) and REQ-041 to REQ-043 (criteria of it)

**Narrative.** As the developer, I want one script that seeds the demo data, so that anyone can show the product with a known spike to find.

**Why it matters.** B1: the planted spike is how the demo shows the outlet and issue that moved most.

**From the PRD.**
- REQ-040: "The repository includes a seed script that creates the demo data."
- REQ-041: "The seed script creates 5 outlets."
- REQ-042: "The seed script creates 1,500 reviews spread across 6 months."
- REQ-043: "The seed data includes a planted wait-time spike at one outlet."

**Preconditions.**
- PostgreSQL is running in Docker.

**Acceptance criteria.**

- AC-US-02-006-1. Given an empty database, when the seed script runs, then it completes with one command from the repository.
  Covers: REQ-040
- AC-US-02-006-2. Given the seed has run, when outlets are counted, then there are exactly 5.
  Covers: REQ-041
- AC-US-02-006-3. Given the seed has run, when reviews are counted, then there are exactly 1,500, dated across 6 months.
  Covers: REQ-042
- AC-US-02-006-4. Given the seed has run, when wait-time reviews per week are counted for each outlet, then one outlet shows a spike that the movers view ranks first for that week (spike size set in design; measure per Q-004, confirmed 2026-10-06).
  Covers: REQ-043
- AC-US-02-006-5. Given the seed has run, when accounts are listed, then one brand admin and one outlet manager per outlet exist, each assigned to its outlet (Q-003, confirmed 2026-10-06).
  Covers: REQ-040

**Not in this story.**
- The 25-outlet example use case (the seed stays at 5 outlets; PRD constraints).
- Pre-filled tags: whether seeding runs the model is set in design (PRD section 10 note on cost).

**Depends on.**
- US-01-001: outlets have somewhere to live.

**Assumptions.**
- The seed creates the accounts (Q-003, confirmed 2026-10-06).
- The spike is measured the way movers are: count of negative wait-time reviews (Q-004, confirmed 2026-10-06).
- inferred: the movers view and digest always show the latest complete week (Q-005, confirmed 2026-10-06), so the spike is visible there only if it falls in that week at demo time; where the seed places it is set in design.

**Resolve before build.** none

## EP-08 Stretch: tenant colours (not in the MVP)

Goal: each tenant sees the interface in its own colours. Built only after the MVP, and only if you approve it.
Covers: REQ-049

### US-01-010 Show each tenant's own colours (stretch)

Epic: EP-08   Priority: Could   Points: TBD (estimate)
Persona: Brand admin, group 01   Ticket: unassigned
Covers: REQ-049   Judgement: story (stretch)

**Narrative.** As a brand admin, I want the interface in my brand's colours, so that it feels like our own tool.

**Why it matters.** B2: a tool that feels like the brand's own is more likely to be used for every reply. Stretch only.

**From the PRD.**
- REQ-049: "The system applies each tenant's own colours to the interface."

**Preconditions.**
- The MVP is complete and the design system exists.

**Acceptance criteria.**

- AC-US-01-010-1. Given the brand's own colour set, when its users open the interface, then accents use that set and every text and colour pair passes the design system's contrast check (Q-001, confirmed 2026-10-06: one brand per installation, so the tenant is the brand).
  Covers: REQ-049
- AC-US-01-010-2. Given no tenant colour set, when the interface opens, then it uses the default theme.
  Covers: REQ-049

**Not in this story.**
- Anything in the MVP (EP-01 to EP-07).
- Several tenants in one installation (Q-001, confirmed 2026-10-06: one brand per installation).

**Depends on.**
- US-01-005: the first screen whose colours it themes.

**Assumptions.**
- With one brand per installation, "tenant colours" means that brand's colours (Q-001, confirmed 2026-10-06).

**Resolve before build.** none (Q-001 confirmed 2026-10-06)
