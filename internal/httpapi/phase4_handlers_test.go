package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
)

type fakeReviews struct{ got reviews.Filter }

func (f *fakeReviews) List(_ context.Context, _ auth.User, flt reviews.Filter) (reviews.Page, error) {
	f.got = flt
	day := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	return reviews.Page{Total: 2, Next: &reviews.Cursor{Date: day, ID: 5}, Reviews: []reviews.Review{
		{ID: 5, OutletID: 1, OutletName: "Koramangala", ReviewDate: day, Rating: 1, Text: "खाने में कीड़ा", ReplyStatus: "none",
			Tags: &reviews.Tags{Themes: []string{"food"}, Sentiment: "negative", IsUrgent: true, UrgentReasons: []string{"food_safety"}, PromptVersion: 1}},
	}}, nil
}

type fakeDashboard struct{}

func (fakeDashboard) Trends(context.Context, auth.User) (dashboard.Trends, error) {
	return dashboard.Trends{}, nil
}
func (fakeDashboard) Heatmap(context.Context, auth.User) (dashboard.Heatmap, error) {
	return dashboard.Heatmap{}, nil
}
func (fakeDashboard) Movers(context.Context, auth.User) (dashboard.Movers, error) {
	return dashboard.Movers{}, nil
}

type fakeStatus struct {
	cents  int64
	credit bool
}

func (f fakeStatus) UntaggedCount(context.Context, auth.Scope) (int, error) { return 5, nil }
func (f fakeStatus) BudgetCents(context.Context) (int64, error)             { return f.cents, nil }
func (f fakeStatus) WorkerState() string                                    { return "idle" }
func (f fakeStatus) CreditExhausted() bool                                  { return f.credit }

func phase4Handler(r ReviewService, s StatusReader) http.Handler {
	return New(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: fakeDB{}, Auth: newFakeAuth(),
		Reviews: r, Dashboard: fakeDashboard{}, Status: s, Brand: testBrand})
}

func getJSON(h http.Handler, path string) *httpResult {
	rec := do(h, request{method: http.MethodGet, path: path, cookie: adminCookie})
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return &httpResult{code: rec.Code, body: body}
}

type httpResult struct {
	code int
	body map[string]any
}

func TestListReviews_ParsesFiltersAndAnswersTheShape(t *testing.T) {
	f := &fakeReviews{}
	res := getJSON(phase4Handler(f, fakeStatus{}), "/api/v1/reviews?q=wait&filter[theme]=staff&filter[is_urgent]=true&filter[review_date][gte]=2026-09-28&limit=10")
	if res.code != 200 || f.got.Q != "wait" || f.got.Theme != "staff" || f.got.Urgent == nil || !*f.got.Urgent || f.got.Limit != 10 || f.got.From == nil {
		t.Fatalf("code %d filter %+v", res.code, f.got)
	}
	row := res.body["data"].([]any)[0].(map[string]any)
	tags := row["tags"].(map[string]any)
	if row["review_date"] != "2026-10-03" || row["review_text"] != "खाने में कीड़ा" || tags["urgent_reasons"].([]any)[0] != "food_safety" || row["reply_status"] != "none" {
		t.Fatalf("row = %v", row)
	}
	page := res.body["page"].(map[string]any)
	if page["has_more"] != true || page["next_cursor"] == nil || res.body["total"] != 2.0 {
		t.Fatalf("page = %v total %v", page, res.body["total"])
	}
}

func TestListReviews_BadQueriesAre400(t *testing.T) {
	for _, q := range []string{"?filter[colour]=red", "?limit=0", "?limit=101", "?filter[is_urgent]=yes", "?filter[sentiment]=angry",
		"?filter[review_date][gte]=03-10-2026", "?cursor=junk", "?q=", "?filter[outlet_id]=x", "?q=a&q=b"} {
		if res := getJSON(phase4Handler(&fakeReviews{}, fakeStatus{}), "/api/v1/reviews"+q); res.code != 400 {
			t.Errorf("%s: %d, want 400", q, res.code)
		}
	}
}

func TestTaggingStatus_States(t *testing.T) {
	cases := []struct {
		s     fakeStatus
		state string
	}{
		{fakeStatus{cents: 800}, "ok"},
		{fakeStatus{cents: 801}, "budget_exhausted"},
		{fakeStatus{cents: 12, credit: true}, "provider_credit_exhausted"},
	}
	for _, tc := range cases {
		res := getJSON(phase4Handler(&fakeReviews{}, tc.s), "/api/v1/tagging/status")
		budget := res.body["budget"].(map[string]any)
		if res.code != 200 || budget["state"] != tc.state || res.body["untagged_count"] != 5.0 || budget["limit_minor"] != 800.0 {
			t.Errorf("%+v: %d %v", tc.s, res.code, res.body)
		}
	}
}

func TestThemesAndDashboard_RoutesAnswerAndNeedSignIn(t *testing.T) {
	h := phase4Handler(&fakeReviews{}, fakeStatus{})
	if res := getJSON(h, "/api/v1/themes"); res.code != 200 || len(res.body["data"].([]any)) != 5 {
		t.Fatalf("themes = %d %v", res.code, res.body)
	}
	for _, p := range []string{"/api/v1/dashboard/trends", "/api/v1/dashboard/heatmap", "/api/v1/dashboard/movers", "/api/v1/tagging/status", "/api/v1/reviews", "/api/v1/themes"} {
		if rec := do(h, request{method: http.MethodGet, path: p}); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without a cookie: %d", p, rec.Code)
		}
		if res := getJSON(h, p); res.code != 200 {
			t.Errorf("%s: %d", p, res.code)
		}
	}
}
