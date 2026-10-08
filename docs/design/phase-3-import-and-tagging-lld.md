# Low Level Design: CSV import and tagging, build phase 3

- Task: none (no task ids yet). HLD: [review-intelligence-hld.md](review-intelligence-hld.md) sections 2 (flow A), 3, 4, 7, 8 and 12 (phase 3). ADRs: [0003](../adr/0003-use-sqlc-pgx-and-goose-for-data-access.md), [0005](../adr/0005-use-rest-with-openapi-for-the-api.md), [0006](../adr/0006-run-tagging-in-process-from-untagged-rows.md), [0008](../adr/0008-tag-and-draft-with-prompt-only-haiku-calls.md)
- Author: unattributed (no `.bearing/company.json`), 2026-10-08, status Draft
- Version: v1
- Companions: [data-model.md](data-model.md) (migration 2, sections 4 and 5), [GenAI solution](../genai/review-classification-and-replies-solution.md) section 4.1 (the tagging call, the line format and its validation), [phase 2 LLD](phase-2-model-gateway-lld.md) (the gateway this phase calls), screen `app/S-07`

Serves: US-01-002, US-01-003, US-01-004, REQ-002 to REQ-014, Q-011 to Q-014, Q-021 to Q-023, tenets 1, 4, 5 and 8.
Acceptance criteria the tests below prove: AC-US-01-002-1 to -6, AC-US-01-003-1 to -6, AC-US-01-004-1 to -5, AC-US-02-002-2 (a stored tag names its prompt version), and AC-US-01-001-1's second half (a new outlet can be the outlet of imported reviews). AC-US-01-003-5 and -7 are proved on recorded or fixture answers only: whether the model itself flags Hinglish food-safety complaints is measured by the evaluation in phase 7, never by replay.

Decisions this design rests on:

| Point | Decision | Source |
| --- | --- | --- |
| Tagging approach | Prompt only, Claude Haiku 4.5, one line per review, validated per review id in code, missing and invalid ids retried | ADR-0008, accepted by the product owner 2026-10-08 |
| Request settings | `max_tokens: 1000`, `temperature: 0`, no `reasoning` field (OpenRouter enables Claude thinking only when `reasoning` is sent) | OpenRouter model page lists `temperature`, `max_tokens` and `reasoning` as accepted for `anthropic/claude-haiku-4.5`; reasoning docs (checked 2026-10-08) |
| Determinism | Temperature 0 lowers variation but does not make answers repeatable: Anthropic states identical inputs may still produce different outputs. Repeatability comes from recordings (replay), and the validator never assumes a stable answer | Anthropic API docs, checked 2026-10-08 |
| Theme list | A Go list in `internal/tagging/themes.go`, edited by a developer in the same merge request as a new tagging prompt version; a test fails when the current prompt does not name every configured code | Q-012 ("developer-edited configuration"), GenAI 4.3 |
| Deferred to phase 4 | `GET /tagging/status`, `GET /themes` and the top bar's tagging indicator: their readers (overview banners, filters, heatmap) arrive there. Phase 3 shows progress through the outlets list's `untagged_count` | HLD section 12 lists them under no phase; their x-story-ids include phase 4 stories |

## 1. Scope

The CSV import (connector interface, CSV connector, Google connector declared only, de-duplication, rejected rows, `POST /api/v1/imports`) and the tagging worker (one goroutine in the server, advisory lock, one pass over a snapshot of untagged reviews in batches of 20, validation per review id, bounded retries), plus the S-07 Import screen and the outlet counts on S-06. The seed's own tagging (phase 7) reuses the worker's pass and lock; it is not built here.

## 2. Module layout

| File | Owns | Est. lines |
| --- | --- | --- |
| `db/migrations/00002_create_imports_reviews_and_tags.sql` (new) | `sentiment`, `imports`, `import_rejections`, `reviews`, `review_tags`, their indexes; copied from `schema.sql` | 120 |
| `db/queries/imports.sql`, `db/queries/tagging.sql` (new); `db/queries/outlets.sql` | import writes and reads; snapshot, texts, tag insert; outlet counts | 110 |
| `internal/store/imports.go`, `internal/store/tagging.go` (new); `internal/store/outlets.go` | the transaction, the lock on a dedicated connection, domain types out | 220 |
| `internal/connector/connector.go`, `csv.go`, `google.go` (new) | `Connector`, `Row`, `Rejection`, `FileError`; CSV parsing and row checks; Google declared only | 250 |
| `internal/imports/service.go` (new) | the import flow: role, repeat lookup, fetch, outlet match, store, signal | 150 |
| `internal/httpapi/imports_handler.go` (new); `errors.go`, `server.go` | multipart reading, `Idempotency-Key`, 413, 415, the error mapping | 160 |
| `prompts/tagging/v1.md`, `prompts/tagging/current`, `prompts/embed.go` (new) | the tagging prompt v1 and the `//go:embed` that feeds `Parse` | 80 |
| `internal/tagging/themes.go`, `message.go`, `parse.go`, `worker.go` (new) | theme list; the user message; the line parser and validator; the worker and its pass | 380 |
| `internal/config/config.go`, `cmd/api/main.go` | `TAGGING_ENABLED`; build the registry and worker, signal at start, stop on shutdown | 60 |
| `web/src/lib/api.ts`; `web/src/features/imports/*` (new); `web/src/app/routes.tsx`, `web/src/components/AppShell.tsx` | multipart upload; S-07; the Import route and nav item for the brand admin | 330 |

## 3. Types and boundaries

- `connector.Connector`: `Name() string` and `Fetch(ctx) (Batch, error)`. `Batch` holds `Rows` (row number, outlet name, source, date, rating, text, reviewer name) and `Rejections` (row number, reason). A whole-file problem is a `*connector.FileError` (field, reason, message). The import flow takes any `Connector`, so a second source implements it without changing the flow (AC-US-01-002-4).
- `connector.Google`: implements `Connector`; `Fetch` returns `ErrNotImplemented`, holds no client and makes no network call (AC-US-01-002-5).
- `connector.NewCSV(r io.Reader)`: the one place a CSV row is validated (section 4). `api/openapi.yaml` `ImportResult` is the response shape; the handler maps the import result to it.
- `tagging.Result`: review id, theme codes (in list order), sentiment, urgent reasons (in the order `food_safety`, `harassment`, `legal_threat`, which the API returns), `IsUrgent` (true exactly when any reason is present). Built only by `tagging.ParseAnswer`.
- `tagging.Model`: `Complete(ctx, gateway.Request) (gateway.Response, error)`; `*gateway.Gateway` is the only production value (tenet 1).

## 4. CSV rules

- UTF-8, a leading byte-order mark dropped. Header names compared trimmed and ignoring case; the six columns `outlet, source, date, rating, text, reviewer_name` are required, others ignored. A missing or repeated column, an empty file or unreadable CSV (a bare quote) is `FileError`: 400 `csv_invalid`, nothing stored, `details` names each missing column.
- Rows are numbered as a spreadsheet numbers them: the header is row 1. One reason per rejected row, the first rule it breaks, in this order: field count; not UTF-8 or holds a NUL byte (PostgreSQL text refuses NUL); outlet blank; source blank or over 200 characters; date blank or not `YYYY-MM-DD`; rating blank or not a whole number 1 to 5; text blank or over 5,000 characters; reviewer name blank or over 200 characters; then, in the import flow, an outlet name that matches no outlet ignoring case. A quoted value in a reason is cut at 60 characters.
- Values are stored unchanged (AC-US-01-002-2); blank means empty after trimming. The 5 MB cap counts file bytes: 413 `file_too_large`.

## 5. Import flow

```mermaid
sequenceDiagram
    participant UI as S-07
    participant H as imports handler
    participant S as imports.Service
    participant DB as PostgreSQL
    participant W as tagging worker
    UI->>H: POST /api/v1/imports (multipart, Idempotency-Key)
    alt not signed in / manager / cross-site
        H-->>UI: 401 / 403 role_not_allowed / 403 cross_site_request
    else key missing or not a UUID, or no file part
        H-->>UI: 400 malformed_request
    else not multipart
        H-->>UI: 415 unsupported_media_type
    end
    H->>S: Import(user, key, file name, CSV connector)
    S->>DB: import by request id
    alt already imported under this key
        S-->>H: the stored result (201, no second import)
    end
    S->>S: Fetch rows
    alt FileError / over 5 MB
        S-->>H: 400 csv_invalid / 413 file_too_large
    end
    S->>DB: outlets, matched by name ignoring case
    S->>DB: BEGIN; insert import ON CONFLICT (request_id) DO NOTHING
    alt no row: the same key committed meanwhile
        S->>DB: ROLLBACK; read the stored result
    else
        S->>DB: insert reviews ON CONFLICT (natural key) DO NOTHING RETURNING id; insert rejections; set counts; COMMIT
        S->>W: Signal() when imported > 0
    end
    S-->>H: result
    H-->>UI: 201 ImportResult
```

Counts: imported is the number of ids returned, duplicates the valid rows that returned none (already stored, or repeated within the file), rejected the rejections. The reviews insert is one statement over arrays, in file order, so review ids follow the file. A store error rolls the whole import back: 500 `internal`.

## 6. Tagging pass

```mermaid
sequenceDiagram
    participant W as Worker.Run
    participant DB as PostgreSQL
    participant G as gateway
    W->>W: wait for a signal (channel holds one)
    alt TAGGING_ENABLED false
        W->>W: log paused, tag nothing
    end
    W->>DB: dedicated connection; pg_advisory_lock(tagging key)
    W->>DB: untagged review ids, in id order (snapshot)
    loop each batch of 20
        W->>DB: texts of the batch
        loop round 0 to 2 (first call, then at most 2 retries)
            W->>G: one request for the pending ids; one request per id seen twice
            alt any gateway error (budget stop, 402, timeout, 5xx after retries, missing recording)
                W->>W: log; end the pass (HLD section 6: tagging stops, the next pass retries)
            end
            W->>W: ParseAnswer: valid by id, invalid and missing, seen twice
            W->>DB: insert valid results ON CONFLICT (review_id) DO NOTHING, with the prompt version
        end
        W->>W: ids still unresolved wait for the next pass
    end
    W->>DB: unlock; close the connection
    W->>DB: untagged ids newer than the snapshot?
    alt some
        W->>W: one more pass over those only
    end
```

Rules (GenAI 4.1, HLD section 3):

- The user message is one `<review id="N">text</review>` line per review in id order; the text has every `<review` and `</review` removed (ignoring case, until none is left) and is then cut at 2,000 characters.
- A line is `id|themes|sentiment|reasons`, four fields split on `|`, each trimmed. A line whose first field is not an integer is not a result and is ignored. Codes are compared in lower case. Themes are `-` or comma-separated codes from the list, none repeated; sentiment is `pos`, `neu` or `neg`; reasons are `-` or comma-separated from the three, none repeated.
- An id outside the batch is discarded (AC-US-01-004-2). An id on two or more lines is discarded and retried alone. An id whose one line is invalid counts as missing (AC-US-01-004-3). Position never matters (AC-US-01-004-1).
- When the answer stopped at max_tokens (`finish_reason` `length`), the text after the last newline is dropped: a cut line can look valid with a reason missing.
- A pass ends on any gateway error or store error; a context cancel (shutdown) ends it with nothing half stored. A panic in a pass is recovered and logged; the worker keeps running.
- One log line per batch (ids, calls, tagged, unresolved) and one per pass (snapshot size, tagged, unresolved, stop reason); review text is never logged.

## 7. Data access

| Query | Shape | Index |
| --- | --- | --- |
| `GetImportByRequestID` | one row by `request_id` | `uq_imports_request_id` |
| `ListImportRejections` | rejections of one import by row number | `import_rejections_pkey` |
| `CreateImport` | insert, `ON CONFLICT (request_id) DO NOTHING RETURNING` | `uq_imports_request_id` |
| `InsertReviews` | `INSERT ... SELECT FROM unnest(...) WITH ORDINALITY ORDER BY ord ON CONFLICT (natural key) DO NOTHING RETURNING id` | `uq_reviews_natural_key` |
| `InsertImportRejections`, `SetImportCounts` | arrays; update by id | primary keys |
| `ListOutlets` (changed) | adds `review_count` and `untagged_count` per outlet as subqueries | `idx_reviews_outlet_id_review_date` (leading column), `review_tags_pkey` |
| `UntaggedReviewIDs` | `reviews` with no `review_tags` row, `id > $after`, by id | `reviews_pkey`, `review_tags_pkey` (anti-join) |
| `ReviewTexts` | id and text where `id = ANY($1)` | `reviews_pkey` |
| `InsertReviewTag` | one row, `ON CONFLICT (review_id) DO NOTHING` | `review_tags_pkey` |

Transactions: the import is one transaction (import row, reviews, rejections, counts), so a repeat waits on the request id and returns the committed result (data model section 5). A batch's results are stored in one transaction after each call. The model call is never inside a transaction.

Concurrency: one worker goroutine per server; the coalescing channel turns signals during a pass into one more pass. The advisory lock (`pg_advisory_lock(7300301)`, session level, on a connection acquired from the pool and held for the pass, then released and the connection closed) keeps the phase 7 seed and eval commands from tagging at the same time. A second server process on the same database would wait on the lock, and `ON CONFLICT (review_id) DO NOTHING` keeps one result per review even then (tenet 4).

## 8. Errors

| Error | Created | Mapped |
| --- | --- | --- |
| `imports.ErrRoleNotAllowed` | service, before the body is read | 403 `role_not_allowed` |
| `*connector.FileError` | CSV connector | 400 `csv_invalid` with details |
| `connector.ErrFileTooLarge` | the capped reader under the CSV reader | 413 `file_too_large` |
| `connector.ErrNotImplemented` | `Google.Fetch` | never reached by a route |
| missing key, bad UUID, no `file` part | handler | 400 `malformed_request` |
| not `multipart/form-data` | handler | 415 `unsupported_media_type` |
| gateway errors (phase 2 LLD section 6) | gateway | logged by the worker; the pass ends |

## 9. Configuration

| Variable | Default | Missing or invalid |
| --- | --- | --- |
| `TAGGING_ENABLED` | `true` | must parse as a boolean, else the server refuses to start; `false` keeps the worker paused (HLD section 12) |

Added to `.env.example`. The theme list is code, not environment (the decisions table above).

## 10. Tests

Unit (`make check`):

- `connector`: `TestCSV_ReadsSixColumnsUnchanged` (AC-US-01-002-2), `TestCSV_RejectsRowsWithReasons` (table, one case per rule, AC-US-01-002-3), `TestCSV_FileErrors` (missing column named, repeated column, empty, bare quote), `TestCSV_RowNumbersCountRecordsNotLines`, `TestCSV_TooLarge`, `TestGoogle_FetchesNothing` (AC-US-01-002-5).
- `imports`: `TestImport_ManagerRefused`, `TestImport_RepeatReturnsFirstResultUnread`, `TestImport_UnknownOutletRejected`, `TestImport_SignalsTaggingOnlyWhenSomethingImported` (AC-US-01-002-6), `TestImport_RunsThroughAnyConnector` (AC-US-01-002-4).
- `httpapi`: `TestCreateImport_*`: 201 body shape, 400 key, 400 csv_invalid details, 413, 415, 403.
- `tagging`: `TestThemes_FreshListIsTheFive` (AC-US-01-003-3), `TestPrompt_CurrentNamesEveryTheme`, `TestMessage_StripsReviewTagsAndCuts`, `TestParseAnswer_*` (out-of-batch id, duplicate id, bad theme, missing sentiment, cut-off tail, preamble ignored, reasons ordered, AC-US-01-004-1 to -3, AC-US-01-003-2, -4, -5), `TestPass_45ReviewsMakeThreeRequests` (AC-US-01-003-1), `TestPass_RetriesOnlyMissingIDs` (17 stored, retry holds exactly the 3, AC-US-01-004-4), `TestPass_RetryThatSucceeds`, `TestPass_GivesUpAfterTwoRetries` (AC-US-01-004-5), `TestPass_DuplicateIDRetriedAlone`, `TestPass_GatewayErrorEndsPass`, `TestPass_RunsNewerReviewsAfterSnapshot`, `TestWorker_PausedTagsNothing`, `TestWorker_SignalsCoalesce`.

Integration (`make test-integration`, real PostgreSQL):

- `store`: `TestMigration00002_UpDownUp`, `TestImport_StoresRowsAndCountsDuplicates` (a second import of the same file stores nothing), `TestImport_ConcurrentSameKeyImportsOnce`, `TestOutlets_CountReviewsAndUntagged`, `TestTagging_LockExcludesASecondHolder`, `TestTagging_InsertKeepsTheFirstResult` (AC-US-01-003-6).
- `tagging`: `TestPassThroughReplayGateway`: the real gateway in replay mode answers a synthetic recording written by the test; tags are stored with prompt version 1 (AC-US-02-002-2, tenet 1); a tagged review is not sent again (AC-US-01-003-6).

Web (`make check`): `ImportPage.test.tsx`: ready, no outlets, uploading (button disabled, one key per file), success, partial with rejections, csv_invalid, 413; nav shows Import to the brand admin only.

## 11. Work items

| # | Item | Files | Tests | Est. lines |
| --- | --- | --- | --- | --- |
| 1 | Migration 2, import and tagging queries, store, outlet counts | migration, queries, `internal/store/*`, `internal/outlets/service.go` | store integration tests | 380 |
| 2 | Connector interface, CSV and Google | `internal/connector/*` | connector unit tests | 330 |
| 3 | Import service and `POST /imports` | `internal/imports/*`, `internal/httpapi/*`, `cmd/api/main.go` | imports, handler tests | 380 |
| 4 | Tagging prompt v1, themes, message, parser | `prompts/*`, `internal/tagging/{themes,message,parse}.go` | tagging unit tests | 380 |
| 5 | Worker, `TAGGING_ENABLED`, start-up wiring | `internal/tagging/worker.go`, `internal/store/tagging.go`, config, main | pass and worker tests, replay integration | 390 |
| 6 | S-07 Import screen and nav | `web/src/**` | `ImportPage.test.tsx` | 330 |

Order check: item 3 uses the store (1) and connector (2); item 5 uses the parser (4) and the store's tagging queries (1); item 3's signal is an interface, wired to the worker in item 5. Every item leaves `make check` green.

## 12. Assumptions and open points

- No real recording exists yet. Replay tests use fixture answers and synthetic recordings; live tagging quality (urgent recall, Hinglish, cost per batch, tokens per line) is not verified until the first paid call, which waits for the product owner's approval and the phase 2 section 10 checklist. Owner: product owner.
- Outlet names are matched with Go's `strings.ToLower` after trimming, while the database's unique index uses PostgreSQL `lower()`; they agree for the names in scope (Latin and Devanagari letters). Owner: developer.
- AC-US-01-002-1's wording ("each valid row becomes one stored review") still needs "rows already imported are counted, not stored" through `backlog` (HLD section 17). Owner: product owner.
