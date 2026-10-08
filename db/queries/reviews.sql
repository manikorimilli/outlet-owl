-- Reviews: the searchable list (phase 4 LLD section 5). Every statement is
-- scoped by all_outlets OR outlet_id (tenet 3). Optional filters are null
-- when absent. Known full scan for the brand admin at about 10^4 rows.

-- name: ListReviews :many
SELECT r.id, r.outlet_id, o.name AS outlet_name, r.source, r.review_date, r.rating, r.review_text, r.reviewer_name,
       (t.review_id IS NOT NULL)::boolean AS tagged,
       coalesce(t.themes, '{}')::text[] AS themes,
       coalesce(t.sentiment::text, '')::text AS sentiment,
       coalesce(t.is_urgent, false) AS is_urgent,
       coalesce(t.urgent_reasons, '{}')::text[] AS urgent_reasons,
       coalesce(t.prompt_version, 0)::integer AS prompt_version
FROM reviews r
JOIN outlets o ON o.id = r.outlet_id
LEFT JOIN review_tags t ON t.review_id = r.id
WHERE (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint)
  AND (sqlc.narg(outlet_id)::bigint IS NULL OR r.outlet_id = sqlc.narg(outlet_id)::bigint)
  AND (sqlc.narg(q)::text IS NULL OR r.review_text ILIKE sqlc.narg(q)::text ESCAPE '\' OR r.reviewer_name ILIKE sqlc.narg(q)::text ESCAPE '\')
  AND (sqlc.narg(theme)::text IS NULL OR sqlc.narg(theme)::text = ANY (t.themes))
  AND (sqlc.narg(sentiment)::text IS NULL OR t.sentiment::text = sqlc.narg(sentiment)::text)
  AND (sqlc.narg(is_urgent)::boolean IS NULL OR t.is_urgent = sqlc.narg(is_urgent)::boolean)
  AND (sqlc.narg(date_from)::date IS NULL OR r.review_date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR r.review_date <= sqlc.narg(date_to)::date)
  AND (sqlc.narg(after_date)::date IS NULL OR (r.review_date, r.id) < (sqlc.narg(after_date)::date, sqlc.narg(after_id)::bigint))
ORDER BY r.review_date DESC, r.id DESC
LIMIT sqlc.arg(row_limit);

-- name: CountReviews :one
SELECT count(*)::integer
FROM reviews r
LEFT JOIN review_tags t ON t.review_id = r.id
WHERE (sqlc.arg(all_outlets)::boolean OR r.outlet_id = sqlc.arg(scope_outlet_id)::bigint)
  AND (sqlc.narg(outlet_id)::bigint IS NULL OR r.outlet_id = sqlc.narg(outlet_id)::bigint)
  AND (sqlc.narg(q)::text IS NULL OR r.review_text ILIKE sqlc.narg(q)::text ESCAPE '\' OR r.reviewer_name ILIKE sqlc.narg(q)::text ESCAPE '\')
  AND (sqlc.narg(theme)::text IS NULL OR sqlc.narg(theme)::text = ANY (t.themes))
  AND (sqlc.narg(sentiment)::text IS NULL OR t.sentiment::text = sqlc.narg(sentiment)::text)
  AND (sqlc.narg(is_urgent)::boolean IS NULL OR t.is_urgent = sqlc.narg(is_urgent)::boolean)
  AND (sqlc.narg(date_from)::date IS NULL OR r.review_date >= sqlc.narg(date_from)::date)
  AND (sqlc.narg(date_to)::date IS NULL OR r.review_date <= sqlc.narg(date_to)::date);
