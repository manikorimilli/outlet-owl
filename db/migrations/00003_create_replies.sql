-- Migration: 00003_create_replies   Task: none (HLD build phase 5, LLD work item 1)
-- Store: postgres      Phase: expand (1 of 1)
-- Purpose: create reply_status and replies so managers can draft, edit and mark replies (US-00-002, US-00-003).
-- Locks: Up: a new table referencing reviews and users (SHARE ROW EXCLUSIVE on each for the foreign keys, instant at demo size);
--        Down: drops the table and type, instant.
-- Rows: at most one per review, 10^3 to 10^4 (data model section 4); backfill none.
-- Index: replies_pkey serves every claim and write by review id; idx_replies_replied_by the foreign key.
-- Retention / PII: kept; draft_text and reply_text may greet the reviewer by name.
-- Down: drops replies and reply_status; Down loses: EVERY DRAFT AND EVERY REPLIED STATE. Reviews then show as not replied
--       and drafting again spends model budget unless the recordings match.
--       tested in: internal/store/replies_integration_test.go (TestMigration00003_UpDownUp).
-- Copied from docs/design/schema.sql; a difference between the two is a bug in this file.

-- +goose Up
SET lock_timeout = '2s';
SET statement_timeout = '60s';

CREATE TYPE reply_status AS ENUM ('drafting', 'draft', 'replied');

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

-- +goose Down
SET lock_timeout = '2s';
DROP TABLE replies;
DROP TYPE reply_status;
