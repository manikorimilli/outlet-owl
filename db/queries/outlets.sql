-- Outlets: the list and the add form (phase 1 server LLD, section 5).

-- name: ListOutlets :many
-- The caller's outlets: every outlet for the brand admin (all_outlets), else
-- the manager's one outlet. The scope is an explicit boolean plus an id, never
-- "null means all" (tenet 3). Known full scan, at most about 25 rows.
-- The two counts read reviews by outlet (idx_reviews_outlet_id_review_date)
-- and the untagged anti-join (review_tags_pkey).
SELECT o.id, o.name, o.created_at,
       (SELECT count(*) FROM reviews r WHERE r.outlet_id = o.id)::integer AS review_count,
       (SELECT count(*) FROM reviews r
        WHERE r.outlet_id = o.id
          AND NOT EXISTS (SELECT 1 FROM review_tags t WHERE t.review_id = r.id))::integer AS untagged_count
FROM outlets o
WHERE (sqlc.arg(all_outlets)::boolean OR o.id = sqlc.arg(outlet_id)::bigint)
ORDER BY lower(o.name), o.id;

-- name: ListActiveManagers :many
-- The active managers of the listed outlets; idx_users_outlet_id.
SELECT id, name, outlet_id
FROM users
WHERE outlet_id = ANY (sqlc.arg(outlet_ids)::bigint[])
  AND role = 'outlet_manager'
  AND removed_at IS NULL
ORDER BY lower(name), id;

-- name: CreateOutlet :one
-- No row when the name exists ignoring case; uq_outlets_name_lower.
INSERT INTO outlets (name)
VALUES (sqlc.arg(name))
ON CONFLICT (lower(name)) DO NOTHING
RETURNING id, name, created_at;

-- name: GetOutletByName :one
-- The existing outlet after a CreateOutlet conflict; uq_outlets_name_lower.
SELECT id, name, created_at
FROM outlets
WHERE lower(name) = lower(sqlc.arg(name)::text);

-- name: ListOutletNames :many
-- Every outlet, for matching the CSV outlet column by name ignoring case.
-- Known full scan, at most about 25 rows.
SELECT id, name
FROM outlets
ORDER BY id;
