-- Tagging: the worker's snapshot, the texts it sends and the results it
-- stores (phase 3 LLD, sections 6 and 7).

-- name: UntaggedReviewIDs :many
-- Reviews with no stored result after the given id, in id order: the
-- worker's pending work (ADR-0006). reviews_pkey and the review_tags_pkey
-- anti-join.
SELECT r.id
FROM reviews r
WHERE r.id > sqlc.arg(after_id)::bigint
  AND NOT EXISTS (SELECT 1 FROM review_tags t WHERE t.review_id = r.id)
ORDER BY r.id;

-- name: ReviewTexts :many
-- The texts of one batch; reviews_pkey.
SELECT id, review_text
FROM reviews
WHERE id = ANY (sqlc.arg(ids)::bigint[])
ORDER BY id;

-- name: InsertReviewTag :execrows
-- One result per review, never replaced (REQ-011, tenet 4); review_tags_pkey.
INSERT INTO review_tags (review_id, themes, sentiment, is_urgent, urgent_reasons, prompt_version)
VALUES (sqlc.arg(review_id), sqlc.arg(themes)::text[], sqlc.arg(sentiment)::sentiment, sqlc.arg(is_urgent),
        sqlc.arg(urgent_reasons)::text[], sqlc.arg(prompt_version))
ON CONFLICT (review_id) DO NOTHING;

-- name: LockTagging :exec
-- Blocks until this session holds the tagging lock (ADR-0006). The caller
-- runs it on a connection of its own and closes that connection to release.
SELECT pg_advisory_lock(sqlc.arg(key)::bigint);
