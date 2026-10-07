-- Model call log in the budget schema (phase 2 LLD section 5). Costs cross
-- as decimal strings; the comparison with the limit happens here, in numeric.
-- No index: the running total sums at most about 10^4 rows (data model
-- section 9).

-- name: ReserveModelCall :one
-- Reserves the call's worst-case price only while the running total is at
-- most the limit, in one statement, so the total and the new row come from
-- the same snapshot. No row means the budget is used up.
INSERT INTO budget.model_calls (purpose, model, prompt_version, reserved_cost_usd)
SELECT sqlc.arg(purpose)::budget.model_call_purpose, sqlc.arg(model)::text, sqlc.narg(prompt_version)::integer,
       CAST(sqlc.arg(reserved_cost_usd)::text AS numeric)
WHERE (SELECT coalesce(sum(coalesce(settled_cost_usd, reserved_cost_usd)), 0) FROM budget.model_calls)
      <= CAST(sqlc.arg(limit_usd)::text AS numeric)
RETURNING id;

-- name: SettleModelCall :execrows
UPDATE budget.model_calls
SET outcome = 'settled', input_tokens = sqlc.arg(input_tokens)::integer, output_tokens = sqlc.arg(output_tokens)::integer,
    settled_cost_usd = CAST(sqlc.arg(settled_cost_usd)::text AS numeric), updated_at = now()
WHERE id = sqlc.arg(id) AND outcome = 'reserved';

-- name: FailModelCall :execrows
UPDATE budget.model_calls
SET outcome = 'failed', updated_at = now()
WHERE id = sqlc.arg(id) AND outcome = 'reserved';

-- name: RunningTotalUSD :one
SELECT (coalesce(sum(coalesce(settled_cost_usd, reserved_cost_usd)), 0))::text AS total_usd
FROM budget.model_calls;

-- name: InsertReconciliation :many
-- Inserts the difference when the provider's usage is higher than the local
-- total; nothing otherwise. Only the server calls it, once at start.
INSERT INTO budget.model_calls (purpose, reserved_cost_usd, settled_cost_usd, input_tokens, output_tokens, outcome)
SELECT 'reconciliation', d.diff, d.diff, 0, 0, 'settled'
FROM (
    -- Rounded to the column's scale first, so a provider figure with more
    -- than 8 decimals never inserts a row worth 0.00000000.
    SELECT round(CAST(sqlc.arg(provider_usage_usd)::text AS numeric) - coalesce(sum(coalesce(settled_cost_usd, reserved_cost_usd)), 0), 8) AS diff
    FROM budget.model_calls
) d
WHERE d.diff > 0
RETURNING settled_cost_usd::text AS added_usd;
