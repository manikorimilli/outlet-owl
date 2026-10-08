package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
)

// The dashboard reads its grouped counts through the store.
var _ dashboard.Store = (*Store)(nil)

func day(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

// WeeklyOutletStats returns per outlet and week the review counts, the mean
// rating and the sentiment counts between from and to.
func (s *Store) WeeklyOutletStats(ctx context.Context, scope auth.Scope, from, to time.Time) ([]dashboard.WeekRow, error) {
	rows, err := New(s.Pool).WeeklyOutletStats(ctx, WeeklyOutletStatsParams{
		AllOutlets: scope.All, ScopeOutletID: scope.OutletID, DateFrom: day(from), DateTo: day(to)})
	if err != nil {
		return nil, fmt.Errorf("query weekly outlet stats: %w", err)
	}
	out := make([]dashboard.WeekRow, len(rows))
	for i, r := range rows {
		out[i] = dashboard.WeekRow{OutletID: r.OutletID, WeekStart: r.WeekStart.Time, ReviewCount: int(r.ReviewCount),
			AvgRating: r.AverageRating, Positive: int(r.Positive), Neutral: int(r.Neutral), Negative: int(r.Negative),
			Untagged: int(r.Untagged)}
	}
	return out, nil
}

// NegativeThemeCounts returns negative reviews per outlet, week and theme.
func (s *Store) NegativeThemeCounts(ctx context.Context, scope auth.Scope, from, to time.Time) ([]dashboard.ThemeRow, error) {
	rows, err := New(s.Pool).NegativeThemeCounts(ctx, NegativeThemeCountsParams{
		AllOutlets: scope.All, ScopeOutletID: scope.OutletID, DateFrom: day(from), DateTo: day(to)})
	if err != nil {
		return nil, fmt.Errorf("query negative theme counts: %w", err)
	}
	out := make([]dashboard.ThemeRow, len(rows))
	for i, r := range rows {
		out[i] = dashboard.ThemeRow{OutletID: r.OutletID, WeekStart: r.WeekStart.Time, Theme: r.Theme, Negative: int(r.Negative)}
	}
	return out, nil
}

// ReviewCounts returns the reviews dated between from and to, and how many
// of them are untagged.
func (s *Store) ReviewCounts(ctx context.Context, scope auth.Scope, from, to time.Time) (int, int, error) {
	r, err := New(s.Pool).ReviewCounts(ctx, ReviewCountsParams{
		AllOutlets: scope.All, ScopeOutletID: scope.OutletID, DateFrom: day(from), DateTo: day(to)})
	if err != nil {
		return 0, 0, fmt.Errorf("query review counts: %w", err)
	}
	return int(r.Total), int(r.Untagged), nil
}

// UntaggedCount returns the untagged reviews in the scope.
func (s *Store) UntaggedCount(ctx context.Context, scope auth.Scope) (int, error) {
	n, err := New(s.Pool).UntaggedCount(ctx, UntaggedCountParams{AllOutlets: scope.All, ScopeOutletID: scope.OutletID})
	if err != nil {
		return 0, fmt.Errorf("query untagged count: %w", err)
	}
	return int(n), nil
}

// BudgetCents returns the running model spend in cents, rounded up.
func (s *Store) BudgetCents(ctx context.Context) (int64, error) {
	n, err := New(s.Pool).BudgetCents(ctx)
	if err != nil {
		return 0, fmt.Errorf("query budget cents: %w", err)
	}
	return n, nil
}
