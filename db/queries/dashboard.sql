-- Dashboard reports (phase 4 LLD section 5). Every statement is scoped by
-- all_outlets OR outlet_id (tenet 3); weeks start on Monday
-- (date_trunc('week')). Known full scans for the brand admin at about 10^4 rows.

-- name: WeeklyOutletStats :many
SELECT r.outlet_id,
       date_trunc('week', r.review_date)::date AS week_start,
       count(*)::integer AS review_count,
       round(avg(r.rating), 1)::float8 AS average_rating,
       (count(*) FILTER (WHERE t.sentiment = 'positive'))::integer AS positive,
       (count(*) FILTER (WHERE t.sentiment = 'neutral'))::integer AS neutral,
       (count(*) FILTER (WHERE t.sentiment = 'negative'))::integer AS negative,
       (count(*) FILTER (WHERE t.review_id IS NULL))::integer AS untagged,
       (count(*) FILTER (WHERE rp.status = 'replied'))::integer AS replied
FROM reviews r
LEFT JOIN review_tags t ON t.review_id = r.id
LEFT JOIN replies rp ON rp.review_id = r.id
WHERE (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint)
  AND r.review_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
GROUP BY r.outlet_id, week_start;

-- name: NegativeThemeCounts :many
SELECT r.outlet_id,
       date_trunc('week', r.review_date)::date AS week_start,
       theme::text AS theme,
       count(*)::integer AS negative
FROM reviews r
JOIN review_tags t ON t.review_id = r.id
CROSS JOIN LATERAL unnest(t.themes) AS theme
WHERE (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint)
  AND t.sentiment = 'negative'
  AND r.review_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
GROUP BY r.outlet_id, week_start, theme;

-- name: ReviewCounts :one
SELECT count(*)::integer AS total,
       (count(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM review_tags t WHERE t.review_id = r.id)))::integer AS untagged
FROM reviews r
WHERE (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint)
  AND r.review_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date;

-- name: UntaggedCount :one
-- The tagging status; review_tags_pkey anti-join.
SELECT count(*)::integer
FROM reviews r
WHERE (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint)
  AND NOT EXISTS (SELECT 1 FROM review_tags t WHERE t.review_id = r.id);
