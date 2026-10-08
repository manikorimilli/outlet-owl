-- Replies: the claim, the draft, edits and marking replied (phase 5 LLD
-- section 3). Every statement reaches the reply through its review and the
-- caller's outlet (tenet 3); replies_pkey and reviews_pkey.

-- name: GetReviewDetail :one
SELECT r.id, r.outlet_id, o.name AS outlet_name, r.source, r.review_date, r.rating, r.review_text, r.reviewer_name,
       (t.review_id IS NOT NULL)::boolean AS tagged,
       coalesce(t.themes, '{}')::text[] AS themes,
       coalesce(t.sentiment::text, '')::text AS sentiment,
       coalesce(t.is_urgent, false) AS is_urgent,
       coalesce(t.urgent_reasons, '{}')::text[] AS urgent_reasons,
       coalesce(t.prompt_version, 0)::integer AS tag_prompt_version
FROM reviews r
JOIN outlets o ON o.id = r.outlet_id
LEFT JOIN review_tags t ON t.review_id = r.id
WHERE r.id = sqlc.arg(id)
  AND (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint);

-- name: GetReply :one
SELECT rp.status::text AS status, rp.draft_text, rp.prompt_version, rp.reply_text, rp.replied_at, rp.updated_at,
       u.id AS replied_by_id, u.name AS replied_by_name
FROM replies rp
LEFT JOIN users u ON u.id = rp.replied_by
WHERE rp.review_id = sqlc.arg(review_id);

-- name: ClaimDraft :one
-- No row when a reply exists or the review is not in the manager's outlet.
INSERT INTO replies (review_id)
SELECT r.id FROM reviews r WHERE r.id = sqlc.arg(review_id) AND r.outlet_id = sqlc.arg(outlet_id)
ON CONFLICT (review_id) DO NOTHING
RETURNING updated_at;

-- name: TakeOverClaim :one
-- A drafting claim older than 60 seconds belongs to a request that died.
UPDATE replies SET updated_at = now()
WHERE review_id = sqlc.arg(review_id) AND status = 'drafting' AND updated_at < now() - interval '60 seconds'
RETURNING updated_at;

-- name: LandDraft :execrows
UPDATE replies
SET status = 'draft', draft_text = sqlc.arg(text)::text, reply_text = sqlc.arg(text)::text,
    prompt_version = sqlc.arg(prompt_version), updated_at = now()
WHERE review_id = sqlc.arg(review_id) AND status = 'drafting' AND updated_at = sqlc.arg(claimed);

-- name: ReleaseClaim :exec
DELETE FROM replies
WHERE review_id = sqlc.arg(review_id) AND status = 'drafting' AND updated_at = sqlc.arg(claimed);

-- name: InsertHandReply :execrows
-- A reply written by hand when there is none (drafting unavailable).
INSERT INTO replies (review_id, status, reply_text)
SELECT r.id, 'draft', sqlc.arg(text)::text FROM reviews r
WHERE r.id = sqlc.arg(review_id) AND r.outlet_id = sqlc.arg(outlet_id)
ON CONFLICT (review_id) DO NOTHING;

-- name: SaveReplyText :execrows
UPDATE replies rp SET reply_text = sqlc.arg(text)::text, updated_at = now()
FROM reviews r
WHERE rp.review_id = sqlc.arg(review_id) AND r.id = rp.review_id AND r.outlet_id = sqlc.arg(outlet_id)
  AND rp.status = 'draft' AND rp.updated_at = sqlc.arg(seen);

-- name: MarkReplied :execrows
UPDATE replies rp
SET status = 'replied', reply_text = sqlc.arg(text)::text, replied_by = sqlc.arg(user_id), replied_at = now(), updated_at = now()
FROM reviews r
WHERE rp.review_id = sqlc.arg(review_id) AND r.id = rp.review_id AND r.outlet_id = sqlc.arg(outlet_id)
  AND rp.status = 'draft' AND rp.updated_at = sqlc.arg(seen);
