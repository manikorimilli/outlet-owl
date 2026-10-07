//go:build integration

package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

const testModel = "anthropic/claude-haiku-4.5"

func newBudgetStore(t *testing.T) (*Store, *storetest.DB, context.Context) {
	t.Helper()
	db := storetest.New(t)
	return &Store{Pool: db.Pool}, db, context.Background()
}

// spend puts a settled call of the given cost on the log, as earlier calls
// would have.
func spend(ctx context.Context, t *testing.T, s *Store, usd string) {
	t.Helper()
	if _, err := s.Pool.Exec(ctx, `
		INSERT INTO budget.model_calls (purpose, model, prompt_version, input_tokens, output_tokens,
			reserved_cost_usd, settled_cost_usd, outcome)
		VALUES ('tagging', $1, 1, 0, 0, $2::numeric, $2::numeric, 'settled')`, testModel, usd); err != nil {
		t.Fatalf("spend %s: %v", usd, err)
	}
}

func reserve(ctx context.Context, s *Store) (int64, error) {
	return s.ReserveModelCall(ctx, "tagging", testModel, 1, "0.01500000", "8")
}

func TestReserveModelCall_RefusesAboveEightDollars(t *testing.T) {
	s, _, ctx := newBudgetStore(t)
	spend(ctx, t, s, "8.01")

	_, err := reserve(ctx, s)

	if !errors.Is(err, gateway.ErrBudgetExhausted) {
		t.Fatalf("err = %v, want ErrBudgetExhausted", err)
	}
	if total, _ := s.RunningTotalUSD(ctx); total != "8.01000000" {
		t.Fatalf("total = %s, want 8.01000000: a refused call must add nothing", total)
	}
}

func TestReserveModelCall_AllowsAtExactlyEightAndRefusesTheNext(t *testing.T) {
	for _, start := range []string{"7.99", "8.00"} {
		t.Run(start, func(t *testing.T) {
			s, _, ctx := newBudgetStore(t)
			spend(ctx, t, s, start)

			if _, err := reserve(ctx, s); err != nil {
				t.Fatalf("at %s the call must be allowed (Q-015): %v", start, err)
			}
			if _, err := reserve(ctx, s); start == "8.00" && !errors.Is(err, gateway.ErrBudgetExhausted) {
				t.Fatalf("after the crossing call the next must be refused, got %v", err)
			}
		})
	}
}

func TestRunningTotal_SurvivesAReconnect(t *testing.T) {
	s, db, ctx := newBudgetStore(t)
	spend(ctx, t, s, "1.25")
	if _, err := reserve(ctx, s); err != nil {
		t.Fatal(err)
	}
	db.Pool.Close()

	pool, err := pgxpool.New(ctx, db.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	total, err := (&Store{Pool: pool}).RunningTotalUSD(ctx)

	if err != nil || total != "1.26500000" {
		t.Fatalf("total after reconnect = %q, %v; want 1.26500000 (settled plus the open reserve)", total, err)
	}
}

func TestSettleAndFail_OnlyChangeAReservedRow(t *testing.T) {
	s, _, ctx := newBudgetStore(t)
	id, err := reserve(ctx, s)
	if err != nil {
		t.Fatal(err)
	}

	if ok, err := s.SettleModelCall(ctx, id, 2600, 300, "0.00410000"); err != nil || !ok {
		t.Fatalf("settle = %v, %v; want a change", ok, err)
	}
	if ok, err := s.FailModelCall(ctx, id); err != nil || ok {
		t.Fatalf("fail after settle = %v, %v; want no change", ok, err)
	}
	if ok, err := s.SettleModelCall(ctx, id, 1, 1, "9.00000000"); err != nil || ok {
		t.Fatalf("second settle = %v, %v; want no change", ok, err)
	}
	if total, _ := s.RunningTotalUSD(ctx); total != "0.00410000" {
		t.Fatalf("total = %s, want the settled 0.00410000", total)
	}
}

func TestInsertReconciliation_OnlyWhenTheProviderIsHigher(t *testing.T) {
	s, _, ctx := newBudgetStore(t)
	spend(ctx, t, s, "1.00")

	if _, added, err := s.InsertReconciliation(ctx, "0.75"); err != nil || added {
		t.Fatalf("provider lower: added = %v, %v; want nothing", added, err)
	}
	got, added, err := s.InsertReconciliation(ctx, "1.40")
	if err != nil || !added || got != "0.40000000" {
		t.Fatalf("provider higher: %q, %v, %v; want 0.40000000 added", got, added, err)
	}
	if total, _ := s.RunningTotalUSD(ctx); total != "1.40000000" {
		t.Fatalf("total = %s, want 1.40000000", total)
	}
}

// GET /key reports usage as a JSON float with more than 8 decimals; a
// difference below the column's scale must not insert a zero row on every
// live or record start.
func TestInsertReconciliation_NeverInsertsAZeroRow(t *testing.T) {
	s, _, ctx := newBudgetStore(t)
	spend(ctx, t, s, "1.23456789")

	if got, added, err := s.InsertReconciliation(ctx, "1.234567891"); err != nil || added {
		t.Fatalf("a sub-scale difference added %q, %v, %v; want nothing", got, added, err)
	}
	var rows int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM budget.model_calls WHERE purpose = 'reconciliation'").Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("reconciliation rows = %d, %v; want 0", rows, err)
	}
}
