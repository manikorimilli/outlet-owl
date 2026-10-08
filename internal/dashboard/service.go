package dashboard

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

// Windows from the API: trends cover 12 complete weeks, the heatmap 4.
const (
	TrendWeeks   = 12
	HeatmapWeeks = 4
)

// WeekRow is one outlet's counts for one week.
type WeekRow struct {
	OutletID    int64
	WeekStart   time.Time
	ReviewCount int
	AvgRating   float64
	Positive    int
	Neutral     int
	Negative    int
	Untagged    int
	Replied     int
}

// ThemeRow is the number of negative reviews of one outlet, week and theme.
type ThemeRow struct {
	OutletID  int64
	WeekStart time.Time
	Theme     string
	Negative  int
}

// Store reads the grouped counts, every query within the scope.
type Store interface {
	ListOutlets(ctx context.Context, scope auth.Scope) ([]outlets.Outlet, error)
	WeeklyOutletStats(ctx context.Context, scope auth.Scope, from, to time.Time) ([]WeekRow, error)
	NegativeThemeCounts(ctx context.Context, scope auth.Scope, from, to time.Time) ([]ThemeRow, error)
	ReviewCounts(ctx context.Context, scope auth.Scope, from, to time.Time) (total, untagged int, err error)
}

// Service builds the reports.
type Service struct {
	store Store
	now   func() time.Time
	loc   *time.Location
}

// NewService builds the dashboard service; now is the clock.
func NewService(store Store, now func() time.Time, loc *time.Location) *Service {
	return &Service{store: store, now: now, loc: loc}
}

// TrendWeek is one week of one outlet (TrendWeek in the API).
type TrendWeek struct {
	WeekStart   time.Time
	ReviewCount int
	AvgRating   *float64
	Positive    int
	Neutral     int
	Negative    int
	Untagged    int
	Replied     int
}

// OutletTrend is one outlet's 12 weeks.
type OutletTrend struct {
	Outlet outlets.Outlet
	Weeks  []TrendWeek
}

// Trends is the trends report.
type Trends struct {
	Weeks   []time.Time
	Outlets []OutletTrend
}

// Trends returns the 12 complete weeks per outlet in scope (AC-US-01-005-1, -2).
func (s *Service) Trends(ctx context.Context, user auth.User) (Trends, error) {
	weeks := lastWeeks(LatestCompleteWeek(s.now(), s.loc), TrendWeeks)
	list, err := s.store.ListOutlets(ctx, user.Scope())
	if err != nil {
		return Trends{}, fmt.Errorf("trends: %w", err)
	}
	rows, err := s.store.WeeklyOutletStats(ctx, user.Scope(), weeks[0].Start, weeks[len(weeks)-1].End)
	if err != nil {
		return Trends{}, fmt.Errorf("trends: %w", err)
	}
	type key struct {
		outlet int64
		week   time.Time
	}
	byKey := make(map[key]WeekRow, len(rows))
	for _, r := range rows {
		byKey[key{r.OutletID, r.WeekStart}] = r
	}
	t := Trends{Weeks: make([]time.Time, len(weeks))}
	for i, w := range weeks {
		t.Weeks[i] = w.Start
	}
	for _, o := range list {
		ot := OutletTrend{Outlet: o, Weeks: make([]TrendWeek, len(weeks))}
		for i, w := range weeks {
			tw := TrendWeek{WeekStart: w.Start}
			if r, ok := byKey[key{o.ID, w.Start}]; ok {
				avg := r.AvgRating
				tw = TrendWeek{WeekStart: w.Start, ReviewCount: r.ReviewCount, AvgRating: &avg,
					Positive: r.Positive, Neutral: r.Neutral, Negative: r.Negative, Untagged: r.Untagged, Replied: r.Replied}
			}
			ot.Weeks[i] = tw
		}
		t.Outlets = append(t.Outlets, ot)
	}
	return t, nil
}

// HeatRow is one outlet's negative reviews per theme.
type HeatRow struct {
	Outlet outlets.Outlet
	Counts map[string]int
	Total  int
}

// Heatmap is the theme heatmap over the last 4 complete weeks.
type Heatmap struct {
	Period   Week
	Themes   []tagging.Theme
	Rows     []HeatRow
	Untagged int
}

// Heatmap counts negative reviews per outlet and theme; a review with two
// themes counts once under each (AC-US-01-006-1, -2, Q-021).
func (s *Service) Heatmap(ctx context.Context, user auth.User) (Heatmap, error) {
	weeks := lastWeeks(LatestCompleteWeek(s.now(), s.loc), HeatmapWeeks)
	period := Week{Start: weeks[0].Start, End: weeks[len(weeks)-1].End}
	list, err := s.store.ListOutlets(ctx, user.Scope())
	if err != nil {
		return Heatmap{}, fmt.Errorf("heatmap: %w", err)
	}
	rows, err := s.store.NegativeThemeCounts(ctx, user.Scope(), period.Start, period.End)
	if err != nil {
		return Heatmap{}, fmt.Errorf("heatmap: %w", err)
	}
	_, untagged, err := s.store.ReviewCounts(ctx, user.Scope(), period.Start, period.End)
	if err != nil {
		return Heatmap{}, fmt.Errorf("heatmap: %w", err)
	}
	h := Heatmap{Period: period, Themes: tagging.Themes, Untagged: untagged}
	idx := map[int64]int{}
	for i, o := range list {
		counts := map[string]int{}
		for _, th := range tagging.Themes {
			counts[th.Code] = 0
		}
		h.Rows = append(h.Rows, HeatRow{Outlet: o, Counts: counts})
		idx[o.ID] = i
	}
	for _, r := range rows {
		i, ok := idx[r.OutletID]
		if !ok {
			continue // an outlet outside the list
		}
		if _, known := h.Rows[i].Counts[r.Theme]; !known {
			continue // a theme no longer in the list
		}
		h.Rows[i].Counts[r.Theme] += r.Negative
		h.Rows[i].Total += r.Negative
	}
	return h, nil
}

// Mover is one outlet and theme pair that changed.
type Mover struct {
	Outlet   outlets.Outlet
	Theme    tagging.Theme
	Previous int
	Current  int
	Change   int
}

// Movers is the movers report.
type Movers struct {
	Week        Week
	Previous    Week
	ReviewCount int
	Untagged    int
	Movers      []Mover
}

// Movers ranks outlet and theme pairs by the size of the change in negative
// reviews between the latest complete week and the one before (AC-US-01-007-1).
// Ties put a rise before a fall, then outlet name, then theme list order;
// pairs with no change are left out.
func (s *Service) Movers(ctx context.Context, user auth.User) (Movers, error) {
	week := LatestCompleteWeek(s.now(), s.loc)
	prev := weekFrom(week.Start.AddDate(0, 0, -7))
	m := Movers{Week: week, Previous: prev, Movers: []Mover{}}
	list, err := s.store.ListOutlets(ctx, user.Scope())
	if err != nil {
		return Movers{}, fmt.Errorf("movers: %w", err)
	}
	rows, err := s.store.NegativeThemeCounts(ctx, user.Scope(), prev.Start, week.End)
	if err != nil {
		return Movers{}, fmt.Errorf("movers: %w", err)
	}
	if m.ReviewCount, _, err = s.store.ReviewCounts(ctx, user.Scope(), week.Start, week.End); err != nil {
		return Movers{}, fmt.Errorf("movers: %w", err)
	}
	if _, m.Untagged, err = s.store.ReviewCounts(ctx, user.Scope(), prev.Start, week.End); err != nil {
		return Movers{}, fmt.Errorf("movers: %w", err)
	}
	type key struct {
		outlet int64
		theme  string
	}
	cur, before := map[key]int{}, map[key]int{}
	for _, r := range rows {
		if r.WeekStart.Equal(week.Start) {
			cur[key{r.OutletID, r.Theme}] += r.Negative
		} else {
			before[key{r.OutletID, r.Theme}] += r.Negative
		}
	}
	order := map[string]int{}
	for i, th := range tagging.Themes {
		order[th.Code] = i
	}
	for _, o := range list {
		for _, th := range tagging.Themes {
			k := key{o.ID, th.Code}
			if c := cur[k] - before[k]; c != 0 {
				m.Movers = append(m.Movers, Mover{Outlet: o, Theme: th, Previous: before[k], Current: cur[k], Change: c})
			}
		}
	}
	slices.SortFunc(m.Movers, func(a, b Mover) int {
		return cmp.Or(
			cmp.Compare(abs(b.Change), abs(a.Change)),
			cmp.Compare(b.Change, a.Change), // a rise before a fall of the same size
			cmp.Compare(strings.ToLower(a.Outlet.Name), strings.ToLower(b.Outlet.Name)),
			cmp.Compare(order[a.Theme.Code], order[b.Theme.Code]),
		)
	})
	return m, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
