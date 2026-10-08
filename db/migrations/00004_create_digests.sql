-- Migration: 00004_create_digests   Task: none (HLD build phase 6)
-- Store: postgres      Phase: expand (1 of 1)
-- Purpose: create digest_status and digests so the weekly digest is claimed before it is sent (US-01-009, tenet 8).
-- Locks: Up: a new table with no foreign keys, no lock on existing data; Down: drops the table and type, instant.
-- Rows: one per generated digest, tens a year; backfill none.
-- Index: uq_digests_request_id serves the repeat lookup and the claim.
-- Retention / PII: kept; recipient_email and body (quotes urgent reviews) are personal data.
-- Down: drops digests and digest_status; Down loses: THE RECORD OF EVERY DIGEST SENT. A repeat of an old request id
--       would then send again.
--       tested in: internal/store/digests_integration_test.go (TestMigration00004_UpDownUp).
-- Copied from docs/design/schema.sql; a difference between the two is a bug in this file.

-- +goose Up
SET lock_timeout = '2s';
SET statement_timeout = '60s';

CREATE TYPE digest_status AS ENUM ('sending', 'sent', 'failed');


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

-- +goose Down
SET lock_timeout = '2s';
DROP TABLE digests;
DROP TYPE digest_status;
