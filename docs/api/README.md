# OutletOwl API

The REST contract between the Go server and the React UI (ADR-0005, tenet 7).

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
- no rate limiting;
- versioning is `/api/v1` only, with no `X-API-Version` date header.

Limits decided by the product owner on 2026-10-06: CSV uploads up to 5 MB; review and reply text up to 5,000 characters; outlet name, source, reviewer name and file name up to 200.
