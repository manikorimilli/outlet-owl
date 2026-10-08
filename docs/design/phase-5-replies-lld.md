# Low Level Design: reply drafts and replies, build phase 5

- Task: none. HLD: [review-intelligence-hld.md](review-intelligence-hld.md) flow B, sections 3, 5, 7, 8 and 12 (phase 5). ADRs: [0007](../adr/0007-use-jwt-cookies-for-sign-in.md), [0008](../adr/0008-tag-and-draft-with-prompt-only-haiku-calls.md)
- Author: unattributed, 2026-10-08, status Draft. Version: v1
- Companions: [data-model.md](data-model.md) section 5 (claims, transitions, outlet scope on writes), [GenAI solution](../genai/review-classification-and-replies-solution.md) section 4.2 (the reply call), `api/openapi.yaml` (getReview, createReplyDraft, saveReply, markReplied), screen `app/S-05`

Serves: US-00-002, US-00-003, US-02-002 (reply prompt v1). Tests prove AC-US-00-002-1 to -3 and -7, AC-US-00-003-1 to -4. AC-US-00-002-4 and -5 (the draft's language) depend on the model and are checked by the phase 7 tone check, not by replay. AC-US-00-002-6 is the phase 4 fonts. AC-US-00-003-5 is checked by eye on S-05.

## 1. Scope

The `replies` table and its claim, the reply prompt v1, `GET /reviews/{id}` and the three reply writes, the reply status in the review list, filter and trends, and the S-05 review page. Only an active manager of the review's outlet drafts, edits or marks replied; the brand admin reads (Q-002).

## 2. Layout

| File | Owns |
| --- | --- |
| `db/migrations/00003_create_replies.sql` (new) | `reply_status`, `replies`, `idx_replies_replied_by`, copied from `schema.sql` |
| `db/queries/replies.sql` (new); `reviews.sql`, `dashboard.sql` | review detail, claim, take over, land, release, save, hand-written insert, mark replied; reply status joined into the list and trends |
| `internal/replies/service.go`, `message.go` (new) | the rules and transitions, the reply user message, the 45 second deadline |
| `internal/store/replies.go` (new) | the statements, each scoped to the caller's outlet |
| `prompts/reply/v1.md`, `current` (new) | the brand tone and language rule (GenAI 4.2); `max_tokens: 1000`, temperature left to the provider |
| `internal/httpapi/replies_handler.go` (new) | the four routes |
| `web/src/features/reply/*` (new) | S-05 |

## 3. Flows and rules

- **Draft** (`POST /draft`): a non-manager is 403 `role_not_allowed`; a review outside the manager's outlet is 404. An existing reply in `draft` or `replied` is returned (200, no model call). A `drafting` row younger than 60 seconds answers 202 with `Retry-After: 2`; an older one is taken over with `UPDATE ... WHERE updated_at < now() - 60 s RETURNING updated_at`. Otherwise the claim is `INSERT ... SELECT FROM reviews WHERE id AND outlet_id ON CONFLICT DO NOTHING RETURNING updated_at`; losing the race reads the row again. The winner calls the gateway inside one 45 second deadline. A gateway error, a blank or cut-off answer, or an answer over 5,000 characters releases the claim (`DELETE ... WHERE updated_at = claimed`) and answers 503: `budget_exhausted` for the USD 8 stop or a 402, else `model_unavailable`. A good answer lands as `UPDATE ... SET status 'draft', draft_text, reply_text, prompt_version WHERE updated_at = claimed`; if the claim was taken over meanwhile, nothing is written and the stored row is returned.
- **Save** (`PUT /reply`): with `based_on_updated_at` null and no reply, a hand-written reply is inserted as `draft`. Otherwise `UPDATE ... WHERE status = 'draft' AND updated_at = seen`. No match: 409 `already_replied` for a replied row, `draft_in_progress` for a drafting one, else `reply_changed`.
- **Mark replied** (`POST /replied`): `UPDATE ... SET status 'replied', reply_text, replied_by, replied_at WHERE status = 'draft' AND updated_at = seen`. No match on a replied row returns it unchanged (200, safe to repeat); on a drafting row 409 `draft_in_progress`; otherwise 409 `reply_changed`.
- `reply_text` is 1 to 5,000 characters after trimming, else 422 `validation_failed`.
- The user message names the outlet, the signer ("<manager first name>, outlet manager, <outlet>"), the reviewer's first name (the first word of the name, left out when it is one letter or not a word), the rating and the review text cut at 2,000 characters inside `<review>` tags.
- Review list: `reply_status` is the row's status or `none`; `filter[reply_status]=none` matches no row or a drafting row. Trends count `replied` per week.

## 4. Data access

All statements use `replies_pkey` (review id) and join `reviews` on its primary key for the outlet condition; `idx_replies_replied_by` is the foreign-key index. Each write is one statement, so no explicit transaction. The model call is outside every statement.

## 5. Tests

- `replies` unit, with a fake store and model: stored draft returned without a call (AC-US-00-002-2); draft made, stored with the prompt version (-1, -3); budget stop releases the claim and answers budget_exhausted (-7); cut-off answer released; a fresh drafting row answers 202; the admin refused (AC-US-00-003-3); message building and the first-name rule.
- store integration: claim once under two claims; take over a stale claim; land only with the claim's token; save with a stale token is a conflict; mark replied twice returns the same row (AC-US-00-003-2); another outlet's manager writes nothing (AC-US-00-003-3); list and trends show the status (AC-US-00-003-4).
- handler: status codes and Retry-After. Web: S-05 drafting, unavailable, edit, mark replied, admin read-only.

## 6. Work items

1. Migration, queries, store, list and trends status (about 350 lines).
2. Reply prompt, service, handlers, wiring (about 380 lines).
3. S-05 screen and links from S-04 and S-02 (about 350 lines).
