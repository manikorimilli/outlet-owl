# OutletOwl: finish the AI checks (handoff)
Updated: 2026-10-08 21:30 IST   Branch: bugfix/a11y-table-scroll   Base: main (+1 commit)
Gate: make check on 16e0282: 9 gates run, 0 skipped, passed (24 Go packages; 17 web test files, 91 tests)
Only on this machine: commit 16e0282 (branch has no upstream); 92 untracked files in testdata/recordings/tagging and testdata/recordings/evaluation (free-model recordings, no keys in them); local branch docs/threat-model (already merged)

## Next (do this first)
- Push and merge this branch: git push -u origin bugfix/a11y-table-scroll, then a PR to main.
- When the owner has set OPENROUTER_API_KEY and MODEL_ID in .env (MODEL_GATEWAY_MODE=record): make seed SEED_ARGS="-reviews=500". It replaces the dummy keyword tags now in the local database.
- Then sign in as manager2@example.in, open a Koramangala review and check the AI draft; then make tone-check and score the sheet.

## Criteria (backlog US-00 to US-02)
- Build phases 1 to 7, screens S-01 to S-09: done; merged to main, CI green on PRs #1 to #8.
- US-00-002 drafted reply: partly; code and replay tests pass (internal/replies/service_test.go), never seen with a live model answer (free model hit 429).
- US-02-004 evaluation, urgent recall at least 90%: not started; needs testdata/eval/reviews.jsonl, 100 reviews labelled by the product owner. Never fabricate the labels.
- US-02-005 tone check: not started; needs live drafts.
- US-02-006 seed: done (internal/seed/generate_test.go); -reviews=N added for trial runs.
- Accessibility: one High finding fixed on this branch (scrolling tables, WCAG 2.1.1); gate web/src/test/a11y.ts.

## Done
- 16e0282 fix(web): let keyboard users scroll wide tables; add an a11y gate
- Earlier on main: model choice MODEL_ID (ADR-0009), reasoning sent off, listen on 127.0.0.1, README and docs/runbooks/local-operations.md, threat model docs/security/threat-model-outletowl.md, UI polish.

## Blockers
- No API key: the owner will add OPENROUTER_API_KEY and choose MODEL_ID later. Every AI step waits on that.
- Evaluation labels: the owner writes the 100 labelled reviews.

## Open questions
- Which model to use for the real run (MODEL_ID empty means Claude Haiku 4.5, the designed model). Owner decides.

## Traps
- The local database holds DUMMY tags written by keyword for 500 reviews so the heatmap shows; they are not AI output. make seed replaces them.
- The free model apodex/apodex-1.1-mini:free tagged 1,090 of 1,500 reviews (100% sentiment, 95% urgent recall on that set), then hit OpenRouter's 429 limit (20 a minute, 50 a day under USD 10 of credit).
- The budget log total counts reserved prices for failed calls; it is above what OpenRouter billed.
- Threat model: T-01 (no sign-in rate limit) and T-17 (no security headers) are open; fix before any shared deployment.
- testdata/recordings/tagging no longer matches the 500-review seed; safe to delete.

## How to run
- make check (offline gate); make db and make test-integration for the database tests.
- Read first: README.md, docs/runbooks/local-operations.md, AGENTS.md.
