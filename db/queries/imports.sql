-- Imports: the CSV upload and its result (phase 3 LLD, section 7).

-- name: GetImportByRequestID :one
-- A repeat of an upload finds the first result; uq_imports_request_id.
SELECT id, file_name, imported_count, duplicate_count, rejected_count, created_at
FROM imports
WHERE request_id = (sqlc.arg(request_id)::text)::uuid;

-- name: ListImportRejections :many
-- The rejected rows of one import, in row order; import_rejections_pkey.
SELECT row_number, reason
FROM import_rejections
WHERE import_id = sqlc.arg(import_id)
ORDER BY row_number;

-- name: CreateImport :one
-- No row when the request id exists; the insert waits for an import with
-- the same id still in flight (uq_imports_request_id).
INSERT INTO imports (request_id, file_name)
VALUES ((sqlc.arg(request_id)::text)::uuid, sqlc.arg(file_name))
ON CONFLICT (request_id) DO NOTHING
RETURNING id, created_at;

-- name: InsertReviews :many
-- The valid rows of one import, in file order so review ids follow the file.
-- One unnest per array, joined on the position: a subscript on a text array
-- walks it from the start, which is quadratic over a 5 MB file.
-- A row already stored (same outlet, source, date, reviewer and text) returns
-- nothing and is counted as a duplicate; uq_reviews_natural_key.
INSERT INTO reviews (outlet_id, import_id, source, review_date, rating, review_text, reviewer_name)
SELECT o.v, sqlc.arg(import_id)::bigint, s.v, d.v, r.v, t.v, n.v
FROM unnest(sqlc.arg(outlet_ids)::bigint[]) WITH ORDINALITY AS o (v, i)
JOIN unnest(sqlc.arg(sources)::text[]) WITH ORDINALITY AS s (v, i) USING (i)
JOIN unnest(sqlc.arg(review_dates)::date[]) WITH ORDINALITY AS d (v, i) USING (i)
JOIN unnest(sqlc.arg(ratings)::smallint[]) WITH ORDINALITY AS r (v, i) USING (i)
JOIN unnest(sqlc.arg(review_texts)::text[]) WITH ORDINALITY AS t (v, i) USING (i)
JOIN unnest(sqlc.arg(reviewer_names)::text[]) WITH ORDINALITY AS n (v, i) USING (i)
ORDER BY i
ON CONFLICT (outlet_id, source, review_date, reviewer_name, md5(review_text)) DO NOTHING
RETURNING id;

-- name: InsertImportRejections :exec
INSERT INTO import_rejections (import_id, row_number, reason)
SELECT sqlc.arg(import_id)::bigint, n.v, r.v
FROM unnest(sqlc.arg(row_numbers)::integer[]) WITH ORDINALITY AS n (v, i)
JOIN unnest(sqlc.arg(reasons)::text[]) WITH ORDINALITY AS r (v, i) USING (i);

-- name: SetImportCounts :exec
UPDATE imports
SET imported_count = sqlc.arg(imported_count), duplicate_count = sqlc.arg(duplicate_count),
    rejected_count = sqlc.arg(rejected_count)
WHERE id = sqlc.arg(id);
