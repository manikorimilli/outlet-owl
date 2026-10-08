# Low Level Design: weekly digest, build phase 6

- Task: none. HLD: [review-intelligence-hld.md](review-intelligence-hld.md) flow C, sections 3, 6 (MailHog), 7 and 12 (phase 6). Author: unattributed, 2026-10-08, status Draft. Version: v1
- Companions: [data-model.md](data-model.md) (`digests`, the claim before send), `api/openapi.yaml` (createDigest, Digest), screen `app/S-08`, [phase 4 LLD](phase-4-dashboard-and-search-lld.md) (the movers the digest reuses)

Serves: US-01-009, REQ-025 to REQ-029, Q-004 to Q-006, tenet 8. Tests prove AC-US-01-009-1 to -5; AC-US-01-009-6 (the planted spike) needs the phase 7 seed and is checked in its smoke run.

## Design

- `db/migrations/00004_create_digests.sql`: `digest_status`, `digests`, `uq_digests_request_id`, copied from `schema.sql`.
- `internal/digest`: `Compose` (subject and plain-text body), `Service.Generate`, `SMTP` (the standard library's `net/smtp`, no new dependency).
- `POST /api/v1/digests` (brand admin, `Idempotency-Key` required): a known key returns its digest (201 sent, 202 sending, 502 failed) and sends nothing. Otherwise the movers come from `dashboard.Service.Movers` (the same ranking as the screen) and the urgent reviews of the week from the review list; the digest is inserted as `sending` (`ON CONFLICT (request_id) DO NOTHING`), sent, then marked `sent`, or `failed` with the SMTP error (502 `mail_unavailable`). The send and the status write run on a context the browser cannot cancel.
- Body: an untagged-count line first when any review in the two weeks is untagged, then the top mover with both counts, the ranked movers (first 10), and each urgent review with its reasons, outlet, date and text. Plain text only, so review text never becomes markup; the HLD's escaped HTML part is left out to keep it simple. The subject is kept on one line and Q-encoded.
- Configuration: `SMTP_ADDR` (default `localhost:1025`) and `SMTP_FROM` (default `digest@outletowl.local`), in `.env.example`.
- Web: S-08 at `/digest` for the brand admin, one button, a new key per press, the button off while a request runs.

## Tests

`digest`: compose (first line, rank order, urgent reasons, untagged line first, one-line subject), generate (one email to the admin, a repeat sends nothing, MailHog down recorded as failed, a manager refused), SMTP against a loopback fake server (headers, dot stuffing, UTF-8) and with no server. Store integration: migration up, down, up; one claim per key; sent stays sent; a non-Monday week refused. Web: `DigestPage.test.tsx`.
