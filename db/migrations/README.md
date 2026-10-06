# Migrations

goose SQL migrations (ADR-0003), numbered and append-only once applied. The
plan is in docs/design/data-model.md section 8; the first file is
`00001_create_outlets_and_users.sql`. The budget schema
(`budget.model_calls`) gets its own migration set with no Down
(HLD section 4); it arrives with the model gateway.
