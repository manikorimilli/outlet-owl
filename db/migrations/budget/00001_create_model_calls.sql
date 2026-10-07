-- Migration: budget/00001_create_model_calls        Task: none (HLD build phase 2, gateway work item G1)
-- Store: postgres      Phase: expand (1 of 1)
-- Purpose: create the budget schema and the model call log, the running total every live model call is checked
--          against (US-02-001, REQ-030, REQ-031, Q-015, tenet 2).
-- Locks: Up: creates a new schema and table only, no lock on existing data.
-- Rows: one per live HTTP attempt, about 10^3 to 10^4 for the product's life (docs/design/data-model.md section 4).
-- Index: none by design; the running total sums at most about 10^4 rows (data model section 9).
-- Retention / PII: kept forever, no personal data; the seed never truncates this schema.
-- Down: NONE. This set has no Down on purpose: no back-out may drop the running total (HLD section 4). Applied
--       with its own version table: goose -dir db/migrations/budget -table goose_budget_version up.
--       tested in: internal/store/budget_migration_test.go (TestBudgetMigrations_HaveNoDown).
-- Copied from docs/design/schema.sql; a difference between the two is a bug in this file.

-- +goose Up
SET lock_timeout = '2s';
SET statement_timeout = '60s';

CREATE SCHEMA budget;

CREATE TYPE budget.model_call_purpose AS ENUM ('tagging', 'drafting', 'evaluation', 'tone_check', 'reconciliation');
CREATE TYPE budget.model_call_outcome AS ENUM ('reserved', 'settled', 'failed');

CREATE TABLE budget.model_calls (
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    purpose budget.model_call_purpose NOT NULL,
    model text,
    prompt_version integer,
    input_tokens integer,
    output_tokens integer,
    reserved_cost_usd numeric(12,8) NOT NULL,
    settled_cost_usd numeric(12,8),
    outcome budget.model_call_outcome NOT NULL DEFAULT 'reserved',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT model_calls_pkey PRIMARY KEY (id),
    CONSTRAINT chk_model_calls_costs_not_negative CHECK (reserved_cost_usd >= 0 AND (settled_cost_usd IS NULL OR settled_cost_usd >= 0)),
    CONSTRAINT chk_model_calls_tokens_not_negative CHECK ((input_tokens IS NULL OR input_tokens >= 0) AND (output_tokens IS NULL OR output_tokens >= 0)),
    CONSTRAINT chk_model_calls_settled_shape CHECK (
        outcome <> 'settled' OR (settled_cost_usd IS NOT NULL AND input_tokens IS NOT NULL AND output_tokens IS NOT NULL)
    ),
    CONSTRAINT chk_model_calls_reconciliation_shape CHECK (
        (purpose = 'reconciliation' AND model IS NULL AND prompt_version IS NULL AND outcome = 'settled')
        OR (purpose <> 'reconciliation' AND model IS NOT NULL)
    )
);
COMMENT ON TABLE budget.model_calls IS 'Every live model call and its cost; the running total. Never truncated or dropped. Serves US-02-001, US-02-002, US-02-004, US-02-005.';
COMMENT ON COLUMN budget.model_calls.id IS 'Surrogate key; the gateway settles the row it reserved by this id.';
COMMENT ON COLUMN budget.model_calls.purpose IS 'tagging, drafting, evaluation, tone_check, or reconciliation for a start-up adjustment.';
COMMENT ON COLUMN budget.model_calls.model IS 'Model identifier sent to OpenRouter; null only on a reconciliation row.';
COMMENT ON COLUMN budget.model_calls.prompt_version IS 'Number of the prompt version sent; the purpose says which prompt.';
COMMENT ON COLUMN budget.model_calls.input_tokens IS 'Input tokens OpenRouter reported; null until the call settles.';
COMMENT ON COLUMN budget.model_calls.output_tokens IS 'Output tokens OpenRouter reported; null until the call settles.';
COMMENT ON COLUMN budget.model_calls.reserved_cost_usd IS 'Worst-case price reserved before the call (input plus max_tokens); counts until settled.';
COMMENT ON COLUMN budget.model_calls.settled_cost_usd IS 'usage.cost from the response, in USD; replaces the reserved price in the total.';
COMMENT ON COLUMN budget.model_calls.outcome IS 'reserved while in flight, settled with a response, failed without one (stays at the reserved price).';
COMMENT ON COLUMN budget.model_calls.created_at IS 'When the call was reserved, which is the time the log shows.';
COMMENT ON COLUMN budget.model_calls.updated_at IS 'When the call settled or failed.';
