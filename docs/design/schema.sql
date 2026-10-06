-- Review Intelligence (OutletOwl): database schema (PostgreSQL 16)
-- Written by data-model beside docs/design/data-model.md, which gives the
-- reason for every table, column, index and constraint below.
--
-- Applies in one transaction to an empty database: enum types first, then
-- tables in foreign-key order, each followed by its comments and indexes.
-- This file is the reference for the whole model. The goose migrations split
-- it by build phase (data-model.md section 8): the domain tables in the
-- public schema, and the budget schema in its own migration set with no Down
-- (HLD section 4). Every change after the first release is its own
-- migration (db-migration), never an edit to this file.

BEGIN;

CREATE SCHEMA budget;

CREATE TYPE user_role AS ENUM ('brand_admin', 'outlet_manager');
CREATE TYPE sentiment AS ENUM ('positive', 'neutral', 'negative');
CREATE TYPE reply_status AS ENUM ('drafting', 'draft', 'replied');
CREATE TYPE digest_status AS ENUM ('sending', 'sent', 'failed');
CREATE TYPE budget.model_call_purpose AS ENUM ('tagging', 'drafting', 'evaluation', 'tone_check', 'reconciliation');
CREATE TYPE budget.model_call_outcome AS ENUM ('reserved', 'settled', 'failed');

-- outlets: one location of the brand whose reviews are imported and compared.
-- Serves US-01-001, US-01-002, US-01-005, US-01-006, US-01-007, US-00-001
CREATE TABLE outlets (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT outlets_pkey PRIMARY KEY (id),
    CONSTRAINT chk_outlets_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT chk_outlets_name_trimmed CHECK (name = btrim(name))
);
COMMENT ON TABLE outlets IS 'One location of the brand. Serves US-01-001, US-01-002, US-01-005, US-01-006, US-01-007, US-00-001.';
COMMENT ON COLUMN outlets.id IS 'Surrogate key; reviews and managers reference it.';
COMMENT ON COLUMN outlets.name IS 'Outlet name as the admin typed it; the CSV outlet column must match it, ignoring case.';
COMMENT ON COLUMN outlets.created_at IS 'When the outlet was added.';
COMMENT ON COLUMN outlets.updated_at IS 'Last change to this row.';
-- Case-insensitive unique name: the CSV outlet column is matched by it, and a repeated add returns the existing outlet.
CREATE UNIQUE INDEX uq_outlets_name_lower ON outlets (lower(name));

-- users: an account loaded from the users file at server start (or written by the seed).
-- Serves US-00-001, US-00-003, US-01-001, US-01-009, US-02-006
CREATE TABLE users (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    email text NOT NULL,
    name text NOT NULL,
    role user_role NOT NULL,
    outlet_id bigint REFERENCES outlets (id) ON DELETE RESTRICT,
    password_hash text NOT NULL,
    removed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT chk_users_email_shape CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+$'),
    CONSTRAINT chk_users_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT chk_users_role_outlet CHECK (
        (role = 'brand_admin' AND outlet_id IS NULL)
        OR (role = 'outlet_manager' AND outlet_id IS NOT NULL)
    )
);
COMMENT ON TABLE users IS 'An account from the users file; one brand admin and outlet managers. Serves US-00-001, US-00-003, US-01-001, US-01-009, US-02-006.';
COMMENT ON COLUMN users.id IS 'Surrogate key; the only identity the JWT carries (ADR-0007).';
COMMENT ON COLUMN users.email IS 'Sign-in name and the digest recipient address. [personal data]';
COMMENT ON COLUMN users.name IS 'Name shown in the top bar and beside a reply the manager marked replied. [personal data]';
COMMENT ON COLUMN users.role IS 'brand_admin or outlet_manager, re-read on every request (ADR-0007).';
COMMENT ON COLUMN users.outlet_id IS 'The one outlet an outlet manager may see and act on; null for the brand admin.';
COMMENT ON COLUMN users.password_hash IS 'bcrypt hash from the users file; never the password.';
COMMENT ON COLUMN users.removed_at IS 'Set when the account left the users file; sign-in is refused, the row stays so replies keep their manager.';
COMMENT ON COLUMN users.created_at IS 'When the account was first loaded.';
COMMENT ON COLUMN users.updated_at IS 'Last change from the users file.';
-- Sign-in looks the account up by email, ignoring case; the users file upsert matches on it.
CREATE UNIQUE INDEX uq_users_email_lower ON users (lower(email));
-- One active brand admin: the digest goes to the brand admin (Q-006).
CREATE UNIQUE INDEX uq_users_one_brand_admin ON users (role) WHERE role = 'brand_admin' AND removed_at IS NULL;
-- Foreign-key index: the managers of an outlet (outlets screen), and no table scan on an outlet delete check.
CREATE INDEX idx_users_outlet_id ON users (outlet_id);

-- imports: one CSV upload and its result, stored under the client's request id.
-- Serves US-01-002
CREATE TABLE imports (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    request_id uuid NOT NULL,
    file_name text NOT NULL,
    imported_count integer NOT NULL DEFAULT 0,
    duplicate_count integer NOT NULL DEFAULT 0,
    rejected_count integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT imports_pkey PRIMARY KEY (id),
    CONSTRAINT chk_imports_counts_not_negative CHECK (imported_count >= 0 AND duplicate_count >= 0 AND rejected_count >= 0)
);
COMMENT ON TABLE imports IS 'One CSV upload and its result. Serves US-01-002.';
COMMENT ON COLUMN imports.id IS 'Surrogate key; reviews record the import that brought them in.';
COMMENT ON COLUMN imports.request_id IS 'Client-made id; a repeat with the same id returns this result and imports nothing (tenet 8).';
COMMENT ON COLUMN imports.file_name IS 'Name of the uploaded file, shown with the result.';
COMMENT ON COLUMN imports.imported_count IS 'Rows stored as new reviews; set before the import commits.';
COMMENT ON COLUMN imports.duplicate_count IS 'Rows skipped because the same review was already imported.';
COMMENT ON COLUMN imports.rejected_count IS 'Rows rejected; each is listed in import_rejections.';
COMMENT ON COLUMN imports.created_at IS 'When the import committed.';
-- A repeated upload (double click, network retry) finds the first result by its request id.
CREATE UNIQUE INDEX uq_imports_request_id ON imports (request_id);

-- import_rejections: each CSV row an import rejected, with its row number and reason.
-- Serves US-01-002
CREATE TABLE import_rejections (
    import_id bigint NOT NULL REFERENCES imports (id) ON DELETE CASCADE,
    row_number integer NOT NULL,
    reason text NOT NULL,
    CONSTRAINT import_rejections_pkey PRIMARY KEY (import_id, row_number),
    CONSTRAINT chk_import_rejections_row_positive CHECK (row_number >= 1),
    CONSTRAINT chk_import_rejections_reason_not_blank CHECK (btrim(reason) <> '')
);
COMMENT ON TABLE import_rejections IS 'A CSV row an import rejected. Serves US-01-002.';
COMMENT ON COLUMN import_rejections.import_id IS 'The import that rejected the row.';
COMMENT ON COLUMN import_rejections.row_number IS 'Row number in the file, counted as the admin sees it in a spreadsheet.';
COMMENT ON COLUMN import_rejections.reason IS 'Why the row was rejected, worded so the admin can fix it.';

-- reviews: one customer review of one outlet, from a CSV import or the seed.
-- Serves US-01-002, US-01-003, US-01-005, US-01-006, US-01-007, US-01-008, US-00-001, US-00-002, US-01-009, US-02-006
CREATE TABLE reviews (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    outlet_id bigint NOT NULL REFERENCES outlets (id) ON DELETE RESTRICT,
    import_id bigint REFERENCES imports (id) ON DELETE RESTRICT,
    source text NOT NULL,
    review_date date NOT NULL,
    rating smallint NOT NULL,
    review_text text NOT NULL,
    reviewer_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT reviews_pkey PRIMARY KEY (id),
    CONSTRAINT chk_reviews_rating_range CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT chk_reviews_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT chk_reviews_text_not_blank CHECK (btrim(review_text) <> ''),
    CONSTRAINT chk_reviews_reviewer_not_blank CHECK (btrim(reviewer_name) <> '')
);
COMMENT ON TABLE reviews IS 'One customer review of one outlet. Serves US-01-002, US-01-003, US-01-005 to US-01-009, US-00-001, US-00-002, US-02-006.';
COMMENT ON COLUMN reviews.id IS 'Integer id; the id the tagging batch sends and every result line must carry (tenet 4).';
COMMENT ON COLUMN reviews.outlet_id IS 'The outlet reviewed, matched from the CSV outlet column; the outlet scope filters on it.';
COMMENT ON COLUMN reviews.import_id IS 'The import that brought the review in; null for reviews the seed command wrote.';
COMMENT ON COLUMN reviews.source IS 'Where the review was written, as the CSV source column says.';
COMMENT ON COLUMN reviews.review_date IS 'Calendar date of the review; weeks are Monday to Sunday in the brand timezone.';
COMMENT ON COLUMN reviews.rating IS 'Star rating, a whole number from 1 to 5.';
COMMENT ON COLUMN reviews.review_text IS 'The review as written, kept unchanged. [personal data]';
COMMENT ON COLUMN reviews.reviewer_name IS 'Name of the person who wrote the review. [personal data]';
COMMENT ON COLUMN reviews.created_at IS 'When the review was stored.';
-- Re-importing a file skips rows already stored: same outlet, source, date, reviewer and text.
CREATE UNIQUE INDEX uq_reviews_natural_key ON reviews (outlet_id, source, review_date, reviewer_name, md5(review_text));
-- Trends, movers, the digest week and a manager's review list: one outlet over a date range, newest first.
CREATE INDEX idx_reviews_outlet_id_review_date ON reviews (outlet_id, review_date);
-- Foreign-key index: the reviews of one import.
CREATE INDEX idx_reviews_import_id ON reviews (import_id);

-- review_tags: the one stored tag result of a review; no row means untagged.
-- Serves US-01-003, US-01-004, US-01-005, US-01-006, US-01-007, US-01-008, US-01-009, US-02-002
CREATE TABLE review_tags (
    review_id bigint NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    themes text[] NOT NULL,
    sentiment sentiment NOT NULL,
    is_urgent boolean NOT NULL,
    prompt_version integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT review_tags_pkey PRIMARY KEY (review_id),
    CONSTRAINT chk_review_tags_themes_no_nulls CHECK (array_position(themes, NULL) IS NULL),
    CONSTRAINT chk_review_tags_prompt_version_positive CHECK (prompt_version >= 1)
);
COMMENT ON TABLE review_tags IS 'The stored tag result of a review; written once, never re-tagged. Serves US-01-003 to US-01-009, US-02-002.';
COMMENT ON COLUMN review_tags.review_id IS 'The review the result line named; one result per review.';
COMMENT ON COLUMN review_tags.themes IS 'Theme codes from the configured list; zero, one or several.';
COMMENT ON COLUMN review_tags.sentiment IS 'One label: positive, neutral or negative.';
COMMENT ON COLUMN review_tags.is_urgent IS 'True when the review concerns food safety, harassment or a legal threat.';
COMMENT ON COLUMN review_tags.prompt_version IS 'Number of the tagging prompt version that produced this result.';
COMMENT ON COLUMN review_tags.created_at IS 'When the result was stored.';

-- replies: the draft and the approved reply of a review, at most one per review.
-- Serves US-00-002, US-00-003, US-01-008, US-02-002
CREATE TABLE replies (
    review_id bigint NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    status reply_status NOT NULL DEFAULT 'drafting',
    draft_text text,
    prompt_version integer,
    reply_text text,
    replied_by bigint REFERENCES users (id) ON DELETE RESTRICT,
    replied_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT replies_pkey PRIMARY KEY (review_id),
    CONSTRAINT chk_replies_draft_has_version CHECK ((draft_text IS NULL) = (prompt_version IS NULL)),
    CONSTRAINT chk_replies_prompt_version_positive CHECK (prompt_version >= 1),
    CONSTRAINT chk_replies_drafting_is_empty CHECK (
        status <> 'drafting' OR (draft_text IS NULL AND reply_text IS NULL)
    ),
    CONSTRAINT chk_replies_replied_complete CHECK (
        (status = 'replied' AND reply_text IS NOT NULL AND replied_by IS NOT NULL AND replied_at IS NOT NULL)
        OR (status <> 'replied' AND replied_by IS NULL AND replied_at IS NULL)
    )
);
COMMENT ON TABLE replies IS 'The draft and the approved reply of a review. Serves US-00-002, US-00-003, US-01-008, US-02-002.';
COMMENT ON COLUMN replies.review_id IS 'The review replied to; the primary key lets only one request claim the draft.';
COMMENT ON COLUMN replies.status IS 'drafting while the model call runs, draft while editable, replied once a manager approved it.';
COMMENT ON COLUMN replies.draft_text IS 'The text the model drafted, kept as drafted; null while drafting or when written by hand. [personal data]';
COMMENT ON COLUMN replies.prompt_version IS 'Number of the reply prompt version that drafted draft_text; null without a model draft.';
COMMENT ON COLUMN replies.reply_text IS 'The text the manager edits and approves; starts as the draft. [personal data]';
COMMENT ON COLUMN replies.replied_by IS 'The outlet manager who marked it replied, which is the approval.';
COMMENT ON COLUMN replies.replied_at IS 'When the manager marked it replied.';
COMMENT ON COLUMN replies.created_at IS 'When the draft was claimed or the reply first written.';
COMMENT ON COLUMN replies.updated_at IS 'Last change; a drafting claim older than 60 seconds may be taken over.';
-- Foreign-key index: replies a manager approved, and no table scan on a user delete check.
CREATE INDEX idx_replies_replied_by ON replies (replied_by);

-- digests: one generated weekly digest, claimed before it is sent.
-- Serves US-01-009
CREATE TABLE digests (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    request_id uuid NOT NULL,
    status digest_status NOT NULL DEFAULT 'sending',
    week_start date NOT NULL,
    recipient_email text NOT NULL,
    untagged_count integer NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    failure_reason text,
    sent_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT digests_pkey PRIMARY KEY (id),
    CONSTRAINT chk_digests_week_start_monday CHECK (EXTRACT(ISODOW FROM week_start) = 1),
    CONSTRAINT chk_digests_untagged_not_negative CHECK (untagged_count >= 0),
    CONSTRAINT chk_digests_status_fields CHECK (
        (status = 'sending' AND sent_at IS NULL AND failure_reason IS NULL)
        OR (status = 'sent' AND sent_at IS NOT NULL AND failure_reason IS NULL)
        OR (status = 'failed' AND sent_at IS NULL AND failure_reason IS NOT NULL)
    )
);
COMMENT ON TABLE digests IS 'One generated weekly digest. Serves US-01-009.';
COMMENT ON COLUMN digests.id IS 'Surrogate key.';
COMMENT ON COLUMN digests.request_id IS 'Client-made id; a repeat returns this digest and never sends again (tenet 8).';
COMMENT ON COLUMN digests.status IS 'sending before SMTP is called, then sent or failed.';
COMMENT ON COLUMN digests.week_start IS 'Monday of the latest complete week the digest covers, in the brand timezone.';
COMMENT ON COLUMN digests.recipient_email IS 'Address the one email went to: the brand admin. [personal data]';
COMMENT ON COLUMN digests.untagged_count IS 'Reviews dated in the two compared weeks that were still untagged.';
COMMENT ON COLUMN digests.subject IS 'Email subject as sent.';
COMMENT ON COLUMN digests.body IS 'Email body as composed, quoting urgent reviews. [personal data]';
COMMENT ON COLUMN digests.failure_reason IS 'The SMTP error shown to the admin when sending failed.';
COMMENT ON COLUMN digests.sent_at IS 'When MailHog accepted the email.';
COMMENT ON COLUMN digests.created_at IS 'When the digest was generated and claimed.';
COMMENT ON COLUMN digests.updated_at IS 'Last status change.';
-- A repeated request (double click, retry) finds the first digest by its request id.
CREATE UNIQUE INDEX uq_digests_request_id ON digests (request_id);

-- model_calls: every live model call with its tokens and cost; the running total is their sum.
-- Serves US-02-001, US-02-002, US-02-004, US-02-005
CREATE TABLE budget.model_calls (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    purpose budget.model_call_purpose NOT NULL,
    model text,
    prompt_version integer,
    input_tokens integer,
    output_tokens integer,
    reserved_cost_usd numeric(12,8) NOT NULL,
    settled_cost_usd numeric(12,8),
    outcome budget.model_call_outcome NOT NULL DEFAULT 'reserved',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT model_calls_pkey PRIMARY KEY (id),
    CONSTRAINT chk_model_calls_costs_not_negative CHECK (reserved_cost_usd >= 0 AND (settled_cost_usd IS NULL OR settled_cost_usd >= 0)),
    CONSTRAINT chk_model_calls_tokens_not_negative CHECK ((input_tokens IS NULL OR input_tokens >= 0) AND (output_tokens IS NULL OR output_tokens >= 0)),
    CONSTRAINT chk_model_calls_settled_shape CHECK (
        outcome <> 'settled' OR (settled_cost_usd IS NOT NULL AND input_tokens IS NOT NULL AND output_tokens IS NOT NULL)
    ),
    CONSTRAINT chk_model_calls_reconciliation_shape CHECK (
        (purpose = 'reconciliation' AND model IS NULL AND prompt_version IS NULL AND outcome = 'settled')
        OR (purpose <> 'reconciliation' AND model IS NOT NULL)
    )
);
COMMENT ON TABLE budget.model_calls IS 'Every live model call and its cost; the running total. Never truncated or dropped. Serves US-02-001, US-02-002, US-02-004, US-02-005.';
COMMENT ON COLUMN budget.model_calls.id IS 'Surrogate key; the gateway settles the row it reserved by this id.';
COMMENT ON COLUMN budget.model_calls.purpose IS 'tagging, drafting, evaluation, tone_check, or reconciliation for a start-up adjustment.';
COMMENT ON COLUMN budget.model_calls.model IS 'Model identifier sent to OpenRouter; null only on a reconciliation row.';
COMMENT ON COLUMN budget.model_calls.prompt_version IS 'Number of the prompt version sent; the purpose says which prompt.';
COMMENT ON COLUMN budget.model_calls.input_tokens IS 'Input tokens OpenRouter reported; null until the call settles.';
COMMENT ON COLUMN budget.model_calls.output_tokens IS 'Output tokens OpenRouter reported; null until the call settles.';
COMMENT ON COLUMN budget.model_calls.reserved_cost_usd IS 'Worst-case price reserved before the call (input plus max_tokens); counts until settled.';
COMMENT ON COLUMN budget.model_calls.settled_cost_usd IS 'usage.cost from the response, in USD; replaces the reserved price in the total.';
COMMENT ON COLUMN budget.model_calls.outcome IS 'reserved while in flight, settled with a response, failed without one (stays at the reserved price).';
COMMENT ON COLUMN budget.model_calls.created_at IS 'When the call was reserved, which is the time the log shows.';
COMMENT ON COLUMN budget.model_calls.updated_at IS 'When the call settled or failed.';

COMMIT;
