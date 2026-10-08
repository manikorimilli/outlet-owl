package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
)

var ist = time.FixedZone("IST", 5*3600+1800)

func d(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }

func TestWeeks_LatestCompleteWeek(t *testing.T) {
	cases := []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, 10, 5, 9, 0, 0, 0, ist), "2026-09-28"},       // Monday: last week just ended
		{time.Date(2026, 10, 4, 23, 0, 0, 0, ist), "2026-09-21"},      // Sunday: this week is not complete
		{time.Date(2026, 10, 8, 12, 0, 0, 0, ist), "2026-09-28"},      // mid-week
		{time.Date(2026, 10, 4, 19, 0, 0, 0, time.UTC), "2026-09-28"}, // Sunday in UTC, already Monday in India
	}
	for _, tc := range cases {
		w := LatestCompleteWeek(tc.now, ist)
		if w.Start != d(tc.want) || w.End != d(tc.want).AddDate(0, 0, 6) {
			t.Errorf("now %v: week %v to %v, want start %s", tc.now, w.Start, w.End, tc.want)
		}
	}
}

// fakeStore answers from fixed rows, applying the scope as the SQL does.
type fakeStore struct {
	outlets []outlets.Outlet
	weeks   []WeekRow
	themes  []ThemeRow
	total   int
	untag   int
}

func (f *fakeStore) ListOutlets(_ context.Context, s auth.Scope) ([]outlets.Outlet, error) {
	var out []outlets.Outlet
	for _, o := range f.outlets {
		if s.All || o.ID == s.OutletID {
			out = append(out, o)
		}
	}
	return out, nil
}
func (f *fakeStore) WeeklyOutletStats(context.Context, auth.Scope, time.Time, time.Time) ([]WeekRow, error) {
	return f.weeks, nil
}
func (f *fakeStore) NegativeThemeCounts(_ context.Context, s auth.Scope, from, to time.Time) ([]ThemeRow, error) {
	var out []ThemeRow
	for _, r := range f.themes {
		if !r.WeekStart.Before(from) && !r.WeekStart.After(to) && (s.All || r.OutletID == s.OutletID) {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeStore) ReviewCounts(context.Context, auth.Scope, time.Time, time.Time) (int, int, error) {
	return f.total, f.untag, nil
}

var (
	now   = func() time.Time { return time.Date(2026, 10, 8, 12, 0, 0, 0, ist) } // latest week 28 Sep
	admin = auth.User{Role: auth.RoleBrandAdmin}
	kora  = outlets.Outlet{ID: 1, Name: "Koramangala"}
	indi  = outlets.Outlet{ID: 2, Name: "Indiranagar"}
)

// AC-US-01-007-1, -2, -3: ranked by the size of the change, a planted spike
// first, both weeks' counts shown, zero changes left out, ties ordered.
func TestMovers_RankedByAbsoluteChange(t *testing.T) {
	cur, prev := d("2026-09-28"), d("2026-09-21")
	f := &fakeStore{outlets: []outlets.Outlet{kora, indi}, themes: []ThemeRow{
		{1, prev, "wait_time", 3}, {1, cur, "wait_time", 11}, // the spike: +8
		{2, prev, "staff", 4}, {2, cur, "staff", 2}, // -2
		{1, cur, "price", 2},                      // +2, ties with -2: the rise first
		{2, prev, "food", 1}, {2, cur, "food", 1}, // no change: left out
		{1, d("2026-09-14"), "food", 9}, // outside the two weeks
	}, total: 66}
	m, err := NewService(f, now, ist).Movers(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Movers) != 3 {
		t.Fatalf("movers = %+v, want 3", m.Movers)
	}
	top := m.Movers[0]
	if top.Outlet.ID != 1 || top.Theme.Code != "wait_time" || top.Previous != 3 || top.Current != 11 || top.Change != 8 {
		t.Fatalf("top = %+v, want Koramangala wait time 3 to 11", top)
	}
	if m.Movers[1].Change != 2 || m.Movers[2].Change != -2 {
		t.Fatalf("ties = %+v, want the rise before the fall", m.Movers[1:])
	}
	if m.Week.Start != cur || m.Previous.Start != prev || m.ReviewCount != 66 {
		t.Fatalf("weeks %v %v count %d", m.Week, m.Previous, m.ReviewCount)
	}
}

// AC-US-01-006-1, -2: one row per outlet, every theme present, the row total.
func TestHeatmap_CountsPerThemeAndRow(t *testing.T) {
	w := d("2026-09-21")
	f := &fakeStore{outlets: []outlets.Outlet{kora, indi}, themes: []ThemeRow{
		{1, w, "food", 2}, {1, w, "staff", 2}, // one review with two themes counts once under each
		{1, w, "ambience", 5}, // a theme no longer in the list
	}, untag: 4}
	h, err := NewService(f, now, ist).Heatmap(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Rows) != 2 || len(h.Rows[1].Counts) != 5 || h.Rows[0].Counts["food"] != 2 || h.Rows[0].Total != 4 || h.Untagged != 4 {
		t.Fatalf("heatmap = %+v", h)
	}
	if h.Period.Start != d("2026-09-07") || h.Period.End != d("2026-10-04") {
		t.Fatalf("period = %+v, want 7 Sep to 4 Oct", h.Period)
	}
}

// AC-US-01-005-1, -2 and AC-US-00-001-2: 12 weeks per outlet, zeros where a
// week is empty; a manager gets their own outlet only.
func TestTrends_FillsTwelveWeeksInScope(t *testing.T) {
	f := &fakeStore{outlets: []outlets.Outlet{kora, indi}, weeks: []WeekRow{
		{OutletID: 1, WeekStart: d("2026-09-28"), ReviewCount: 4, AvgRating: 3.5, Negative: 2, Positive: 2},
	}}
	tr, err := NewService(f, now, ist).Trends(context.Background(), auth.User{Role: auth.RoleOutletManager, Outlet: &auth.OutletRef{ID: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Weeks) != 12 || tr.Weeks[0] != d("2026-07-13") || tr.Weeks[11] != d("2026-09-28") || len(tr.Outlets) != 1 {
		t.Fatalf("trends weeks %v outlets %d", tr.Weeks, len(tr.Outlets))
	}
	ws := tr.Outlets[0].Weeks
	if ws[0].AvgRating != nil || ws[0].ReviewCount != 0 || *ws[11].AvgRating != 3.5 || ws[11].Negative != 2 {
		t.Fatalf("weeks = %+v", ws)
	}
}
