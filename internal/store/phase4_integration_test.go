//go:build integration

package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/imports"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

// seedReviews stores two outlets' reviews and tags some of them.
func seedReviews(t *testing.T) (*Store, context.Context, int64, int64) {
	t.Helper()
	db := storetest.New(t)
	s := &Store{Pool: db.Pool}
	ctx := context.Background()
	a, _, _ := s.CreateOutlet(ctx, "Koramangala")
	b, _, _ := s.CreateOutlet(ctx, "Indiranagar")
	day := func(s string) time.Time { d, _ := time.Parse(time.DateOnly, s); return d }
	rows := []imports.NewReview{
		{OutletID: a.ID, Row: connector.Row{Source: "Google", Date: day("2026-09-29"), Rating: 1, Text: "Waited an hour", ReviewerName: "Asha"}},
		{OutletID: a.ID, Row: connector.Row{Source: "Google", Date: day("2026-09-30"), Rating: 2, Text: "50% off was a lie", ReviewerName: "Ravi"}},
		{OutletID: b.ID, Row: connector.Row{Source: "Zomato", Date: day("2026-09-22"), Rating: 5, Text: "lovely", ReviewerName: "Wait Singh"}},
		{OutletID: b.ID, Row: connector.Row{Source: "Zomato", Date: day("2026-10-01"), Rating: 4, Text: "खाना अच्छा था", ReviewerName: "Priya"}},
	}
	if _, _, err := s.SaveImport(ctx, imports.Write{RequestID: "6192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", FileName: "f.csv", Reviews: rows}); err != nil {
		t.Fatal(err)
	}
	ids, _ := s.UntaggedReviewIDs(ctx, 0)
	if _, err := s.SaveResults(ctx, []tagging.Result{
		{ReviewID: ids[0], Themes: []string{"wait_time", "staff"}, Sentiment: "negative", UrgentReasons: []string{}},
		{ReviewID: ids[1], Themes: []string{"price"}, Sentiment: "negative", UrgentReasons: []string{"legal_threat"}},
		{ReviewID: ids[2], Themes: []string{}, Sentiment: "positive", UrgentReasons: []string{}},
	}, 1); err != nil {
		t.Fatal(err)
	}
	return s, ctx, a.ID, b.ID
}

func texts(rs []reviews.Review) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Text)
	}
	return out
}

// AC-US-01-008-1, -2: search over text and reviewer name; filters on the
// tags; % in a search is literal; untagged reviews match no tag filter.
func TestListReviews_SearchAndFilters(t *testing.T) {
	s, ctx, _, _ := seedReviews(t)
	all := auth.Scope{All: true}
	urgent := true
	cases := []struct {
		f    reviews.Filter
		want string
	}{
		{reviews.Filter{Q: "wait"}, "[Waited an hour lovely]"},
		{reviews.Filter{Q: "50%"}, "[50% off was a lie]"},
		{reviews.Filter{Q: "0_o"}, "[]"},
		{reviews.Filter{Theme: "staff"}, "[Waited an hour]"},
		{reviews.Filter{Sentiment: "negative"}, "[50% off was a lie Waited an hour]"},
		{reviews.Filter{Urgent: &urgent}, "[50% off was a lie]"},
	}
	for _, tc := range cases {
		got, err := s.ListReviews(ctx, all, tc.f, 10)
		n, _ := s.CountReviews(ctx, all, tc.f)
		if err != nil || fmt.Sprint(texts(got)) != tc.want || n != len(got) {
			t.Errorf("%+v: %v (count %d), %v; want %s", tc.f, texts(got), n, err, tc.want)
		}
	}
}

// AC-US-00-001-3: a manager's scope sees one outlet whatever else is asked.
func TestListReviews_ManagerSeesOwnOutletOnly(t *testing.T) {
	s, ctx, _, b := seedReviews(t)
	got, _ := s.ListReviews(ctx, auth.Scope{OutletID: b}, reviews.Filter{Q: "a"}, 10)
	for _, r := range got {
		if r.OutletID != b {
			t.Fatalf("manager scope returned outlet %d", r.OutletID)
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want Indiranagar's 2", texts(got))
	}
}

func TestListReviews_KeysetPages(t *testing.T) {
	s, ctx, _, _ := seedReviews(t)
	first, _ := s.ListReviews(ctx, auth.Scope{All: true}, reviews.Filter{}, 2)
	last := first[1]
	rest, _ := s.ListReviews(ctx, auth.Scope{All: true}, reviews.Filter{After: &reviews.Cursor{Date: last.ReviewDate, ID: last.ID}}, 10)
	if fmt.Sprint(texts(first)) != "[खाना अच्छा था 50% off was a lie]" || fmt.Sprint(texts(rest)) != "[Waited an hour lovely]" {
		t.Fatalf("pages %v then %v", texts(first), texts(rest))
	}
}

// AC-US-00-001-2: every report query is scoped; weeks start on Monday.
func TestDashboardQueries_ScopeAndWeeks(t *testing.T) {
	s, ctx, a, _ := seedReviews(t)
	from, _ := time.Parse(time.DateOnly, "2026-09-21")
	to, _ := time.Parse(time.DateOnly, "2026-10-04")
	weeks, err := s.WeeklyOutletStats(ctx, auth.Scope{OutletID: a}, from, to)
	if err != nil || len(weeks) != 1 || weeks[0].WeekStart.Format(time.DateOnly) != "2026-09-28" || weeks[0].ReviewCount != 2 ||
		weeks[0].AvgRating != 1.5 || weeks[0].Negative != 2 {
		t.Fatalf("weeks = %+v, %v", weeks, err)
	}
	themes, _ := s.NegativeThemeCounts(ctx, auth.Scope{All: true}, from, to)
	if len(themes) != 3 { // wait_time, staff and price, once each
		t.Fatalf("themes = %+v", themes)
	}
	total, untagged, _ := s.ReviewCounts(ctx, auth.Scope{All: true}, from, to)
	if n, _ := s.UntaggedCount(ctx, auth.Scope{All: true}); total != 4 || untagged != 1 || n != 1 {
		t.Fatalf("total %d untagged %d count %d", total, untagged, n)
	}
}

func TestBudgetCents(t *testing.T) {
	s, ctx, _, _ := seedReviews(t)
	spend(ctx, t, s, "8.001")
	if c, err := s.BudgetCents(ctx); err != nil || c != 801 {
		t.Fatalf("cents = %d, %v; want 801", c, err)
	}
}
