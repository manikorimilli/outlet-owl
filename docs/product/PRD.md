# PRD: Review Intelligence for Multi-Outlet Businesses

Source: docs/product/source/brief.txt (the project brief pasted in session, 156 lines; `L<n>` below cites its lines)   Normalised: 2026-10-06
Owner: unconfirmed   Tracker epic: unconfirmed

Verticals named in the brief: restaurants, hotels, clinics and retail chains (L7).
AI depth named in the brief: LLM classification, reply generation and prompt versioning (L10).

## 1. Problem

Not stated as a problem in the input. The closest passage is the intended use case (L19 to L24): a 25-outlet restaurant chain sees that one outlet's wait-time complaints tripled after a staff change, and replies to every review within a day without a dedicated person. The brief says to treat this as the intended use case, not as evidence of results.

## 2. Business objectives

| Id | Objective | Target (measurable) | Source |
| --- | --- | --- | --- |
| B1 | A multi-outlet business sees which outlet and which issue changed most, week over week. | target: unconfirmed | L16, L19 to L20 |
| B2 | Every review gets an on-brand reply, approved by a manager, without a dedicated person. | target: unconfirmed ("within a day" appears in the use case only, which the brief says is not a result) | L15, L20 to L21, L23 to L24 |
| B3 | Sentiment and themes are tracked per outlet over time. | target: unconfirmed | L13 to L15 |

## 3. Non-goals

- Posting replies to Google; the application does not post replies anywhere (L66, L117).
- Live connectors that fetch reviews from an external source (L39, L118).
- A working Google connector; it exists as an interface only (L38, L134).
- Per-brand classification themes (L119).
- Tenant colours in the required initial release; they are a stretch feature (L123, L125).

## 4. Personas

| Persona | Group | Who they are | What they need | Source |
| --- | --- | --- | --- | --- |
| Brand admin | 01 admin | Administers the brand's outlets and review data; signs in with an account created by the seed or configuration (Q-003, confirmed 2026-10-06) | Add outlets, import reviews by CSV, see outlet comparisons and the weekly digest | L29, L33 to L34 |
| Outlet manager | 00 end user | Manages one assigned outlet and sees only that outlet's dashboard, reviews and drafts; only managers approve replies (Q-002, confirmed 2026-10-06) | Edit the drafted reply, approve it and mark it replied | L30, L64 to L65 |

## 5. Requirement statements

One testable statement per id. Ids are never reused or renumbered.

### Outlets and review import

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-001 | The system lets a brand admin add an outlet. | Brand admin | L33 | none |
| REQ-002 | The system lets a brand admin import reviews from a CSV file. | Brand admin | L34, L13 | Q-023 confirmed 2026-10-06: valid rows are imported, rejected rows are reported with row number and reason |
| REQ-003 | The CSV import reads, for each review, the fields outlet, source, date, rating, text and reviewer name. | Brand admin | L35 | none |
| REQ-004 | The system imports reviews through a connector interface, with CSV as one connector, so that other review sources can be added later as further connectors. | Brand admin | L36 to L37 | none |
| REQ-005 | The system defines a Google connector as an interface only, with no working implementation behind it. | Brand admin | L38, L134 | none |

### Review classification

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-006 | The system tags reviews by theme with the model, sending reviews to the model in batches of 20. | | L42, L13 to L14 | Q-021 confirmed 2026-10-06: zero, one or several themes per review; Q-022 confirmed 2026-10-06: tagging starts automatically after each import |
| REQ-007 | The system takes the themes from a configurable list. | | L43 | Q-012 confirmed 2026-10-06: one list for the installation in configuration, changed by a developer; existing tags are kept |
| REQ-008 | The initial theme list is food, wait time, staff, cleanliness and price. | | L44, L14 | none |
| REQ-009 | The system records a sentiment for each tagged review. | | L45, L14 | Q-013 confirmed 2026-10-06: one label per review: positive, neutral or negative |
| REQ-010 | The system flags a review as urgent when it concerns food safety, harassment or a legal threat. | | L46 | none |
| REQ-011 | The system stores each successful tag result and never sends an already tagged review to the model for tagging again. | | L47 | none |
| REQ-012 | The model's batch tagging output identifies each result by its review ID. | | L51 | none |
| REQ-013 | The system validates the batch tagging output per review ID. | | L51 | Q-014 confirmed 2026-10-06: a result for an ID not in the batch is discarded; a result that fails validation counts as missing |
| REQ-014 | After a batch, the system retries tagging only for the review IDs missing from the output. | | L51 to L52 | Q-014 confirmed 2026-10-06: "missing" includes results that failed validation; retries stop at a limit set in design, then the review waits for the next tagging run |

### Dashboard

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-015 | The dashboard shows the rating trend over time for each outlet. | Brand admin, Outlet manager | L55 | none |
| REQ-016 | The dashboard shows the sentiment trend over time for each outlet. | Brand admin, Outlet manager | L56, L14 to L15 | none |
| REQ-017 | The dashboard shows a heatmap of themes by outlet. | Brand admin, Outlet manager | L57 | ambiguous: acceptance set in design |
| REQ-018 | The dashboard shows the biggest movers week over week. | Brand admin, Outlet manager | L58 | Q-004 confirmed 2026-10-06: outlet and theme pairs ranked by change in the number of negative reviews; Q-005 confirmed 2026-10-06: Monday to Sunday weeks, latest complete week against the week before |
| REQ-019 | The dashboard includes a review list that can be searched. | Brand admin, Outlet manager | L59 | Q-020 confirmed 2026-10-06: text search over review text and reviewer name, plus filters for outlet, theme, sentiment, urgent and reply status |

### Reply generation

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-020 | The system drafts a reply to each review with the model, in the brand's tone. | Outlet manager | L62, L15 | Q-007 confirmed 2026-10-06: drafted on demand when a manager opens the review, then stored; Q-010 confirmed 2026-10-06: tone lives in the versioned reply prompt |
| REQ-021 | The drafted reply is written in the review's language, English or Hindi. | Outlet manager | L63 | Q-009 confirmed 2026-10-06: reply in the review's language and script (English, Devanagari Hindi, Hinglish); any other language gets English |
| REQ-022 | The system lets an outlet manager edit a draft reply. | Outlet manager | L64 | none |
| REQ-023 | The system lets an outlet manager mark a reply as replied. | Outlet manager | L64 | Q-008 confirmed 2026-10-06: marking replied is the approval, one action |
| REQ-024 | A reply is not marked replied until an outlet manager has approved it. | Outlet manager | L65, L15 | Q-008 confirmed 2026-10-06: approval and marking replied are one action |

### Weekly digest

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-025 | The system produces a weekly digest that lists the biggest movers. | Brand admin | L69, L16 | Q-004, Q-005 confirmed 2026-10-06 |
| REQ-026 | The weekly digest lists the urgent reviews. | Brand admin | L69 | Q-005 confirmed 2026-10-06: urgent reviews dated in the latest complete Monday to Sunday week |
| REQ-027 | The weekly digest names the outlet and the issue that moved most. | Brand admin | L70, L16 | Q-004 confirmed 2026-10-06: "issue" means theme |
| REQ-028 | The system generates the digest on demand. | Brand admin | L71 | Q-005 confirmed 2026-10-06: the brand admin generates it; no scheduled sending |
| REQ-029 | The system sends the digest by email to the local mail catcher. | Brand admin | L72, L79 | Q-006 confirmed 2026-10-06: one email to the brand admin only |

### Model gateway and prompts

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-030 | The system records the tokens used and the cost of every model call. | | L86 | none |
| REQ-031 | The system refuses any model call once the running total cost of model calls passes USD 8. | | L87, L89 to L91 | Q-015 confirmed 2026-10-06: refused once the recorded total is above USD 8; the crossing call is allowed; the total counts every call and survives restarts |
| REQ-032 | The system sets max_tokens to at most 1000 on every model call. | | L85 | none |
| REQ-033 | The system versions the prompts it sends to the model. | | L10, L132 | Q-011 confirmed 2026-10-06: tagging and reply prompts as numbered versions in the repository; each stored tag and draft records its version; no re-tagging on a new version |

### Testing

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-034 | The automated tests replay recorded model responses instead of calling the model. | | L94 | none |
| REQ-035 | The CI pipeline makes no live model calls. | | L95 | none |

### Evaluations

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-036 | The project includes an evaluation that measures theme accuracy on 100 labelled reviews. | | L99 | Q-018 confirmed 2026-10-06: precision and recall per theme; no pass mark |
| REQ-037 | The evaluation measures urgent-flag accuracy on the same 100 labelled reviews. | | L99 | Q-017 confirmed 2026-10-06: reports urgent recall and precision and lists missed urgent reviews |
| REQ-038 | The classification evaluation passes or fails on urgent recall. | | L100, L103 to L104 | Q-017 confirmed 2026-10-06: pass mark given by the user: urgent recall 90% or more passes, below 90% fails |
| REQ-039 | The project includes a tone check of 30 reply drafts. | | L101, L103 to L104 | Q-019 confirmed 2026-10-06: a person scores each draft against a written rubric; report only, no pass mark (decided by the user) |

### Seed data

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-040 | The repository includes a seed script that creates the demo data. | | L111 | none |
| REQ-041 | The seed script creates 5 outlets. | | L108, L113 | none |
| REQ-042 | The seed script creates 1,500 reviews spread across 6 months. | | L109 | none |
| REQ-043 | The seed data includes a planted wait-time spike at one outlet. | | L110 | ambiguous: acceptance set in design |

### User interface

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-044 | The interface presents a clear information hierarchy, readable charts and tables, consistent typography and spacing, and restrained colours. | Brand admin, Outlet manager | L143, L146 to L147 | ambiguous: acceptance set in design |
| REQ-045 | The interface makes outlet comparisons easy to find. | Brand admin, Outlet manager | L148 | ambiguous: acceptance set in design |
| REQ-046 | The interface makes urgent reviews easy to find. | Brand admin, Outlet manager | L148 | ambiguous: acceptance set in design |
| REQ-047 | The interface makes reply actions easy to find. | Outlet manager | L148 | ambiguous: acceptance set in design |
| REQ-048 | The interface displays English and Hindi text legibly. | Brand admin, Outlet manager | L151 | Q-009 confirmed 2026-10-06: Devanagari and Latin-script Hindi |

### Stretch (not in the required initial release)

| Id | Statement | Persona | Source | Flags |
| --- | --- | --- | --- | --- |
| REQ-049 | The system applies each tenant's own colours to the interface. | Brand admin | L123, L125, L130 to L131 | stretch: not in the initial release; Q-001 confirmed 2026-10-06: one brand per installation, so the tenant's colours are that brand's colours |

## 6. Constraints

Runtime
- Runtime needs are exactly: an OpenRouter key capped at USD 10, PostgreSQL in Docker, MailHog in Docker for the digest. Nothing else (L76 to L80).
- The local mail catcher that receives the digest is MailHog (L72, L79).

Model gateway
- All model calls go through one gateway function on OpenRouter (L83).
- Requested model: claude-haiku-4-5 (L84). The requested model is not changed. Confirmed (Q-016, 2026-10-06): OpenRouter's own identifier for Claude Haiku 4.5 is used, checked against OpenRouter's model list before the first live call; no fallback model. The exact string is not yet verified.
- max_tokens is capped at 1000 (L85, REQ-032).
- Tokens and cost are logged per call (L86, REQ-030).
- Budget wording, preserved as written: "Refuse calls once the running total passes USD 8." (L87). Boundary confirmed (Q-015, 2026-10-06): a call is refused once the recorded total is above USD 8; the call that crosses USD 8 is allowed; the total counts every call, evaluation and recording runs included, and survives restarts.

Testing
- Tests replay recorded responses; CI makes no live model calls (L94 to L95).

Connectors
- Importing goes through a connector interface; the Google connector stays an interface only, because Google API access takes weeks (L36 to L38, L134).

Seed data
- The demo seed has 5 outlets and is kept distinct from the 25-outlet example use case (L113).
- The seed is created by a script (L111).

Design stage (process constraints from the brief)
- Avoid clutter and decorative elements that distract from the review data (L149 to L150). Motion is limited to a short fade-in on load and a hover lift on cards and buttons (200 to 420 ms), switched off when the person's system asks for reduced motion; charts and data are never animated (decided by the product owner on 2026-10-06, screen-design round 2).
- The proposed design is shown to the requester before it is implemented (L152).
- A reference website may be suggested at the design stage with the relevant design qualities explained; another product's branding is not copied (L153 to L154).
- Visual polish is kept separate from adding new product features (L155).
- No UI framework or component library is selected during the PRD step (L156).
- No application tech stack is chosen during the PRD step; options are presented with a recommendation at technology selection and the requester decides (session instruction 5).

## 7. Open questions

23 entries in docs/product/questions.md: 0 open, 0 need your confirmation; 23 confirmed by you on 2026-10-06. Q-022 and Q-023 were added by backlog. Urgent recall pass mark given 2026-10-06: 90% (Q-017). Theme accuracy (Q-018) and the tone check (Q-019) are report only, with no pass mark (decided 2026-10-06). No values are left to give.

## 8. Could not extract

- Problem statement: the input gives an intended use case, not a problem statement. The brand owner or product sponsor can state the problem in their words.
- Business objective targets: none of B1 to B3 has a target the input calls a target. The product sponsor can set them.
- Owner and tracker epic: not in the input.

## 9. Glossary

| Term | Meaning | Source |
| --- | --- | --- |
| Outlet | One location of a multi-outlet business (a restaurant, hotel, clinic or shop) whose reviews are collected and compared. | L7, L13 |
| Brand | The business that owns the outlets; replies are drafted in its tone. One brand per installation (Q-001, confirmed 2026-10-06). | L15, L62 |
| Connector | The interface through which reviews are imported; CSV is implemented, Google is an interface only, others can be added later. | L36 to L38 |
| Theme | A topic a review is tagged with, from a configurable list; initially food, wait time, staff, cleanliness, price. | L13 to L14, L43 to L44 |
| Urgent flag | A marker on a review that concerns food safety, harassment or a legal threat. | L46 |
| Biggest movers | Outlet and theme pairs ranked by the change in the number of negative reviews for that theme, one week against the week before; "issue" means theme (Q-004, confirmed 2026-10-06). | L58, L69 to L70 |
| Weekly digest | A summary of movers and urgent reviews, naming the outlet and the issue that moved most, generated on demand and emailed to the local mail catcher. | L16, L69 to L72 |
| Local mail catcher | MailHog running in Docker, which receives the digest email instead of a real mailbox. | L72, L79 |
| Running total | The accumulated cost of model calls, against which the USD 8 refusal rule applies. | L87 |
| Recorded responses | Saved model responses that tests replay so no live model call is made. | L94 |
| Tenant colours | Per-tenant interface colours; stretch feature only. | L123, L125 |

## 10. Suggestions (not requirements)

These are recommendations, kept apart from the requirements above. None is in scope until you approve it.

- Your brg_* shorthand mapped to the installed Bearing skills:
  - brg_design_system is `bearing:design-system`. Useful: the product is dashboard heavy, and one set of tokens for charts, heatmap scales, tables and Hindi-capable typography supports REQ-044 to REQ-048. Compatible with scope. Use it at the design stage, after `bearing:screen-design` shows you the proposed screens.
  - brg_theme is `bearing:themes`. Useful only for the stretch feature REQ-049, and it needs the design system first. With one brand per installation (Q-001, confirmed 2026-10-06) it themes that one brand. Defer until the initial release is done.
  - brg_prompt is `bearing:prompt-registry`. Useful: the brief names prompt versioning (L10), which is REQ-033. The brand's tone lives in the versioned reply prompt (Q-001, Q-010, Q-011 confirmed 2026-10-06). Compatible with scope.
  - brg_events is `bearing:analytics-events`. Not recommended now: digest open rate is not a requirement in the brief, MailHog is a local catcher with no real recipients, and measuring opens would need extra tracking that the "Nothing else" runtime rule (L80) does not allow.
  - Google connector as an interface only: already REQ-005; agreed.
- Other installed skills that match stated requirements: `bearing:llm-gateway` for REQ-030 to REQ-032, `bearing:llm-eval` for REQ-036 to REQ-039.
- Batch size and token cap together: 20 reviews per call under max_tokens 1000 leaves about 50 output tokens per review, so the tagging output needs a compact format. The per-ID retry (REQ-014) also covers a truncated batch. Design note only; no requirement changes.
- Seed data: if Hindi replies (REQ-021, REQ-048) are to be shown in the demo, the seed needs some Hindi reviews. The 6 months of seed reviews could end near the seed date so the latest week has data for the digest. Rough cost estimate, to be checked in design: tagging 1,500 seed reviews is about 75 batch calls, roughly USD 0.5 at Claude Haiku 4.5 list prices.
