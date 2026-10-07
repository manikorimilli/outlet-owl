package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
)

// The gateway keeps its budget log in the store.
var _ gateway.Store = (*Store)(nil)

// ReserveModelCall reserves a call's worst-case price, as a decimal string in
// USD, while the running total is at most limitUSD. It returns
// gateway.ErrBudgetExhausted when the total is already above the limit.
func (s *Store) ReserveModelCall(ctx context.Context, purpose, model string, promptVersion int, reservedUSD, limitUSD string) (int64, error) {
	pv := int32(promptVersion)
	id, err := New(s.Pool).ReserveModelCall(ctx, ReserveModelCallParams{
		Purpose:         BudgetModelCallPurpose(purpose),
		Model:           model,
		PromptVersion:   &pv,
		ReservedCostUsd: reservedUSD,
		LimitUsd:        limitUSD,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, gateway.ErrBudgetExhausted
	}
	if err != nil {
		return 0, fmt.Errorf("reserve model call: %w", err)
	}
	return id, nil
}

// SettleModelCall records a reserved call's tokens and cost. It reports
// whether a reserved row changed; a second settle changes nothing.
func (s *Store) SettleModelCall(ctx context.Context, id int64, inputTokens, outputTokens int, costUSD string) (bool, error) {
	n, err := New(s.Pool).SettleModelCall(ctx, SettleModelCallParams{
		ID:             id,
		InputTokens:    int32(inputTokens),
		OutputTokens:   int32(outputTokens),
		SettledCostUsd: costUSD,
	})
	if err != nil {
		return false, fmt.Errorf("settle model call %d: %w", id, err)
	}
	return n == 1, nil
}

// FailModelCall marks a reserved call failed; it stays counted at its
// reserved price. It reports whether a reserved row changed.
func (s *Store) FailModelCall(ctx context.Context, id int64) (bool, error) {
	n, err := New(s.Pool).FailModelCall(ctx, id)
	if err != nil {
		return false, fmt.Errorf("fail model call %d: %w", id, err)
	}
	return n == 1, nil
}

// RunningTotalUSD is the sum of settled cost, or reserved cost where not yet
// settled, over every call, as a decimal string in USD.
func (s *Store) RunningTotalUSD(ctx context.Context) (string, error) {
	total, err := New(s.Pool).RunningTotalUSD(ctx)
	if err != nil {
		return "", fmt.Errorf("read the running total: %w", err)
	}
	return total, nil
}

// InsertReconciliation adds the difference when the provider's usage is
// higher than the local total, and returns it; added is false when nothing
// was inserted.
func (s *Store) InsertReconciliation(ctx context.Context, providerUsageUSD string) (addedUSD string, added bool, err error) {
	rows, err := New(s.Pool).InsertReconciliation(ctx, providerUsageUSD)
	if err != nil {
		return "", false, fmt.Errorf("insert reconciliation: %w", err)
	}
	if len(rows) == 0 {
		return "", false, nil
	}
	return rows[0], true, nil
}
