# API: OutletOwl API v1.1.0

Generated from `api/openapi.yaml` by openapi-spec (scripts/api_doc.py). Edit the spec, not this file.

**Style:** REST over HTTP with JSON bodies (one multipart upload for CSV import); one service, one same-origin browser client, no GraphQL (ADR-0005). · **Base path:** `/api/v1` · **Versioning:** Major version in the path (/api/v1); no date header. The UI and the server ship in one binary and the UI's types are generated from this spec, so a v2 is needed only if a second client appears (product owner, 2026-10-06).

**Authentication.** A signed JWT in an HttpOnly, SameSite=Strict cookie set by POST /auth/login, 8 hour expiry (ADR-0007). Every operation needs it except login and health. The server re-reads the user's role and outlet on every request; a record outside the caller's outlet is 404, and an action the caller's role may not take is 403. Public operations: POST /auth/login, POST /auth/logout, GET /health.

**Errors.** Every 4xx and 5xx returns the error envelope; clients switch on `code`.

| Code | HTTP status | Meaning |
| --- | --- | --- |
| `csv_invalid` | 400 | The upload is not a readable CSV or lacks a required column; nothing was imported. details names the column. |
| `malformed_request` | 400 | The body or a parameter could not be parsed, or a filter is not in the spec. |
| `invalid_credentials` | 401 | Email and password do not match an active account. |
| `unauthorized` | 401 | No session cookie, or an invalid or expired one; sign in again. |
| `role_not_allowed` | 403 | The caller can see the record but their role may not take this action (a manager adding an outlet, the brand admin drafting or replying). |
| `not_found` | 404 | No such record, or it belongs to an outlet the caller may not see. |
| `already_replied` | 409 | The reply is approved and its text can no longer change. |
| `draft_in_progress` | 409 | The model is still drafting this reply; wait and reload. |
| `outlet_name_taken` | 409 | An outlet with this name, ignoring capitals, already exists; details names it. |
| `reply_changed` | 409 | The reply changed since the client read it; reload it and apply the change again. |
| `file_too_large` | 413 | The CSV file is over 5 MB; split it into smaller files. |
| `unsupported_media_type` | 415 | The body is not application/json, or the import is not multipart/form-data. |
| `validation_failed` | 422 | A field breaks a rule (blank, too long, unknown theme); details names the field. |
| `internal` | 500 | Unexpected failure; quote request_id. |
| `mail_unavailable` | 502 | MailHog did not accept the digest; start MailHog and generate again with a new key. |
| `budget_exhausted` | 503 | The model budget is used up (over USD 8, or the provider's credit limit); drafting is unavailable, write the reply by hand. |
| `database_unavailable` | 503 | The server cannot reach PostgreSQL. |
| `model_unavailable` | 503 | The model did not answer within 45 seconds or returned an error; write the reply by hand or try later. |

## Conventions

1. Paths are plural kebab-case nouns and a non-CRUD action is a sub-resource (`POST /invoices/{id}/send`). Why: a client can guess the URL of a resource it has not seen, and verbs in paths multiply without limit.
2. JSON fields are snake_case, ids are strings, timestamps are RFC 3339 UTC with `Z`. Why: one casing and one clock remove a whole class of client parsing bugs.
3. Money is an integer `<name>_minor` plus an ISO 4217 `currency`. Why: floats cannot hold 0.10 exactly, and an amount without a currency is ambiguous.
4. Every 4xx and 5xx returns the one error envelope with a stable `code` and a `request_id`. Why: clients switch on `code`, not on English, and support finds the log line from `request_id`.
5. A record the caller may not see is 404, never 403. Why: a 403 confirms the record exists, which turns a guessed id into an oracle.
6. Every list is cursor-paginated with a capped `limit`. Why: offsets skip or repeat rows while data changes, and an uncapped page is a denial of service.
7. Every POST that creates or charges requires `Idempotency-Key`; the same key and body replays the first response for 24 hours. Why: mobile networks retry, and a retry must not create a second record or a second charge.
8. The path carries the major version and `X-API-Version` carries dated changes inside it. Why: installed clients cannot be forced to upgrade, so breaking changes need a new major and everything else a date.
9. Removal is `deprecated: true` plus a `Sunset` header and at least 90 days. Why: a removed field breaks a client nobody told, and oasdiff can only warn about what is still in the spec.
10. Every response carries the rate-limit headers and a 429 carries `Retry-After`. Why: a client that can see its budget backs off before it is throttled.
11. Ids are integers, not strings. Why: The data model keeps review ids short integers so 20 tagging result lines fit under max_tokens 1000 (HLD section 8); every id is far below 2^53, so JavaScript reads it exactly.
12. Sign-in is a cookie, not a bearer header. Why: ADR-0007 chose an HttpOnly cookie so page scripts can never read the token; same origin and SameSite=Strict keep cross-site requests from sending it.
13. Small bounded lists (outlets, themes) and computed reports (trends, heatmap, movers) return the whole result with no cursor; only the review list is cursor-paginated. Why: Outlets and themes are tens of rows, and a report is one computed answer; paging them would only add round trips.
14. Idempotency-Key is required on POST /imports and POST /digests and stored with the record for its lifetime; a repeat returns the first result whatever the body. Why: The data model stores the key as request_id under a unique constraint (tenet 8) and keeps no body hash; the record lives as long as the data, so there is no 24 hour expiry.
15. POST /outlets, POST /reviews/{review_id}/draft and POST /reviews/{review_id}/replied take no Idempotency-Key. Why: Each has a natural key the database enforces: the outlet name ignoring case, and one reply per review. A repeat returns the existing record (draft, replied) or 409 outlet_name_taken.
16. Reply writes carry based_on_updated_at, the reply's updated_at the client last read (null when it saw no reply); a mismatch is 409 reply_changed. Why: An outlet may have several managers and a manager may have two tabs open; a manager must only save over, or approve, text they have seen (REQ-024, product owner 2026-10-06).
17. No rate limiting and no rate-limit headers. Why: A local product with about 6 signed-in staff; neither the PRD nor the HLD asks for limits, and the model spend has its own USD 8 stop.
18. CSV uploads are capped at 5 MB; review and reply text at 5,000 characters; outlet name, source, reviewer name and file name at 200. Why: Decided by the product owner on 2026-10-06; 5 MB is about 15,000 reviews, and the limits close the data model's open length concern.

## auth

Sign in, sign out and who the caller is (ADR-0007).

Serves US-00-001.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| POST | `/auth/login` | Sign in with email and password (`login`) | none | 200 Signed in; the cookie is set | 400 malformed_request, 400 csv_invalid, 401 invalid_credentials, 422 validation_failed, 500 internal |
| POST | `/auth/logout` | Sign out (`logout`) | none | 204 Signed out; the cookie is cleared | 500 internal |
| GET | `/me` | The signed-in user, their outlet and the brand (`getMe`) | cookieAuth | 200 The caller | 401 unauthorized, 500 internal |

Idempotency:

- `POST /auth/login`: not idempotent; a retry repeats the action.
- `POST /auth/logout`: not idempotent; a retry repeats the action.

## outlets

The brand's outlets; the CSV outlet column is matched to their names.

Serves US-01-001, US-00-001.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/outlets` | List outlets with their managers and review counts (`listOutlets`) | cookieAuth | 200 The outlets the caller may see | 401 unauthorized, 500 internal |
| POST | `/outlets` | Add an outlet (brand admin) (`createOutlet`) | cookieAuth | 201 Created | 400 malformed_request, 400 csv_invalid, 401 unauthorized, 403 role_not_allowed, 409 outlet_name_taken, 422 validation_failed, 500 internal |

Idempotency:

- `POST /outlets`: not idempotent, no Idempotency-Key (finding).

## imports

CSV upload through the connector interface; tagging starts after each import.

Serves US-01-002.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| POST | `/imports` | Import reviews from a CSV file (brand admin) (`createImport`) | cookieAuth | 201 Imported; a repeat with the same Idempotency-Key returns this result again | 400 malformed_request, 400 csv_invalid, 401 unauthorized, 403 role_not_allowed, 413 file_too_large, 415 unsupported_media_type, 500 internal |

Idempotency:

- `POST /imports`: Idempotency-Key required. Client UUID, made once per user action. A repeat with the same key returns the first result and has no second side effect.

## reviews

The searchable review list, one review with its reply, drafting, editing and marking replied.

Serves US-01-003, US-01-006, US-01-008, US-00-001, US-01-007, US-00-002, US-00-003.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/themes` | The configured theme list (`listThemes`) | cookieAuth | 200 The themes in display order | 401 unauthorized, 500 internal |
| GET | `/reviews` | Search and filter reviews, newest first (`listReviews`) | cookieAuth | 200 One page | 400 malformed_request, 400 csv_invalid, 401 unauthorized, 404 not_found, 422 validation_failed, 500 internal |
| GET | `/reviews/{review_id}` | One review with its tags, reply and the outlet's managers (`getReview`) | cookieAuth | 200 The review | 401 unauthorized, 404 not_found, 500 internal |
| POST | `/reviews/{review_id}/draft` | Get the stored draft, or draft one with the model (outlet manager) (`createReplyDraft`) | cookieAuth | 200 The reply, stored or just drafted; 202 Another request is drafting this review; call again after Retry-After seconds | 401 unauthorized, 403 role_not_allowed, 404 not_found, 500 internal, 503 budget_exhausted, 503 model_unavailable |
| PUT | `/reviews/{review_id}/reply` | Save the reply text (outlet manager) (`saveReply`) | cookieAuth | 200 The saved reply, with its new updated_at | 400 malformed_request, 400 csv_invalid, 401 unauthorized, 403 role_not_allowed, 404 not_found, 409 reply_changed, 409 draft_in_progress, 409 already_replied, 415 unsupported_media_type, 422 validation_failed, 500 internal |
| POST | `/reviews/{review_id}/replied` | Approve the reply text and mark the review replied (outlet manager) (`markReplied`) | cookieAuth | 200 The replied reply | 400 malformed_request, 400 csv_invalid, 401 unauthorized, 403 role_not_allowed, 404 not_found, 409 reply_changed, 409 draft_in_progress, 415 unsupported_media_type, 422 validation_failed, 500 internal |

Idempotency:

- `POST /reviews/{review_id}/draft`: not idempotent; a retry repeats the action.
- `PUT /reviews/{review_id}/reply`: idempotent by definition; a retry has the same effect.
- `POST /reviews/{review_id}/replied`: not idempotent; a retry repeats the action.

## dashboard

Computed reports for the overview and themes screens; weeks run Monday to Sunday in the brand timezone.

Serves US-01-005, US-00-001, US-01-006, US-01-007.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/dashboard/trends` | Weekly rating, sentiment and reply counts per outlet, last 12 complete weeks (`getTrends`) | cookieAuth | 200 The trends | 401 unauthorized, 500 internal |
| GET | `/dashboard/heatmap` | Negative reviews by outlet and theme, last 4 complete weeks (`getHeatmap`) | cookieAuth | 200 The heatmap | 401 unauthorized, 500 internal |
| GET | `/dashboard/movers` | Biggest movers, latest complete week against the week before (`getMovers`) | cookieAuth | 200 The movers | 401 unauthorized, 500 internal |

## tagging

Untagged count, tagging worker state and the model budget state.

Serves US-01-003, US-02-001, US-01-005.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/tagging/status` | Untagged count, worker state and budget state (`getTaggingStatus`) | cookieAuth | 200 The status | 401 unauthorized, 500 internal |

## digests

Generate the weekly digest and send it to the brand admin through MailHog.

Serves US-01-009.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| POST | `/digests` | Generate the weekly digest and email it to the brand admin (`createDigest`) | cookieAuth | 201 Sent; 202 A repeat of a request whose digest is still marked sending; nothing is sent again | 400 malformed_request, 400 csv_invalid, 401 unauthorized, 403 role_not_allowed, 500 internal, 502 mail_unavailable |

Idempotency:

- `POST /digests`: Idempotency-Key required. Client UUID, made once per user action. A repeat with the same key returns the first result and has no second side effect.

## system

Health check for the local operator.

Serves unnumbered.

| Method | Path | Does | Auth | Success | Errors |
| --- | --- | --- | --- | --- | --- |
| GET | `/health` | Whether the server can reach PostgreSQL (`getHealth`) | none | 200 Healthy | 503 database_unavailable |

## Findings

Raised by the generator; each is a change to the spec or a decision to record.

- POST /outlets: creates (201) with no Idempotency-Key; a retried request makes a second record
