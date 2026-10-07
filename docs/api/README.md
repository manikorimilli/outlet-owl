# OutletOwl API

The REST contract between the Go server and the React UI (ADR-0005, tenet 7).

Version: 1.1.1 (v3; base: 1.1.0, commit 6cb6409)

## Changes from 1.1.0

Why this revision: the critic review of the phase 1 low-level designs (2026-10-07) found that login and add-outlet return 415 without the spec saying so, and that SameSite=Strict alone does not stop a page on another localhost port from sending the session cookie. The product owner approved both fixes in session on 2026-10-07.

| Section | Change | Driven by | Impact |
| --- | --- | --- | --- |
| `POST /auth/login`, `POST /outlets` | 415 `unsupported_media_type` listed (the server already planned it for a body that is not `application/json`) | phase 1 critic; tenet 7 | documents existing behaviour; the UI always sends JSON |
| `x-conventions`, `Error` codes, every POST and PUT | New 403 `cross_site_request`: a write marked cross-origin by `Sec-Fetch-Site` (or, when it is missing, by an `Origin` that differs from `Host`) is refused; a request with neither header is allowed | phase 1 critic; product owner 2026-10-07 | new error code on writes; same-origin UI calls, including through the Vite proxy, are unaffected |
| `components.responses` | New `CrossSiteRequest` (login, logout); `Forbidden` also lists `cross_site_request` | the same | responses only |
| `POST /outlets` description | 409 `outlet_name_taken` carries the existing name in `details[0].reason` (field `name`) | phase 1 critic NIT | states what the UI reads; no shape change |

## Changes from 1.0.0 (1.1.0)

Why this revision: data model v2 stores why a review is urgent (approved by the product owner on 2026-10-06), and screens S-02, S-04, S-05 and S-08 show the reason beside each urgent review.

| Section | Change | Driven by | Impact |
| --- | --- | --- | --- |
| `components.schemas.Tags` | New required field `urgent_reasons`: an array of `food_safety`, `harassment`, `legal_threat`, no repeats, empty when not urgent; `is_urgent` is true exactly when it is not empty | data model v2; REQ-010; screens S-02, S-04, S-05, S-08 | responses only, additive; the UI's generated types gain the field; no request changes |
| Examples | Every `tags` example carries `urgent_reasons` | the same | API.md regenerated (its endpoint tables show no schema fields, so only the version line changes) |

Review (critic, 2026-10-06, on this change only): BLOCKER 0, MAJOR 0, MINOR 2, NIT 2. Fixed: the order of `urgent_reasons` is now fixed (food_safety, harassment, legal_threat) and the screens show each reason; the one case where `is_urgent` can be true with no reason (old rows under the data model's fallback migration 00005, if ever used) is stated on `is_urgent`; S-05 is named. Not applied: expressing the `is_urgent` rule as a JSON Schema `if`/`then`/`else`, because it complicates the generated UI types and the database CHECK already enforces it.

- **Spec:** [`api/openapi.yaml`](../../api/openapi.yaml), OpenAPI 3.1. It is the source of truth: a change to a request or response shape changes this file in the same merge request, and the UI's TypeScript types are generated from it, never edited by hand.
- **Readable design:** [`API.md`](API.md), generated from the spec. Do not edit it; change the spec and regenerate.
- **Base path:** `/api/v1`, served by the Go binary on the same origin as the UI.
- **Built from:** the PRD and backlog (`docs/product/`), ADR-0001 to ADR-0007, the HLD (`docs/design/review-intelligence-hld.md`, section 5), the data model (`docs/design/data-model.md`) and the screens (`docs/design/screens/app/`).

## Docs route

unconfirmed: no docs route. There is no server code yet. The house style serves the spec at `/docs` with Redoc or Scalar; if one is added, its files must ship inside the binary, because the runtime loads nothing from a CDN (HLD section 3). Whether to add it is a decision for `low-level-design`.

## Lint

```bash
npx @redocly/cli@2.54.3 lint api/openapi.yaml
```

The spec is valid with 4 warnings, each deliberate: no `license` field (a private project), a localhost server URL (the product runs locally), and no 4xx response on `POST /auth/logout` and `GET /health`, which never refuse a caller.

## Regenerate API.md

```bash
python3 <bearing plugin>/skills/openapi-spec/scripts/api_doc.py --spec api/openapi.yaml --style <bearing plugin>/skills/openapi-spec/references/api-style.md --out docs/api/API.md
```

It needs PyYAML (`uv run --with pyyaml` works where uv is installed).

## Style

The house rules are in the Bearing `openapi-spec` style reference and are copied, with their reasons, into the Conventions section of API.md. This API keeps them except where the spec's `x-conventions` says otherwise, each with its reason:

- ids are integers, not strings;
- sign-in is an HttpOnly cookie (ADR-0007), not a bearer header;
- only the review list is cursor-paginated; small lists and computed reports return whole;
- `Idempotency-Key` on `POST /imports` and `POST /digests` is kept for the record's lifetime, with no body hash;
- `POST /outlets`, the draft and mark-replied rely on natural keys instead of `Idempotency-Key`;
- reply writes carry `based_on_updated_at`, and a stale write is 409 `reply_changed`;
- writes from another origin are refused with 403 `cross_site_request` (`Sec-Fetch-Site`, with an `Origin` to `Host` fallback);
- no rate limiting;
- versioning is `/api/v1` only, with no `X-API-Version` date header.

Limits decided by the product owner on 2026-10-06: CSV uploads up to 5 MB; review and reply text up to 5,000 characters; outlet name, source, reviewer name and file name up to 200.

## Revision history

- 1.0.0 (2026-10-06, commit a7d1998): first version.
- 1.1.0 (2026-10-06, commit 6cb6409): `urgent_reasons` on `Tags`.
- 1.1.1 (2026-10-07): 415 on login and add-outlet; 403 `cross_site_request` on writes; the 409 outlet name detail stated.
