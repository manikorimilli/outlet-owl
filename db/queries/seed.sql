-- Seed: the demo reset and its reviews (HLD flow D, data model section 5).
-- The seed holds the tagging lock while it runs; the budget schema is never
-- touched.

-- name: ResetDomainTables :exec
TRUNCATE digests, replies, review_tags, import_rejections, reviews, imports, users, outlets;

-- name: RestartReviewIDs :exec
-- Review ids, batch order and texts must match the recordings on every seed.
ALTER TABLE reviews ALTER COLUMN id RESTART WITH 1;

-- name: InsertSeedReviews :execrows
INSERT INTO reviews (outlet_id, source, review_date, rating, review_text, reviewer_name)
SELECT o.v, s.v, d.v, r.v, t.v, n.v
FROM unnest(sqlc.arg(outlet_ids)::bigint[]) WITH ORDINALITY AS o (v, i)
JOIN unnest(sqlc.arg(sources)::text[]) WITH ORDINALITY AS s (v, i) USING (i)
JOIN unnest(sqlc.arg(review_dates)::date[]) WITH ORDINALITY AS d (v, i) USING (i)
JOIN unnest(sqlc.arg(ratings)::smallint[]) WITH ORDINALITY AS r (v, i) USING (i)
JOIN unnest(sqlc.arg(review_texts)::text[]) WITH ORDINALITY AS t (v, i) USING (i)
JOIN unnest(sqlc.arg(reviewer_names)::text[]) WITH ORDINALITY AS n (v, i) USING (i)
ORDER BY i;
