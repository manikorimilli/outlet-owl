-- Migration: 00001_create_outlets_and_users        Task: none (HLD build phase 1, server work item 1)
-- Store: postgres      Phase: expand (1 of 1)
-- Purpose: create user_role, outlets and users so the users file can be loaded and outlets added (US-01-001, US-00-001).
-- Locks: Up: creates new tables only, no lock on existing data; Down: drops the two tables, instant on the demo size.
-- Rows: outlets ~5 and users ~6 in the demo, at most tens (source: docs/design/data-model.md section 4); backfill none.
-- Index: uq_outlets_name_lower serves the outlet lookup by name ignoring case (CSV import, add-outlet conflict);
--        uq_users_email_lower serves sign-in by email and the users file upsert;
--        uq_users_one_brand_admin keeps one active brand admin (Q-006);
--        idx_users_outlet_id serves the managers of an outlet and the foreign key (phase 1 LLD section 5).
-- Retention / PII: users kept and marked removed when gone from the users file; users.email and users.name are personal data.
-- Down: drops users, outlets and user_role; Down loses: EVERY ACCOUNT AND EVERY OUTLET. Sign-in then fails until
--       the migration is applied again and the server restarts (it reloads the users file); outlets must be re-added.
--       tested in: internal/store/migrations_integration_test.go (TestMigration00001_UpDownUp).
-- Copied from docs/design/schema.sql; a difference between the two is a bug in this file.

-- +goose Up
SET lock_timeout = '2s';
SET statement_timeout = '60s';

CREATE TYPE user_role AS ENUM ('brand_admin', 'outlet_manager');

CREATE TABLE outlets (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT outlets_pkey PRIMARY KEY (id),
    CONSTRAINT chk_outlets_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT chk_outlets_name_trimmed CHECK (name = btrim(name))
);
COMMENT ON TABLE outlets IS 'One location of the brand. Serves US-01-001, US-01-002, US-01-005, US-01-006, US-01-007, US-00-001.';
COMMENT ON COLUMN outlets.id IS 'Surrogate key; reviews and managers reference it.';
COMMENT ON COLUMN outlets.name IS 'Outlet name as the admin typed it; the CSV outlet column must match it, ignoring case.';
COMMENT ON COLUMN outlets.created_at IS 'When the outlet was added.';
COMMENT ON COLUMN outlets.updated_at IS 'Last change to this row.';
-- Case-insensitive unique name: the CSV outlet column is matched by it, and a repeated add returns the existing outlet.
CREATE UNIQUE INDEX uq_outlets_name_lower ON outlets (lower(name));

CREATE TABLE users (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    email text NOT NULL,
    name text NOT NULL,
    role user_role NOT NULL,
    outlet_id bigint REFERENCES outlets (id) ON DELETE RESTRICT,
    password_hash text NOT NULL,
    removed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT chk_users_email_shape CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+$'),
    CONSTRAINT chk_users_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT chk_users_role_outlet CHECK (
        (role = 'brand_admin' AND outlet_id IS NULL)
        OR (role = 'outlet_manager' AND outlet_id IS NOT NULL)
    )
);
COMMENT ON TABLE users IS 'An account from the users file; one brand admin and outlet managers. Serves US-00-001, US-00-003, US-01-001, US-01-009, US-02-006.';
COMMENT ON COLUMN users.id IS 'Surrogate key; the only identity the JWT carries (ADR-0007).';
COMMENT ON COLUMN users.email IS 'Sign-in name and the digest recipient address. [personal data]';
COMMENT ON COLUMN users.name IS 'Name shown in the top bar and beside a reply the manager marked replied. [personal data]';
COMMENT ON COLUMN users.role IS 'brand_admin or outlet_manager, re-read on every request (ADR-0007).';
COMMENT ON COLUMN users.outlet_id IS 'The one outlet an outlet manager may see and act on; null for the brand admin.';
COMMENT ON COLUMN users.password_hash IS 'bcrypt hash from the users file; never the password.';
COMMENT ON COLUMN users.removed_at IS 'Set when the account left the users file; sign-in is refused, the row stays so replies keep their manager.';
COMMENT ON COLUMN users.created_at IS 'When the account was first loaded.';
COMMENT ON COLUMN users.updated_at IS 'Last change from the users file.';
-- Sign-in looks the account up by email, ignoring case; the users file upsert matches on it.
CREATE UNIQUE INDEX uq_users_email_lower ON users (lower(email));
-- One active brand admin: the digest goes to the brand admin (Q-006).
CREATE UNIQUE INDEX uq_users_one_brand_admin ON users (role) WHERE role = 'brand_admin' AND removed_at IS NULL;
-- Foreign-key index: the managers of an outlet (outlets screen), and no table scan on an outlet delete check.
CREATE INDEX idx_users_outlet_id ON users (outlet_id);

-- +goose Down
SET lock_timeout = '2s';
DROP TABLE users;
DROP TABLE outlets;
DROP TYPE user_role;
