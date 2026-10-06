# Entity relationship diagram: Review Intelligence (OutletOwl)

PostgreSQL. 9 tables (8 in `public`, 1 in `budget`), 7 relationships. Written beside [data-model.md](data-model.md) from the same design; the DDL is [schema.sql](schema.sql).

## Diagram

```mermaid
erDiagram
    outlets ||--o{ users : "is managed by"
    outlets ||--o{ reviews : "is reviewed in"
    imports ||--o{ reviews : "brought in"
    imports ||--o{ import_rejections : "rejected"
    reviews ||--o| review_tags : "is tagged by"
    reviews ||--o| replies : "is answered by"
    users ||--o{ replies : "approved"
    outlets {
        bigint id PK
        text name UK "matched by the CSV outlet column, ignoring case"
        timestamptz created_at
        timestamptz updated_at
    }
    users {
        bigint id PK
        text email UK "sign-in name (personal data)"
        text name "personal data"
        user_role role "brand_admin or outlet_manager"
        bigint outlet_id FK "null for the brand admin"
        text password_hash "bcrypt"
        timestamptz removed_at "left the users file"
        timestamptz created_at
        timestamptz updated_at
    }
    imports {
        bigint id PK
        uuid request_id UK "client-made, tenet 8"
        text file_name
        integer imported_count
        integer duplicate_count
        integer rejected_count
        timestamptz created_at
    }
    import_rejections {
        bigint import_id PK, FK
        integer row_number PK
        text reason
    }
    reviews {
        bigint id PK "sent to the model as the result key"
        bigint outlet_id FK
        bigint import_id FK "null for seeded reviews"
        text source
        date review_date "calendar date, brand timezone"
        smallint rating "1 to 5"
        text review_text "personal data"
        text reviewer_name "personal data"
        timestamptz created_at
    }
    review_tags {
        bigint review_id PK, FK "no row means untagged"
        text_array themes "configured theme codes"
        sentiment sentiment "positive, neutral, negative"
        boolean is_urgent
        integer prompt_version
        timestamptz created_at
    }
    replies {
        bigint review_id PK, FK "one per review, claimed first"
        reply_status status "drafting, draft, replied"
        text draft_text "personal data"
        integer prompt_version
        text reply_text "personal data"
        bigint replied_by FK
        timestamptz replied_at
        timestamptz created_at
        timestamptz updated_at
    }
    digests {
        bigint id PK
        uuid request_id UK "client-made, tenet 8"
        digest_status status "sending, sent, failed"
        date week_start "a Monday"
        text recipient_email "personal data"
        integer untagged_count
        text subject
        text body "personal data"
        text failure_reason
        timestamptz sent_at
        timestamptz created_at
        timestamptz updated_at
    }
    model_calls {
        bigint id PK "budget schema"
        model_call_purpose purpose
        text model
        integer prompt_version
        integer input_tokens
        integer output_tokens
        numeric reserved_cost_usd
        numeric settled_cost_usd
        model_call_outcome outcome
        timestamptz created_at
        timestamptz updated_at
    }
```

`themes` is `text[]` in the schema; mermaid cannot print brackets in a type, so the diagram writes `text_array`.

## Relationships

| From | | To | Meaning |
| --- | --- | --- | --- |
| outlets | one-to-many | users | An outlet can have several managers and each manager has one outlet; the brand admin has none. An outlet with managers cannot be deleted (RESTRICT). |
| outlets | one-to-many | reviews | Every review belongs to exactly one outlet; an outlet with reviews cannot be deleted (RESTRICT), so no review loses the outlet it is compared under. |
| imports | one-to-many | reviews | An import brings in many reviews; seeded reviews have no import. An import with reviews cannot be deleted (RESTRICT). |
| imports | one-to-many | import_rejections | An import lists each row it rejected; the list belongs to the import and goes with it (CASCADE). |
| reviews | one-to-zero-or-one | review_tags | A review has at most one tag result, and none while untagged; the result follows its review (CASCADE). |
| reviews | one-to-zero-or-one | replies | A review has at most one reply, created when a manager first opens it; the reply follows its review (CASCADE). |
| users | one-to-many | replies | A manager approves many replies; a manager who approved a reply cannot be deleted (RESTRICT), only marked removed, so the reply keeps its approver. |

`model_calls` has no relationship on purpose: it lives in the `budget` schema, which the seed reset never truncates and no Down migration drops, so it must not depend on any domain row (HLD section 4, tenet 2).
