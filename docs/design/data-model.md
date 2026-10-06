# Data model: Review Intelligence (OutletOwl)

**Store:** PostgreSQL · **Tables:** 9 · **Columns:** 71 · **Indexes:** 10 · **Personal-data columns:** 8

One PostgreSQL database holds everything the product remembers: outlets, accounts loaded from the users file, CSV imports and their rejected rows, reviews, the one tag result per review, the one reply per review, generated digests, and the model call log that is the USD 8 running total. The Go server (API, tagging worker, model gateway, digest builder) and the seed, eval and tone-check commands read and write it through sqlc queries (ADR-0003). It is small by design: tens of thousands of rows at most (HLD section 8), so a few b-tree indexes serve every screen and nothing else is needed.

- Task: none (design before the first task)
- Serves: US-01-001 to US-01-009, US-00-001 to US-00-003, US-02-001 to US-02-006; REQ-001 to REQ-043; ADR-0001, ADR-0003, ADR-0006, ADR-0007
- ADRs: PostgreSQL is fixed by the PRD (section 6) and ADR-0003; no store ADR is needed
- Stores: postgres
- Companion files: `docs/design/schema.sql`, `docs/design/data-dictionary.csv`, `docs/design/erd.md`
- Author: manikorimilli, 2026-10-06. Status: Draft. Version: v2 (base: v1, commit 88dec6b)

## Changes from v1

Why this revision: the product owner approved on 2026-10-06 storing why a review is urgent, because the approved screens S-02, S-04 and S-08 and the digest show the reason beside each urgent review, and v1 stored only a yes or no flag (decision recorded in docs/genai/review-classification-and-replies-solution.md; ADR-0008 stays Proposed).

| Section | Change | Driven by | Impact |
| --- | --- | --- | --- |
| 4. PostgreSQL tables (`review_tags`) | New column `urgent_reasons text[]`, limited to `food_safety`, `harassment`, `legal_threat`; `is_urgent` must be true exactly when it is not empty; reasons cannot repeat | REQ-010; screens S-02, S-04, S-08; product owner approval, 2026-10-06 | schema.sql, data-dictionary.csv, erd.md; OpenAPI `Tags` needs `urgent_reasons` (openapi-spec revision) |
| 5. Writers, claims and derived rows | The tagging writer stores the urgent reasons from the result line | genai solution section 4.1 | low-level-design for the tagging worker |
| 6. Enumerations | New closed value set `urgent_reasons` (a CHECK, not an enum type) | REQ-010 | none beyond the above |
| 8. Migration plan | The column joins planned migration 00002; a fallback 00005, with its Down and deploy order, is named if 00002 is already applied | no migration exists yet (`db/migrations/` absent, checked 2026-10-06) | db-migration |
- Decided by the product owner for this model, 2026-10-06: all six CSV fields are required; ratings are whole stars from 1 to 5; an outlet may have several managers; personal data is kept, with no automatic deletion.

Not covered: US-01-010 (tenant colours, stretch) stores nothing; REQ-044 to REQ-049 are interface requirements with no data.

## 1. Why these stores

### PostgreSQL

**Holds:** every entity in section 2, in two schemas: `public` for the domain tables and `budget` for the model call log.

The PRD allows exactly one database, PostgreSQL in Docker (section 6, "Nothing else"), and ADR-0003 builds on it. The data is relational and consistency-critical in small volumes: a tag result must belong to a review, a reply must be claimed once, a digest and an import must not repeat under the same request id, and the running total must be read inside the database before each call (tenet 2). Every read is a filtered aggregate over at most about 10^4 reviews.

The `budget` schema exists so the running total survives everything that resets demo data: the seed truncates the `public` domain tables only, and the budget migrations have no Down (HLD section 4, REQ-031, Q-015).

| Considered | Why not |
| --- | --- |
| MongoDB | The PRD's runtime rule allows PostgreSQL only; the entities are fixed shapes joined on every dashboard read. |
| ClickHouse | Not allowed at runtime; the trends and movers aggregate about 10^4 rows, which PostgreSQL answers in milliseconds. |
| A search service (Elasticsearch, Meilisearch) | Not allowed at runtime (ADR-0003); search is a substring match over at most about 10^4 reviews. |

## 2. Entities: ownership and lifecycle

All entities are owned by the API server module of the one Go binary (HLD section 4); the model call log is written only by the model gateway.

| Entity | Owner | Created by | Changed by | Ended by | Stories | PII | Retention |
| --- | --- | --- | --- | --- | --- | --- | --- |
| outlet | API server | admin adds it (US-01-001); seed | never (no edit path) | never; seed reset truncates | US-01-001, US-01-002, US-01-005 to US-01-007, US-00-001 | no | kept |
| user | API server | users file upsert at start; seed writes the file | users file upsert (name, role, outlet, hash) | marked removed when gone from the file; seed reset truncates | US-00-001, US-00-003, US-01-001, US-01-009, US-02-006 | yes | kept |
| import | API server (CSV connector) | admin uploads a CSV (US-01-002) | its counts, once, before the same transaction commits | never; seed reset truncates | US-01-002 | no | kept |
| import rejection | API server (CSV connector) | the same import transaction | never | with its import | US-01-002 | no | kept |
| review | API server (CSV connector) | CSV import; seed command | never | never; seed reset truncates | US-01-002, US-01-003, US-01-005 to US-01-009, US-00-001, US-00-002, US-02-006 | yes | kept |
| review tag result | tagging worker | tagging worker or seed command, once per review | never (REQ-011) | with its review | US-01-003 to US-01-009, US-02-002 | no | kept |
| reply | API server | manager opens a review (draft claim) or writes by hand | draft lands, manager edits, manager marks replied | with its review | US-00-002, US-00-003, US-01-008, US-02-002 | yes | kept |
| digest | digest builder | admin generates it (US-01-009) | status sending to sent or failed | never; seed reset truncates | US-01-009 | yes | kept |
| model call | model gateway | reserved before every live call; start-up reconciliation | settled or failed after the call | never; not truncated, no Down | US-02-001, US-02-002, US-02-004, US-02-005 | no | kept forever (REQ-031) |

Not modelled:

- tagging attempts and retry counts, because ADR-0006 keeps them in memory for one pass and "untagged" is the absence of a tag result;
- the theme list and the brand timezone, because they are configuration (Q-012, HLD section 4);
- prompt versions, recordings and the 100 labelled evaluation reviews, because they are files in the repository (Q-011, HLD sections 4 and 5);
- tone-check scores, because US-02-005 writes them to a Markdown file a person fills in (HLD section 3);
- refused model calls, because a refusal sends nothing and costs nothing (AC-US-02-001-4); it is logged by the server, not stored;
- sessions, because sign-in is a JWT cookie with nothing stored (ADR-0007).

## 3. Relationships

| From | To | Cardinality | FK column | On delete | Why |
| --- | --- | --- | --- | --- | --- |
| outlets | users | one-to-many | users.outlet_id | RESTRICT | An outlet that still has managers cannot vanish under them; several managers per outlet are allowed (product owner, 2026-10-06). |
| outlets | reviews | one-to-many | reviews.outlet_id | RESTRICT | Every review is compared under its outlet; deleting an outlet must not silently delete its reviews. |
| imports | reviews | one-to-many | reviews.import_id | RESTRICT | An import with reviews stays, so a bad import can be traced (HLD section 4); seeded reviews have no import. |
| imports | import_rejections | one-to-many | import_rejections.import_id | CASCADE | The rejected-row list is part of the import's result and has no meaning without it. |
| reviews | review_tags | one-to-zero-or-one | review_tags.review_id | CASCADE | A tag result is derived from its review and follows it. |
| reviews | replies | one-to-zero-or-one | replies.review_id | CASCADE | A reply is derived from its review and follows it. |
| users | replies | one-to-many | replies.replied_by | RESTRICT | A replied review shows its approving manager (AC-US-00-003-2), so that user is marked removed, never deleted. |

No path in the PRD deletes an outlet, an import or a review (HLD section 17), so the CASCADE rules act only when a developer resets the database. The diagram and one sentence per relationship are in [erd.md](erd.md).

## 4. PostgreSQL tables

Tables in the order `schema.sql` creates them. Column rows are copied from `data-dictionary.csv`.

### `outlets`: Outlets (hot: no)

One location of the brand whose reviews are imported and compared.

Serves US-01-001, US-01-002, US-01-005, US-01-006, US-01-007, US-00-001. Expected volume: 5 in the demo, 25 in the example use case (10^1); one row per outlet the admin adds.

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `id` | `bigint` | No | PK | `identity` | Surrogate key; reviews and managers reference it. | AC-US-01-001-1: an added outlet can be chosen as the outlet of imported reviews. |
| `name` | `text` | No | UK |  | Outlet name as the admin typed it; the CSV outlet column must match it, ignoring case. | AC-US-01-001-1: the admin adds an outlet with a name; HLD flow A matches each CSV row's outlet by this name. |
| `created_at` | `timestamptz` | No |  | `now()` | When the outlet was added. | Rule: audit columns (database postgres reference). |
| `updated_at` | `timestamptz` | No |  | `now()` | Last change to this row. | Rule: audit columns (database postgres reference). |

**Indexes**

- `uq_outlets_name_lower`: unique on (lower(name)). The CSV connector finds each row's outlet by name ignoring case, and a repeated add of the same name returns the existing outlet instead of a second one (HLD flow A and section 5; S-06 duplicate-name state).

**Constraints**

- `chk_outlets_name_not_blank`: `btrim(name) <> ''`. Refuses an outlet no CSV row could ever name.
- `chk_outlets_name_trimmed`: `name = btrim(name)`. Refuses "Koramangala " beside "Koramangala", which the case-insensitive unique index alone would let through.

### `users`: Users (hot: no)

An account loaded from the users file at server start; the seed writes the same file.

Serves US-00-001, US-00-003, US-01-001, US-01-009, US-02-006. Expected volume: 6 in the demo, about 26 for the example (10^1).

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `id` | `bigint` | No | PK | `identity` | Surrogate key; the only identity the JWT carries. | ADR-0007: the token identifies the user only and role and outlet are re-read from this row. |
| `email` | `text` | No | UK |  | Sign-in name and the digest recipient address. **(personal data)** | AC-US-00-001-1: the manager signs in; AC-US-01-009-5: the digest is addressed to the brand admin. |
| `name` | `text` | No |  |  | Name shown in the top bar and beside a reply the manager marked replied. **(personal data)** | AC-US-00-003-2: the review shows as replied with the manager. |
| `role` | `user_role` | No |  |  | brand_admin or outlet_manager, re-read on every request. | Q-002: the admin sees all outlets, a manager one; ADR-0007 re-reads the role per request. |
| `outlet_id` | `bigint` | Yes | FK outlets.id (RESTRICT) |  | The one outlet an outlet manager may see and act on; null for the brand admin. | Q-002 and AC-US-00-001-2 to -4: a manager sees only the assigned outlet; tenet 3 takes the scope from this row. |
| `password_hash` | `text` | No |  |  | bcrypt hash from the users file; never the password. | ADR-0007: the password is checked against a bcrypt hash; AC-US-00-001-1 refuses a wrong password. |
| `removed_at` | `timestamptz` | Yes |  |  | Set when the account left the users file; sign-in is refused and the row stays. | AC-US-00-003-2: a replied review keeps showing its manager after that manager leaves the users file, so the row cannot be deleted. |
| `created_at` | `timestamptz` | No |  | `now()` | When the account was first loaded. | Rule: audit columns (database postgres reference). |
| `updated_at` | `timestamptz` | No |  | `now()` | Last change from the users file. | Rule: audit columns; the start-up upsert changes name, role, outlet or hash. |

**Indexes**

- `uq_users_email_lower`: unique on (lower(email)). Sign-in finds the account by email ignoring case (AC-US-00-001-1), and the start-up upsert matches file lines to rows by it.
- `uq_users_one_brand_admin`: unique on (role) where `role = 'brand_admin' AND removed_at IS NULL`. One active brand admin: the HLD has one brand admin per installation (section 2) and the digest goes to "the brand admin" (Q-006). The query repeating the predicate is the digest's recipient lookup.
- `idx_users_outlet_id`: on (outlet_id). The managers of an outlet on the outlets screen (S-06), and the foreign-key index rule.

**Constraints**

- `chk_users_email_shape`: `email ~ '^[^@[:space:]]+@[^@[:space:]]+$'`. Refuses a users file line whose email could neither sign in nor receive the digest.
- `chk_users_name_not_blank`: `btrim(name) <> ''`. Refuses a manager with no name to show beside a reply.
- `chk_users_role_outlet`: a brand admin has no outlet and an outlet manager has exactly one. Refuses a manager with no outlet, who would see nothing, and an admin tied to one outlet, whom the outlet scope would wrongly narrow (Q-002, tenet 3).

### `imports`: Imports (hot: no)

One CSV upload and its result, stored under the client's request id so a repeat returns it.

Serves US-01-002. Expected volume: one per upload, tens a year (10^1 to 10^2).

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `id` | `bigint` | No | PK | `identity` | Surrogate key; reviews record the import that brought them in. | HLD section 4 review fix: a review references its import so a bad import can be found. |
| `request_id` | `uuid` | No | UK |  | Client-made id; a repeat returns this result and imports nothing. | Tenet 8: a repeated upload returns the first result with no second row. |
| `file_name` | `text` | No |  |  | Name of the uploaded file, shown with the result. | Screen S-07 success state names the imported file; a repeat must return the same result. |
| `imported_count` | `integer` | No |  | `0` | Rows stored as new reviews; set before the import commits. | AC-US-01-002-1: the admin sees how many reviews were imported. |
| `duplicate_count` | `integer` | No |  | `0` | Rows skipped because the same review was already imported. | HLD flow A: rows already imported are counted, not stored; S-07 shows the skipped count. |
| `rejected_count` | `integer` | No |  | `0` | Rows rejected; each is listed in import_rejections. | AC-US-01-002-3 and Q-023: rejected rows are reported. |
| `created_at` | `timestamptz` | No |  | `now()` | When the import committed. | Rule: audit columns; imports are never updated, so there is no updated_at. |

**Indexes**

- `uq_imports_request_id`: unique on (request_id). A repeated upload with the same id finds the first result and imports nothing (tenet 8, S-07).

**Constraints**

- `chk_imports_counts_not_negative`: all three counts are zero or more. Refuses a result the screen could not show.

### `import_rejections`: Import rejections (hot: no)

Each CSV row an import rejected, with its row number and reason.

Serves US-01-002. Expected volume: zero for a clean file, at most one per row of a bad one (10^0 to 10^3). The primary key (import_id, row_number) is also the foreign-key index.

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `import_id` | `bigint` | No | PK FK imports.id (CASCADE) |  | The import that rejected the row. | AC-US-01-002-3: each rejected row is reported for its import; tenet 8 returns the same list on a repeat. |
| `row_number` | `integer` | No | PK |  | Row number in the file, counted as the admin sees it in a spreadsheet. | AC-US-01-002-3 and Q-023: each rejected row is reported with its row number. |
| `reason` | `text` | No |  |  | Why the row was rejected, worded so the admin can fix it. | AC-US-01-002-3 and Q-023: each rejected row is reported with its reason. |

**Constraints**

- `chk_import_rejections_row_positive`: `row_number >= 1`. Refuses a row number the admin could not find in the file.
- `chk_import_rejections_reason_not_blank`: `btrim(reason) <> ''`. Q-023 says every rejected row is reported with a reason.

### `reviews`: Reviews (hot: no)

One customer review of one outlet, from a CSV import or the seed.

Serves US-01-002, US-01-003, US-01-005 to US-01-009, US-00-001, US-00-002, US-02-006. Expected volume: 1,500 in the seed; about 290 a week for the 25-outlet example, about 15,000 a year (10^3 to 10^4), driven by imports. Read by every dashboard screen, but never near the 10^6 rows that make a table hot.

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `id` | `bigint` | No | PK | `identity` | Integer id; the id the tagging batch sends and every result line must carry. | AC-US-01-004-1 and tenet 4: results are stored by review id; HLD section 8 keeps ids short integers to fit 20 lines under max_tokens 1000. |
| `outlet_id` | `bigint` | No | FK outlets.id (RESTRICT) UK |  | The outlet reviewed, matched from the CSV outlet column. | REQ-003 and AC-US-01-002-2: the stored review holds its outlet; tenet 3 filters every outlet query on it. |
| `import_id` | `bigint` | Yes | FK imports.id (RESTRICT) |  | The import that brought the review in; null for reviews the seed command wrote. | HLD section 4 review fix: review import; the seed (AC-US-02-006-1) writes reviews without an upload. |
| `source` | `text` | No | UK |  | Where the review was written, as the CSV source column says. | REQ-003 and AC-US-01-002-2: the stored review holds its source unchanged. |
| `review_date` | `date` | No | UK |  | Calendar date of the review. | REQ-003; Q-005 and AC-US-01-007-1: weeks run Monday to Sunday; HLD review fix: a calendar date in the brand timezone. |
| `rating` | `smallint` | No |  |  | Star rating, a whole number from 1 to 5. | REQ-003; AC-US-01-005-1 averages it per week; scale 1 to 5 whole stars decided by the product owner on 2026-10-06. |
| `review_text` | `text` | No | UK |  | The review as written, kept unchanged. **(personal data)** | REQ-003 and AC-US-01-002-2; AC-US-01-008-1 searches it; it is sent to the model for tags and drafts. |
| `reviewer_name` | `text` | No | UK |  | Name of the person who wrote the review. **(personal data)** | REQ-003 and AC-US-01-002-2; AC-US-01-008-1 searches it; required, decided by the product owner on 2026-10-06. |
| `created_at` | `timestamptz` | No |  | `now()` | When the review was stored. | Rule: audit columns; reviews are never edited, so there is no updated_at. |

**Indexes**

- `uq_reviews_natural_key`: unique on (outlet_id, source, review_date, reviewer_name, md5(review_text)). The import inserts with `ON CONFLICT DO NOTHING` on this key and counts the skipped rows as "already imported" (HLD flow A, S-07). `md5` keeps the key short for long texts and is immutable, so it is allowed in an index. Its leading column also serves the `outlet_id` foreign-key check.
- `idx_reviews_outlet_id_review_date`: on (outlet_id, review_date). One outlet over a date range: the weekly rating and sentiment trends (AC-US-01-005-1, -2), the two weeks the movers compare (AC-US-01-007-1), the digest's urgent reviews of one week (AC-US-01-009-4) and a manager's review list, newest first (AC-US-00-001-3).
- `idx_reviews_import_id`: on (import_id). The reviews of one import, and the foreign-key index rule.

**Constraints**

- `chk_reviews_rating_range`: `rating BETWEEN 1 AND 5`. Whole stars from 1 to 5 (product owner, 2026-10-06); `smallint` already refuses 3.5, and the CSV connector reports the row as rejected before it reaches the database.
- `chk_reviews_source_not_blank`, `chk_reviews_text_not_blank`, `chk_reviews_reviewer_not_blank`: refuse an empty source, text or reviewer name, because all six CSV fields are required (product owner, 2026-10-06) and a blank text gives the model nothing to tag.

### `review_tags`: Review tags (hot: no)

The one stored tag result of a review. No row means untagged, which is the tagging worker's pending work (ADR-0006).

Serves US-01-003, US-01-004, US-01-005 to US-01-009, US-02-002. Expected volume: one per tagged review (10^3 to 10^4). The primary key on review_id serves the untagged anti-join (`reviews` with no `review_tags` row), so the untagged count and the worker's snapshot need no other index.

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `review_id` | `bigint` | No | PK FK reviews.id (CASCADE) |  | The review the result line named; one result per review. | REQ-011 and AC-US-01-003-6: one stored result per review, never re-tagged; ADR-0006: no row means untagged. |
| `themes` | `text[]` | No |  |  | Theme codes from the configured list; zero, one or several. | AC-US-01-003-2 and Q-021: zero, one or several themes from the list; AC-US-01-006-2 counts each theme once. |
| `sentiment` | `sentiment` | No |  |  | One label: positive, neutral or negative. | AC-US-01-003-4 and Q-013: one sentiment label per review. |
| `is_urgent` | `boolean` | No |  |  | True when the review concerns food safety, harassment or a legal threat; true exactly when urgent_reasons is not empty. | REQ-010 and AC-US-01-003-5; AC-US-01-008-3 and AC-US-01-009-4 list urgent reviews. |
| `urgent_reasons` | `text[]` | No |  | `'{}'` | Why the review is urgent: food_safety, harassment, legal_threat; several allowed, none repeated, empty when not urgent. | REQ-010 names the three reasons; screens S-02, S-04 and S-08 and the digest show the reason beside each urgent review; stored per reason and several allowed (product owner, 2026-10-06). |
| `prompt_version` | `integer` | No |  |  | Number of the tagging prompt version that produced this result. | AC-US-02-002-2 and Q-011: a stored tag names the prompt version that produced it. |
| `created_at` | `timestamptz` | No |  | `now()` | When the result was stored. | Rule: audit columns; a result is never changed, so there is no updated_at. |

**Constraints**

- `chk_review_tags_themes_no_nulls`: `array_position(themes, NULL) IS NULL`. Refuses a null theme, which the heatmap could not place in a column.
- `chk_review_tags_prompt_version_positive`: `prompt_version >= 1`. Versions are numbered from v1 (Q-011).
- `chk_review_tags_urgent_matches_reasons`: `is_urgent = (cardinality(urgent_reasons) > 0)`. Refuses a review flagged urgent with no reason to show, and a review that has a reason but is missing from the urgent filter, the urgent count and the digest (AC-US-01-008-3, AC-US-01-009-4).
- `chk_review_tags_urgent_reasons_allowed`: `urgent_reasons <@ ARRAY['food_safety', 'harassment', 'legal_threat']`. Refuses a reason the screens and the digest have no label for.
- `chk_review_tags_urgent_reasons_distinct`: no null and no reason twice. Refuses a list such as `{food_safety, food_safety}`, which would count one urgent review twice in a per-reason report.

`urgent_reasons` is `text[]` with a CHECK, not an array of an enum type, for the same reason `themes` is: pgx v5 reads and writes `text[]` with no setup, while an array of a custom enum has to be registered on every connection first (ADR-0003). The set is still closed by the CHECK.

`is_urgent` stays a stored column rather than a generated one, so the urgent filter and count keep reading one boolean. Once `chk_review_tags_urgent_matches_reasons` is validated, the two cannot disagree on any row; section 8 names the one case where it is not yet validated.

The database cannot check theme codes against the configured list, because the list is configuration that a developer changes while old tags are kept (Q-012). The validator rejects any code not in the list before storing (AC-US-01-004-3); see the open concerns.

### `replies`: Replies (hot: no)

The draft and the approved reply of a review, at most one per review.

Serves US-00-002, US-00-003, US-01-008, US-02-002. Expected volume: one per review a manager opens, at most one per review (10^3 to 10^4).

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `review_id` | `bigint` | No | PK FK reviews.id (CASCADE) |  | The review replied to; the primary key lets only one request claim the draft. | AC-US-00-002-2: a stored draft is shown again with no model call; HLD flow B claims the row before calling the model. |
| `status` | `reply_status` | No |  | `'drafting'` | drafting while the model call runs, draft while editable, replied once a manager approved it. | AC-US-00-003-2 and -4 and Q-008: replied only through the manager's approval; AC-US-01-008-2 filters on reply status. |
| `draft_text` | `text` | Yes |  |  | The text the model drafted, kept as drafted; null while drafting or when written by hand. **(personal data)** | AC-US-00-002-1: the draft is generated, stored and shown; AC-US-00-002-7: a hand-written reply has no draft. |
| `prompt_version` | `integer` | Yes |  |  | Number of the reply prompt version that drafted draft_text; null without a model draft. | AC-US-00-002-3 and AC-US-02-002-2: the version is stored with the draft. |
| `reply_text` | `text` | Yes |  |  | The text the manager edits and approves; starts as the draft. **(personal data)** | AC-US-00-003-1: the edited text is stored and shown next time; AC-US-00-003-2: replied shows the final text. |
| `replied_by` | `bigint` | Yes | FK users.id (RESTRICT) |  | The outlet manager who marked it replied, which is the approval. | AC-US-00-003-2: the review shows as replied with the manager. |
| `replied_at` | `timestamptz` | Yes |  |  | When the manager marked it replied. | AC-US-00-003-2: the review shows as replied with the time. |
| `created_at` | `timestamptz` | No |  | `now()` | When the draft was claimed or the reply first written. | Rule: audit columns (database postgres reference). |
| `updated_at` | `timestamptz` | No |  | `now()` | Last change; a drafting claim older than 60 seconds may be taken over. | Rule: audit columns; HLD section 8 draft deadline of 45 seconds, so an older drafting claim belongs to a crashed request. |

**Indexes**

- `idx_replies_replied_by`: on (replied_by). The foreign-key index rule, so checking whether a user may be deleted does not scan the table. The reply status filter (AC-US-01-008-2) joins on the primary key.

**Constraints**

- `chk_replies_draft_has_version`: a model draft and its prompt version are present together. Refuses a draft that cannot name the prompt version that made it (AC-US-02-002-2).
- `chk_replies_prompt_version_positive`: `prompt_version >= 1`. Versions are numbered from v1 (Q-011).
- `chk_replies_drafting_is_empty`: a row in `drafting` has no text yet. Refuses a claim that already shows text, which would hide that the model call is still running.
- `chk_replies_replied_complete`: `replied` has its final text, manager and time; any other status has neither manager nor time. Refuses a review shown as replied without the approval (REQ-024, AC-US-00-003-2, -4).

### `digests`: Digests (hot: no)

One generated weekly digest, claimed as `sending` before SMTP is called.

Serves US-01-009. Expected volume: one per press of "Generate digest", about 52 a year (10^1 to 10^2).

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `id` | `bigint` | No | PK | `identity` | Surrogate key. | Rule: every table has a primary key; request_id is the client's, not ours. |
| `request_id` | `uuid` | No | UK |  | Client-made id; a repeat returns this digest and never sends again. | AC-US-01-009-5: one email; tenet 8 and HLD flow C step 3. |
| `status` | `digest_status` | No |  | `'sending'` | sending before SMTP is called, then sent or failed. | HLD flow C step 3: the row is claimed as sending before SMTP; HLD section 7: MailHog down marks it failed. |
| `week_start` | `date` | No |  |  | Monday of the latest complete week the digest covers. | AC-US-01-009-2 and Q-005: the latest complete Monday to Sunday week; S-08 names the week. |
| `recipient_email` | `text` | No |  |  | Address the one email went to: the brand admin. **(personal data)** | AC-US-01-009-5 and Q-006: one email to the brand admin; S-08 shows who it was sent to. |
| `untagged_count` | `integer` | No |  |  | Reviews dated in the two compared weeks that were still untagged. | HLD flow C step 2 review fix: the first line states untagged reviews in the two weeks. |
| `subject` | `text` | No |  |  | Email subject as sent. | S-08 success state shows the email as composed; a repeat returns the recorded digest. |
| `body` | `text` | No |  |  | Email body as composed, quoting urgent reviews. **(personal data)** | AC-US-01-009-3 and -4: movers and urgent reviews with their text; HLD flow C: a repeat returns the recorded digest. |
| `failure_reason` | `text` | Yes |  |  | The SMTP error shown to the admin when sending failed. | HLD section 7: MailHog down, the row is marked failed and the admin sees the error. |
| `sent_at` | `timestamptz` | Yes |  |  | When MailHog accepted the email. | S-08 success state shows the time the digest was sent. |
| `created_at` | `timestamptz` | No |  | `now()` | When the digest was generated and claimed. | Rule: audit columns; AC-US-01-009-1: generated at that moment. |
| `updated_at` | `timestamptz` | No |  | `now()` | Last status change. | Rule: audit columns (database postgres reference). |

**Indexes**

- `uq_digests_request_id`: unique on (request_id). A repeated request finds the first digest and never sends a second email (AC-US-01-009-5, tenet 8).

**Constraints**

- `chk_digests_week_start_monday`: `EXTRACT(ISODOW FROM week_start) = 1`. Weeks run Monday to Sunday (Q-005); refuses a digest that covers some other seven days.
- `chk_digests_untagged_not_negative`: `untagged_count >= 0`. Refuses a count the first line could not state.
- `chk_digests_status_fields`: `sending` has neither time nor error, `sent` has its time, `failed` has its error. Refuses a digest shown as sent with no time, or failed with nothing to tell the admin.

### `model_calls`: Model calls (hot: no)

Every live model call with its tokens and cost, in the `budget` schema. The running total is `sum(coalesce(settled_cost_usd, reserved_cost_usd))` over all rows, read immediately before each live call (tenet 2).

Serves US-02-001, US-02-002, US-02-004, US-02-005. Expected volume: one per live call, bounded by the budget: about 1,050 tagging batches or about 4,000 drafts before USD 8 (HLD section 8), so 10^3 to 10^4 for the product's life. One row per HTTP attempt: a 429 or 5xx retry (HLD section 8) is a new row, because an earlier attempt may also have been billed. Replayed calls write nothing (AC-US-02-003-1); tests that insert rows to check the USD 8 stop run on their own database (section 12).

| Column | Type | Null | Key | Default | Description | Why |
| --- | --- | --- | --- | --- | --- | --- |
| `id` | `bigint` | No | PK | `identity` | Surrogate key; the gateway settles the row it reserved by this id. | HLD section 3: a cost row is reserved before the call and settled after it. |
| `purpose` | `budget.model_call_purpose` | No |  |  | tagging, drafting, evaluation, tone_check, or reconciliation for a start-up adjustment. | AC-US-02-001-1: tagging, drafting and evaluation calls all pass the gateway; AC-US-02-004-5: evaluation cost is in the total. |
| `model` | `text` | Yes |  |  | Model identifier sent to OpenRouter; null only on a reconciliation row. | AC-US-02-001-2: a log row records the model. |
| `prompt_version` | `integer` | Yes |  |  | Number of the prompt version sent; the purpose says which prompt. | HLD section 10: the cost log is the audit of every call and its prompt version (Q-011). |
| `input_tokens` | `integer` | Yes |  |  | Input tokens OpenRouter reported; null until the call settles. | REQ-030 and AC-US-02-001-2: a log row records input tokens. |
| `output_tokens` | `integer` | Yes |  |  | Output tokens OpenRouter reported; null until the call settles. | REQ-030 and AC-US-02-001-2: a log row records output tokens. |
| `reserved_cost_usd` | `numeric(12,8)` | No |  |  | Worst-case price reserved before the call; counts until settled. | REQ-031 and Q-015 with HLD section 3: a timed-out or cancelled call stays counted at its reserved price. |
| `settled_cost_usd` | `numeric(12,8)` | Yes |  |  | usage.cost from the response, in USD; replaces the reserved price in the total. | REQ-030 and AC-US-02-001-2: a log row records the cost in USD. |
| `outcome` | `budget.model_call_outcome` | No |  | `'reserved'` | reserved while in flight, settled with a response, failed without one. | HLD section 3: the running total sums settled cost, or reserved cost where not settled. |
| `created_at` | `timestamptz` | No |  | `now()` | When the call was reserved, which is the time the log shows. | AC-US-02-001-2: a log row records the time. |
| `updated_at` | `timestamptz` | No |  | `now()` | When the call settled or failed. | Rule: audit columns (database postgres reference). |

**Constraints**

- `chk_model_calls_costs_not_negative`: both costs are zero or more. Refuses a row that would lower the running total.
- `chk_model_calls_tokens_not_negative`: token counts are zero or more.
- `chk_model_calls_settled_shape`: a settled call has its cost and both token counts. Refuses a settled row that leaves AC-US-02-001-2 without tokens or cost.
- `chk_model_calls_reconciliation_shape`: a reconciliation row has no model or prompt version and is settled; every other row names its model. Refuses a call row without the model AC-US-02-001-2 asks for.

No index: the running total sums at most about 10^4 rows before each call, well under a millisecond's work at this size.

**Start-up reconciliation.** The HLD keeps the larger of the local total and the usage OpenRouter reports for the key (section 3). To keep that figure in the database (tenet 2), the gateway inserts one `reconciliation` row for the difference when the provider's usage is higher: `reserved_cost_usd` and `settled_cost_usd` both equal the difference, both token counts are 0, `outcome` is settled. Nothing is inserted when the local total is equal or higher. Only the server reconciles, once at start; the seed, eval and tone-check commands never do, so two inserts of the same difference cannot happen. The total stays one sum over one table.

## 5. Writers, claims and derived rows

Every path that writes a table, and where each NOT NULL value comes from:

| Table | Writer paths | Values from |
| --- | --- | --- |
| outlets | admin add form; seed | name typed by the admin, trimmed, or the seed's 5 names |
| users | users file upsert at server start; the seed's own upsert of the file it writes | every column from the file line; `removed_at` set for emails no longer in the file and cleared for emails back in it |
| imports, import_rejections | CSV connector, in one transaction with the reviews | request id from the client; counts start at 0 and are set from the row checks before commit |
| reviews | CSV connector; seed command | all six CSV fields, all required; the seed generates all six; `import_id` null for the seed |
| review_tags | tagging worker; seed command (same validator, same advisory lock) | the validated result line (themes, sentiment, urgent reasons; `is_urgent` set from whether any reason is present) and the current tagging prompt version |
| replies | draft claim; draft lands; hand-written reply; edit; mark replied | status from the step; draft and version from the gateway; text from the manager; `replied_by` from the signed-in user |
| digests | digest builder | week, count, subject and body computed before the claim; recipient from the active brand admin |
| model_calls | model gateway (reserve, settle, fail); start-up reconciliation in the server only | model, prompt version and worst-case price before each HTTP attempt; tokens and `usage.cost` after it; for reconciliation, the difference and zero tokens |

**Outlet scope on writes.** Every write a manager makes names the review and the caller's outlet together, taken from the user row re-read on this request (tenet 3), so a review id from another outlet changes nothing and spends nothing (AC-US-00-001-4, AC-US-00-003-3):

- draft claim: `INSERT INTO replies (review_id) SELECT id FROM reviews WHERE id = $review AND outlet_id = $scope ON CONFLICT DO NOTHING RETURNING updated_at`;
- edit, hand-written reply and mark replied: the `UPDATE replies` joins `reviews` and filters `outlet_id = $scope`, or the `INSERT` selects from `reviews` the same way.

One integration test per write endpoint uses another outlet's review id and expects a refusal and no row.

**Reply transitions.** Allowed: none to `drafting` (claim), none to `draft` (hand-written reply when drafting is unavailable), `drafting` to `draft` (the draft lands), `draft` to `draft` (edit), `draft` to `replied` (mark replied). Nothing leaves `replied`, and a replied row's text never changes, so the approved text is final (REQ-024). Every update names the status it expects in its `WHERE`.

**Edits by two managers.** An outlet may have several managers, and one manager may have two tabs open. Edit and mark replied carry the `updated_at` the client last read: `UPDATE replies ... WHERE review_id = $1 AND status = 'draft' AND updated_at = $seen`. No match on a replied row returns that row unchanged, which is the idempotent repeat of mark replied; no match otherwise is a conflict (409) and the client reloads. A manager therefore approves only text they have seen (AC-US-00-003-2).

**Claims and expiry.** All checked inside the writing statement; no index or constraint calls `now()`.

- Draft claim: the claim returns its `updated_at`, which is the claim's token. Only the winner calls the gateway. The draft landing (`drafting` to `draft`) and the failure delete both also match `updated_at = $claimed`, so a request whose claim was taken over writes nothing. If the gateway fails or the budget stop is active, the winner deletes its own claim so the manager can write by hand or retry. A request that crashed leaves a `drafting` row; a later open takes it over with `UPDATE ... SET updated_at = now() WHERE status = 'drafting' AND updated_at < now() - interval '60 seconds' RETURNING updated_at`. The 45 second draft deadline (HLD section 8) is one context that wraps every gateway attempt and the landing write, so a live request never runs past 45 seconds and an abandoned claim blocks drafting for at most 60 seconds. A paused laptop can stretch the Go deadline beyond the database clock; the token match means the late request then writes nothing.
- Digest claim: inserted as `sending` under its request id before SMTP. A row left in `sending` by a crash stays so; a repeat of that request id returns it and sends nothing (HLD section 7).
- Model call reservation: inserted as `reserved` before each HTTP attempt, then settled or failed. A crash leaves it `reserved`, and it keeps counting at its worst-case price (HLD section 4).
- Import: the `imports` row (counts at 0), its rejections and the new reviews are written in one transaction, then the counts are set before commit, so a repeat of the request id waits on that row and then returns the committed result.

**Users file upsert.** One transaction at server start, and the same function in the seed: mark every email not in the file removed first, then upsert each line with `removed_at = NULL`. The server refuses to start unless the file holds exactly one `brand_admin` line, so `uq_users_one_brand_admin` never sees two active admins and the installation is never left without one.

**Seed reset.** The seed command takes the tagging advisory lock before it truncates and holds it until its own tagging ends, so a server tagging pass can never store a result line for an old review id on a new review with the same id (tenet 4). It truncates the domain tables in one statement, restarts the `reviews` identity only (review ids, batch order and texts must match the recordings, HLD Flow D), and upserts the users from the file it writes in the same transaction, so accounts exist at once (AC-US-02-006-5). User ids are never reused, so a cookie issued before the reset names no user and its holder is signed out (ADR-0007).

Derived rows follow their parent: a review's tag result and reply are deleted with it (CASCADE). No PRD path deletes a review; the seed reset truncates all domain tables together. A changed tagging prompt or theme list never touches stored tags (Q-011, Q-012).

Time: review dates and `digests.week_start` are calendar dates. Weeks are computed from the brand timezone in configuration (default Asia/Kolkata, HLD section 3), and every `timestamptz` is stored in UTC.

## 6. Enumerations

| Name | Values | Why |
| --- | --- | --- |
| `user_role` | `brand_admin`, `outlet_manager` | The two personas (PRD section 4, Q-002); a new role is a migration and a change to every outlet-scope query. |
| `sentiment` | `positive`, `neutral`, `negative` | One label per review (Q-013); fixed by the decision, used by trends, the movers' negative count and the list filter. |
| `reply_status` | `drafting`, `draft`, `replied` | Q-008 makes approval and marking replied one action, so there is no `approved` state; `drafting` is the claim (HLD flow B). |
| `review_tags.urgent_reasons` (CHECK, not an enum type) | `food_safety`, `harassment`, `legal_threat` | The three urgent categories REQ-010 names, each shown on screen and in the digest. A new reason is a migration that rewrites both `chk_review_tags_urgent_reasons_allowed` and `chk_review_tags_urgent_reasons_distinct` (each names the values), plus a new tagging prompt version. |
| `digest_status` | `sending`, `sent`, `failed` | The claim-before-send states (HLD flow C, section 7). |
| `budget.model_call_purpose` | `tagging`, `drafting`, `evaluation`, `tone_check`, `reconciliation` | Every caller of the gateway (AC-US-02-001-1) plus the start-up adjustment; a new caller is a migration in the budget set. |
| `budget.model_call_outcome` | `reserved`, `settled`, `failed` | The reserve-then-settle lifecycle (HLD section 3). |

## 7. Retention and personal data

Decided by the product owner on 2026-10-06: personal data is kept, with no automatic deletion. The product runs locally on demo data and the PRD has no delete path. Revisit if real customer reviews are ever loaded.

| Table | Rule | Mechanism | Source |
| --- | --- | --- | --- |
| `outlets`, `imports`, `import_rejections`, `reviews`, `review_tags`, `replies`, `digests` | Kept | none; a developer's seed reset truncates them | product owner, 2026-10-06 |
| `users` | Kept; marked removed when gone from the users file | users file upsert sets `removed_at` | product owner, 2026-10-06; AC-US-00-003-2 |
| `budget.model_calls` | Kept forever | not truncated by the seed; budget migrations have no Down | REQ-031, Q-015, HLD section 4 |

Personal-data columns (8): `users.email` (contact), `users.name` (name), `reviews.reviewer_name` (name), `reviews.review_text` (free text that may name people), `replies.draft_text` and `replies.reply_text` (may greet the reviewer by name), `digests.recipient_email` (contact), `digests.body` (quotes urgent reviews). `users.password_hash` is a credential, not personal data; it never leaves the server. Data leaves the machine only to OpenRouter (HLD section 9).

## 8. Migration plan

goose SQL migrations under `db/migrations/` as ADR-0003 lays out; the repository has none yet, so numbering starts at 00001. The budget schema has its own directory and version table, and no Down (HLD section 4). Each Up starts with `SET lock_timeout = '2s'; SET statement_timeout = '60s';`. Every table is new and empty when created, so there is no lock risk and no backfill. `db-migration` writes the files; this plan writes none.

| # | db-migration name | Phase (expand \| migrate \| contract) | Hot table | Lock risk and batch note |
| --- | --- | --- | --- | --- |
| 1 | 00001_create_outlets_and_users (`user_role`, `outlets`, `users`) | expand; HLD build phase 1 | no | new tables, none |
| B1 | budget/00001_create_model_calls (`budget` schema, its two enums, `model_calls`); no Down | expand; HLD build phase 2 | no | new table, none |
| 2 | 00002_create_imports_reviews_and_tags (`sentiment`, `imports`, `import_rejections`, `reviews`, `review_tags`, their indexes) | expand; HLD build phase 3 | no | new tables, none |
| 3 | 00003_create_replies (`reply_status`, `replies`) | expand; HLD build phase 5 | no | new table, none |
| 4 | 00004_create_digests (`digest_status`, `digests`) | expand; HLD build phase 6 | no | new table, none |

`urgent_reasons` and its three constraints are part of migration 2, because no migration has been written or applied yet.

If migration 2 is already applied when this reaches the build, the change ships instead as 00005_add_review_tags_urgent_reasons, in the same deploy as the tagging worker that writes reasons, never before it. A CHECK added `NOT VALID` still refuses new rows, so a worker that does not write reasons would have every urgent result refused and retried.
- Up: add the column with its constant default `'{}'` (metadata only), then the three CHECKs, `chk_review_tags_urgent_matches_reasons` added `NOT VALID` and the other two validated at once (every existing list is empty, so they pass).
- Validate: `VALIDATE CONSTRAINT chk_review_tags_urgent_matches_reasons` only once no row has `is_urgent` true with an empty list. On demo data, that means a seed reset and a seed run in record mode (about USD 0.57, HLD section 8). On real data, REQ-011 forbids re-tagging, so the constraint stays `NOT VALID` for old rows, which keep `is_urgent` true with no reason, unless the product owner decides otherwise.
- Down: drop the three constraints, then the column.

`review_tags` stays far below the hot-table size, so there is no lock risk.

HLD build phase 4 (dashboard and search) needs no migration: its one index, `idx_reviews_outlet_id_review_date`, is created with the table in migration 2.

## 9. Rules and deviations

Rules checked: 18: 17 against `database/references/postgres.md` and ADR-0003's migration layout. Deviations: 8.

Followed: snake_case plural names; NOT NULL by default; enums only for sets fixed by a decision; money as `numeric`; every foreign key with an explicit ON DELETE and an index on the referencing column (a primary key that leads with it counts); composite index order (equality, then range); expression indexes for `lower()`; advisory lock for one tagging worker (ADR-0006); migration lock and statement timeouts (section 8); keyset pagination for the review list, left to `openapi-spec` and `low-level-design`.

- deviation: ids that leave the system should be UUID v7. Review ids appear in URLs and are sent to the model as `bigint` identities, because the HLD keeps them short integers so 20 result lines fit under max_tokens 1000 (section 8), and every caller is signed-in staff behind the outlet scope. Revisit if any id is shown outside the product.
- deviation: `updated_at` on every table. `imports` (whose counts are set once inside the transaction that creates it), `import_rejections`, `reviews` and `review_tags` are never updated after they commit, so they carry `created_at` only (`import_rejections` takes its time from its import). Revisit if an edit path is added.
- deviation: money carries a `currency` column. Every cost is USD by requirement (REQ-031), so the column name says `_usd` instead. Revisit if a second provider bills in another currency.
- deviation: `tenant_id` first in every composite key, and row-level security. One brand per installation (Q-001) means one tenant; the outlet scope is the boundary instead, applied in every query (tenet 3).
- deviation: free text is `text` with a length CHECK. No story or decision sets a length for review text, names or file names, so no limit is invented here; see the open concerns.
- deviation: a separate API role with no DDL rights and `REVOKE ALL ON SCHEMA public FROM PUBLIC`. The product runs locally in one Docker container with one database user (PRD section 6). Revisit before it runs anywhere shared.
- deviation: ADR-0003 says every goose migration has an Up and a Down. The budget set has no Down, so no back-out can drop the running total (HLD section 4, REQ-031). The ADR is not edited; HLD section 4 already records the exception.
- deviation: backups with point-in-time recovery. Local demo data is re-created by the seed; the budget record is the one thing to keep, and OpenRouter's own key usage is its backstop (start-up reconciliation). Revisit before real data is loaded.

Query shapes without an index: 5, each a sequential scan over at most about 10^4 rows, accepted at this size (HLD section 8: "PostgreSQL is not a limit"):

- text search `ILIKE '%term%'` over `review_text` and `reviewer_name` (AC-US-01-008-1); the fix at scale is a `pg_trgm` GIN index;
- the theme filter and heatmap counts over `review_tags.themes` (AC-US-01-006-1, AC-US-01-008-2); the fix is a GIN index on `themes`;
- the urgent filter and urgent count (AC-US-01-008-3); the fix is a partial index `WHERE is_urgent`;
- the brand admin's review list across all outlets, newest first (AC-US-01-008-1); the fix is an index on (review_date, id);
- the running total over `budget.model_calls` before each live call (tenet 2).

## 10. What the review found

Reviewed by: critic, 2026-10-06 (graded mode; read this document, schema.sql, the dictionary, the backlog, the PRD, the register, HLD sections 2 to 9 and 16, ADR-0003, ADR-0006, ADR-0007 and the tenets). Its verdict: no BLOCKER, every criterion can be served; approve after the three MAJOR findings are fixed. Findings are summarised; each fix below is in this version.

Findings: BLOCKER 0, MAJOR 3, MINOR 4, NIT 1 (open 0).

### MAJOR: budget tests would write to a table nothing may delete (`model_calls`)

The USD 8 tests (AC-US-02-001-4 to -6) must insert cost rows. Run against the demo database, they would leave a USD 8.01 total that no seed reset and no migration removes, and every later live call would be refused.

**Fix:** section 12 says database tests run on a database created for that run, and a test helper refuses to run against the demo database; section 4 says replay writes no rows. Status: fixed in this version. HLD section 12 phase 0 is listed downstream to say the same.

### MAJOR: marking replied could approve text the manager never saw (`replies`)

With several managers per outlet (or two tabs), manager B could mark replied a text manager A had just edited, and a replied row's text could still change.

**Fix:** section 5: edit and mark replied match the `updated_at` the client read and the expected status; a stale write is a 409; nothing leaves `replied` and its text never changes. Status: fixed in this version.

### MAJOR: the seed reset ran outside the tagging lock and reused ids (`reviews`, `review_tags`, `users`)

A server tagging pass in flight during the reset could store an old review's result on a new review with the same id; accounts were empty until the server restarted; and reused user ids let an old cookie resolve to someone else.

**Fix:** section 5, Seed reset: the seed holds the tagging lock from before the truncate until its tagging ends, restarts only the `reviews` identity, and upserts the users in the same transaction; user ids are never reused. Status: fixed in this version. HLD Flow D is listed downstream.

### MINOR: the drafting claim had no owner (`replies`)

A request whose claim was taken over could still delete or overwrite the new claim, paying for two drafts and losing one; the gateway's retries could outlast the 45 second deadline.

**Fix:** section 5: the claim's `updated_at` is its token, matched by the landing write and the failure delete; the 45 second deadline wraps every gateway attempt. Status: fixed in this version.

### MINOR: outlet scope and status transitions were in no listed predicate (`replies`, `review_tags`)

Neither table carries `outlet_id`, so a write by review id alone could claim, edit or approve another outlet's reply.

**Fix:** section 5, Outlet scope on writes and Reply transitions, with one cross-outlet test per write endpoint. Status: fixed in this version.

### MINOR: two writers had no source for a NOT NULL column (`imports`, `model_calls`)

Import counts are known only after the reviews are inserted, and the reconciliation row's token counts and reserved cost were unstated; a command reconciling beside the server could double the adjustment.

**Fix:** import counts default to 0 and are set before commit (schema and section 2); the reconciliation row's values are stated, and only the server reconciles (section 4). Status: fixed in this version.

### MINOR: the users file upsert could leave no brand admin (`users`)

Changing the admin's email in the file could trip `uq_users_one_brand_admin` and end with zero active admins; nothing cleared `removed_at` for a returning email.

**Fix:** section 5, Users file upsert: mark absent emails removed first, upsert with `removed_at = NULL`, and refuse to start unless the file has exactly one brand admin. Status: fixed in this version.

### NIT: four consistency gaps

A name with trailing spaces passed the unique index; the budget set's missing Down was not listed as a deviation from ADR-0003; HLD section 2 still says one manager per outlet; no SMTP timeout is set.

**Fix:** `chk_outlets_name_trimmed` added; the ADR-0003 deviation added to section 9; the HLD wording and the SMTP timeout are open concerns below. Status: fixed in this version (the two outside this document are tracked in section 11).

**Weakest claims, as the critic named them, and where each now stands.**

1. "An abandoned claim blocks drafting for at most 60 seconds": now backed by the claim token and the 45 second context around every attempt; the falsifying spike (pause a process for 70 seconds, take the claim over, resume, count gateway calls) belongs in the US-00-002 tests.
2. "Replayed calls write nothing, so tests leave the total unchanged": true only on a database created for the test run (section 12).
3. "The seed tags under the same advisory lock": now the lock is taken before the truncate (section 5).

### Review of v2 (urgent reasons)

Reviewed by: critic, 2026-10-06, on the changed sections only. Findings: BLOCKER 0, MAJOR 0, MINOR 4, NIT 1 (open 0).

- **MINOR: the fallback 00005 could not finish its own validate step and would break a worker that writes no reasons.** Fix: section 8 now ships 00005 with the reason-writing worker, names its Down, and states the end state for demo and real data. Status: fixed in this version.
- **MINOR: pgx v5 cannot write an array of a custom enum without registering it first.** Fix: `urgent_reasons` is `text[]` with `chk_review_tags_urgent_reasons_allowed`, like `themes`, and section 4 says why. Status: fixed in this version.
- **MINOR: the distinct CHECK names the three values, so a fourth reason would slip past it.** Fix: section 6 says a new reason rewrites both value CHECKs. Status: fixed in this version.
- **MINOR: no acceptance criterion fails if the reasons are wrong or never shown** (AC-US-01-003-5 tests the flag; AC-US-01-009-4 lists urgent reviews with outlet, date and text only). Fix: tracked in section 11 for a backlog revision. Status: fixed in this version (tracked).
- **NIT: the OpenAPI gap had no owner.** Fix: tracked in section 11. Status: fixed in this version (tracked).

## 11. Open concerns

Each concern: the tag, the table, the consequence, an owner and date, and whether it blocks development.

- **[conflict]** HLD Flow D says the seed truncates "with identity counters restarted" and takes the lock to tag afterwards; this model takes the lock before the truncate, restarts only the `reviews` identity and upserts the users itself (section 5). (`reviews`, `users`) Owner: product owner, through an HLD revision, before build phase 7. Blocks development: no.
- **[scope]** Downstream wording this model makes stale, none of it edited here: HLD section 2 and AC-US-02-006-5 say one manager per outlet (several are allowed, product owner 2026-10-06); HLD section 4 user retention "kept while in the users file" (now kept and marked removed); HLD section 12 phase 4 "indexes only" (none needed); HLD section 12 phase 0 must name the per-run test database (section 12). Owner: product owner, through an HLD revision, 2026-10-06. Blocks development: no.
- **[gap]** No story or decision sets a length for review text, reviewer names, outlet names or file names, so the database accepts any length and the model receives the first 2,000 characters (HLD section 8). A CSV cell of several megabytes would be stored whole. (`reviews`, `outlets`, `imports`) Owner: product owner, with `openapi-spec` setting the upload size limit, before build phase 3. Blocks development: no.
- **[risk]** The database cannot check theme codes against the configured list, because the list is configuration and old tags are kept when it changes (Q-012). A duplicate code in one array would count twice in the heatmap (AC-US-01-006-2). The validator must reject unknown and repeated codes before storing (AC-US-01-004-3). (`review_tags`) Owner: developer, in `low-level-design`, build phase 3. Blocks development: no.
- **[risk]** Five query shapes have no index (section 9) and are sequential scans over at most about 10^4 rows. Revisit when `reviews` passes 10^5 rows or any dashboard query takes over 200 ms in the request log. (`reviews`, `review_tags`, `model_calls`) Owner: developer, build phase 4. Blocks development: no.
- **[gap]** No SMTP timeout is set, so a digest left in `sending` cannot be told apart from one still in flight (DigestSendFailed, HLD section 10). (`digests`) Owner: developer, in `low-level-design`, build phase 6. Blocks development: no.
- **[gap]** No path deletes an outlet, an import or a wrong review (HLD section 17); the RESTRICT and CASCADE rules here are ready for one. (`outlets`, `imports`, `reviews`) Owner: product owner, through `prd` if wanted, after the MVP. Blocks development: no.
- **[ambiguity]** The PostgreSQL major version is not pinned in the PRD or the HLD. This model is proven on PostgreSQL 16 and needs nothing newer. Owner: developer, when `new-repo` writes the Docker setup, build phase 0. Blocks development: no.

- **[scope]** Storing urgent reasons (v2) needs matching changes outside this document, none made here: OpenAPI `Tags` gains `urgent_reasons` so screens S-02, S-04 and S-08 can show the reason; the backlog adds the reason to AC-US-01-003-5 (tagging) and AC-US-01-009-4 (digest); the HLD section 4 review tag result row names it. (`review_tags`) Owner: product owner, through `openapi-spec`, `backlog` and `high-level-design` revisions, before build phase 3. Blocks development: no.

## 12. Applying this

`schema.sql` runs top to bottom in one transaction against an empty database: the `budget` schema and enum types first, then tables in foreign-key order. It is the reference for the whole model; the goose migrations in section 8 split it by build phase. Every change after the first release is its own migration, never an edit to this file.

Tests: every test that touches the database runs on a database created for that run (CI's fresh service container; locally, a database the test harness creates and drops), never on the demo database. A test helper refuses to run when `current_database()` is the demo database's name, because the USD 8 tests insert cost rows into `budget.model_calls`, which nothing may delete (REQ-031).

- Gate: `data-model: 9 tables, 71 columns, 10 indexes, 28 checks, 6 enums, 8 personal-data columns, 0 problems` (v2)
- Applied to an empty Postgres: yes, `schema-apply: docs/design/schema.sql applied to postgres:16, 8 tables` (the count is the `public` schema; `budget.model_calls` is the ninth)
- Probed on postgres:16 with good and bad rows, 2026-10-06: 15 bad rows refused, each by the named index or constraint meant to refuse it; a second draft claim on the same review inserted nothing; a manager who approved a reply could not be deleted; the budget rows survived the seed's truncate of the domain tables.
- Probed again for v2 on postgres:16, 2026-10-06: 4 valid tag rows accepted (no reason, one, two, all three); 7 refused, each by the rule meant to refuse it (urgent with no reason, a reason without urgent, a repeated reason, a null reason, an unknown reason, a null list, clearing the reasons of an urgent row).

## Revision history

- v1 (2026-10-06, commit 88dec6b): first version.
