-- Migration: 00002_create_imports_reviews_and_tags   Task: none (HLD build phase 3, LLD work item 1)
-- Store: postgres      Phase: expand (1 of 1)
-- Purpose: create sentiment, imports, import_rejections, reviews and review_tags so CSV imports can be stored
--          and tagged (US-01-002, US-01-003, US-01-004).
-- Locks: Up: creates new tables only and references outlets (a SHARE ROW EXCLUSIVE lock on outlets for the
--        foreign key, instant at tens of rows); Down: drops the five objects, instant on the demo size.
-- Rows: reviews 1,500 in the seed, about 15,000 a year at the 25-outlet rate (data model section 4); backfill none.
-- Index: uq_imports_request_id serves the repeat lookup; uq_reviews_natural_key the de-duplication;
--        idx_reviews_outlet_id_review_date the per-outlet date ranges; idx_reviews_import_id the foreign key;
--        review_tags_pkey the untagged anti-join (data model section 4).
-- Retention / PII: kept; reviews.review_text and reviews.reviewer_name are personal data.
-- Down: drops review_tags, reviews, import_rejections, imports and sentiment; Down loses: EVERY IMPORTED REVIEW,
--       EVERY TAG RESULT AND EVERY IMPORT RESULT. Recovery is importing the files again, and tagging them again
--       spends model budget unless the recordings match.
--       tested in: internal/store/imports_integration_test.go (TestMigration00002_UpDownUp).
-- Copied from docs/design/schema.sql; a difference between the two is a bug in this file.

-- +goose Up
SET lock_timeout = '2s';
SET statement_timeout = '60s';

CREATE TYPE sentiment AS ENUM ('positive', 'neutral', 'negative');

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
    urgent_reasons text[] NOT NULL DEFAULT '{}',
    prompt_version integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT review_tags_pkey PRIMARY KEY (review_id),
    CONSTRAINT chk_review_tags_themes_no_nulls CHECK (array_position(themes, NULL) IS NULL),
    CONSTRAINT chk_review_tags_prompt_version_positive CHECK (prompt_version >= 1),
    CONSTRAINT chk_review_tags_urgent_matches_reasons CHECK (is_urgent = (cardinality(urgent_reasons) > 0)),
    CONSTRAINT chk_review_tags_urgent_reasons_allowed CHECK (urgent_reasons <@ ARRAY['food_safety', 'harassment', 'legal_threat']::text[]),
    CONSTRAINT chk_review_tags_urgent_reasons_distinct CHECK (
        array_position(urgent_reasons, NULL) IS NULL
        AND cardinality(array_remove(urgent_reasons, 'food_safety')) >= cardinality(urgent_reasons) - 1
        AND cardinality(array_remove(urgent_reasons, 'harassment')) >= cardinality(urgent_reasons) - 1
        AND cardinality(array_remove(urgent_reasons, 'legal_threat')) >= cardinality(urgent_reasons) - 1
    )
);
COMMENT ON TABLE review_tags IS 'The stored tag result of a review; written once, never re-tagged. Serves US-01-003 to US-01-009, US-02-002.';
COMMENT ON COLUMN review_tags.review_id IS 'The review the result line named; one result per review.';
COMMENT ON COLUMN review_tags.themes IS 'Theme codes from the configured list; zero, one or several.';
COMMENT ON COLUMN review_tags.sentiment IS 'One label: positive, neutral or negative.';
COMMENT ON COLUMN review_tags.is_urgent IS 'True when the review concerns food safety, harassment or a legal threat; true exactly when urgent_reasons is not empty.';
COMMENT ON COLUMN review_tags.urgent_reasons IS 'Why the review is urgent: food_safety, harassment, legal_threat; several allowed, none repeated, empty when not urgent.';
COMMENT ON COLUMN review_tags.prompt_version IS 'Number of the tagging prompt version that produced this result.';
COMMENT ON COLUMN review_tags.created_at IS 'When the result was stored.';

-- +goose Down
SET lock_timeout = '2s';
DROP TABLE review_tags;
DROP TABLE reviews;
DROP TABLE import_rejections;
DROP TABLE imports;
DROP TYPE sentiment;
