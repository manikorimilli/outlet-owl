# Queries

sqlc input (ADR-0003): one file per area, plain SQL with `-- name:` comments.
`make sqlc` writes the typed Go into internal/store. Every outlet query carries
the caller's outlet condition (tenet 3).
