-- Outlets: the list and the add form (phase 1 server LLD, section 5).

-- name: ListOutlets :many
-- The caller's outlets: every outlet for the brand admin (all_outlets), else
-- the manager's one outlet. The scope is an explicit boolean plus an id, never
-- "null means all" (tenet 3). Known full scan, at most about 25 rows.
SELECT id, name, created_at
FROM outlets
WHERE (sqlc.arg(all_outlets)::boolean OR id = sqlc.arg(outlet_id)::bigint)
ORDER BY lower(name), id;

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
