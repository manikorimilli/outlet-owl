# Low Level Design: seed, evaluation and tone check, build phase 7

- Task: none. HLD: [review-intelligence-hld.md](review-intelligence-hld.md) flow D, sections 3 (commands), 7, 8 and 12 (phase 7). Author: unattributed, 2026-10-08, status Draft. Version: v1
- Companions: [GenAI solution](../genai/review-classification-and-replies-solution.md) sections 4.2 and 7, [data-model.md](data-model.md) section 5 (seed reset)

Serves: US-02-004, US-02-005, US-02-006, REQ-036 to REQ-043, Q-017 to Q-019. Also closes the HLD's "the Go binary serves the UI" (ADR-0002), which no earlier phase built.

## Seed (`make seed`, `cmd/seed`, `internal/seed`)

- Takes the tagging advisory lock, truncates the domain tables (never the budget schema), restarts review ids at 1, creates the 5 outlets, writes the users file (one brand admin, one manager per outlet, all with the local demo password `outletowl-demo` or `SEED_PASSWORD`) and loads it, then inserts 1,500 reviews.
- Reviews are generated deterministically: 26 weeks ending with the latest complete week in the brand timezone, 11 or 12 per outlet and week, mostly positive, some neutral, some negative, about 3% urgent, in English, Devanagari Hindi and Hinglish; Koramangala gets 15 extra wait-time complaints in the latest week (REQ-043). Only the dates move with the run, so ids, batches and texts, and therefore the recordings, match on every seed. `-reviews=N` (default 1,500, at least 145) generates a smaller or larger set for a trial on a rate-limited model; the 15 spike reviews stay, so the spike still shows.
- Then it tags in its own process through the gateway in the operator's mode, under the lock it already holds. In replay with no recordings it fails after seeding and says how to record once (`MODEL_GATEWAY_MODE=record`, about USD 0.57) or to run with `SEED_ARGS=-tag=false`.

## Evaluation (`make eval`, `cmd/eval`, `internal/eval`)

- Reads `testdata/eval/reviews.jsonl`, one `{id, text, themes, sentiment, urgent_reasons}` per line, labelled by the product owner. A missing file stops with what the set must hold.
- Tags in batches of 20 with the tagging worker's own `tagging.Classify` (same prompt, parser and retries) under the purpose `evaluation`; writes no tag results.
- Reports per-theme precision and recall, urgent recall and precision with each missed urgent review, sentiment accuracy and negative precision and recall, the model and the prompt version. Exit 1 when urgent recall is below 90%.

## Tone check (`make tone-check`, `cmd/tonecheck`, `internal/tonecheck`)

- Picks 30 stored reviews spread over the list, at least 5 in Devanagari when present, drafts each through the gateway (purpose `tone_check`, reply prompt), and writes a Markdown sheet with the rubric from reply prompt v1 (8 items, scored 0 to 2). `-report <file>` totals a filled sheet per draft and per item; no verdict.

## Serving the UI

- `WEB_DIST` (default `web/dist`): when it holds `index.html`, the server serves the files on its own origin and answers `index.html` for any other non-API path, so client routes reload. Hashed assets are cached for a year.

## Tests

`seed`: 1,500 reviews in 26 weeks, unique natural keys, the spike at least 3 times the weekly average, deterministic across weeks. `eval`: a gate failure listing the missed review, a pass at full recall, label checks (synthetic fixture set; it proves the arithmetic, not model quality). `tonecheck`: the pick, the sheet, the totals. `httpapi`: files served, client routes fall back to `index.html`, unknown API routes still 404.
