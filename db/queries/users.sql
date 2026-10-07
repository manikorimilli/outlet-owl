-- Users: sign-in lookups and the users file sync (phase 1 server LLD, section 5).
-- Emails and outlet names are compared with lower() inside SQL, so callers pass
-- them as written.

-- name: GetActiveUserByEmail :one
-- Sign-in by email ignoring case; uq_users_email_lower.
SELECT u.id, u.email, u.name, u.role, u.outlet_id, o.name AS outlet_name, u.password_hash, u.created_at
FROM users u
LEFT JOIN outlets o ON o.id = u.outlet_id
WHERE lower(u.email) = lower(sqlc.arg(email)::text)
  AND u.removed_at IS NULL;

-- name: GetActiveUserByID :one
-- The user re-read on every signed-in request (tenet 3); users_pkey.
SELECT u.id, u.email, u.name, u.role, u.outlet_id, o.name AS outlet_name, u.password_hash, u.created_at
FROM users u
LEFT JOIN outlets o ON o.id = u.outlet_id
WHERE u.id = sqlc.arg(id)
  AND u.removed_at IS NULL;

-- name: ResolveOutletsByName :many
-- The outlets the users file's managers name; uq_outlets_name_lower.
SELECT id, name
FROM outlets
WHERE lower(name) IN (SELECT lower(n) FROM unnest(sqlc.arg(names)::text[]) AS n);

-- name: MarkUsersRemovedExcept :execrows
-- Accounts no longer in the file. The list is never empty (the file always
-- keeps its one admin): "<> ALL" over an empty list is true for every row.
-- Known full scan, about 10 rows.
UPDATE users
SET removed_at = now(), updated_at = now()
WHERE removed_at IS NULL
  AND lower(email) <> ALL (SELECT lower(e) FROM unnest(sqlc.arg(emails)::text[]) AS e);

-- name: UpsertUser :execrows
-- One file entry; changes nothing (0 rows) when every field already matches.
-- uq_users_email_lower.
INSERT INTO users (email, name, role, outlet_id, password_hash)
VALUES (sqlc.arg(email), sqlc.arg(name), sqlc.arg(role), sqlc.narg(outlet_id), sqlc.arg(password_hash))
ON CONFLICT (lower(email)) DO UPDATE
SET email = EXCLUDED.email,
    name = EXCLUDED.name,
    role = EXCLUDED.role,
    outlet_id = EXCLUDED.outlet_id,
    password_hash = EXCLUDED.password_hash,
    removed_at = NULL,
    updated_at = now()
WHERE (users.email, users.name, users.role, users.outlet_id, users.password_hash, users.removed_at)
    IS DISTINCT FROM (EXCLUDED.email, EXCLUDED.name, EXCLUDED.role, EXCLUDED.outlet_id, EXCLUDED.password_hash, NULL::timestamptz);
