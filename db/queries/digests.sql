-- Digests: claimed under the request id before SMTP, then sent or failed
-- (HLD flow C, data model section 5); uq_digests_request_id.

-- name: GetDigestByRequestID :one
SELECT id, status::text AS status, week_start, recipient_email, untagged_count, subject, body, failure_reason, sent_at, created_at
FROM digests
WHERE request_id = (sqlc.arg(request_id)::text)::uuid;

-- name: ClaimDigest :one
-- No row when the request id exists; waits for one still in flight.
INSERT INTO digests (request_id, week_start, recipient_email, untagged_count, subject, body)
VALUES ((sqlc.arg(request_id)::text)::uuid, sqlc.arg(week_start), sqlc.arg(recipient_email), sqlc.arg(untagged_count),
        sqlc.arg(subject), sqlc.arg(body))
ON CONFLICT (request_id) DO NOTHING
RETURNING id, created_at;

-- name: MarkDigestSent :one
UPDATE digests SET status = 'sent', sent_at = now(), updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'sending'
RETURNING sent_at;

-- name: MarkDigestFailed :exec
UPDATE digests SET status = 'failed', failure_reason = sqlc.arg(reason)::text, updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'sending';

-- name: ActiveBrandAdminEmail :one
-- The one active brand admin (uq_users_one_brand_admin), the digest's recipient.
SELECT email FROM users WHERE role = 'brand_admin' AND removed_at IS NULL;
