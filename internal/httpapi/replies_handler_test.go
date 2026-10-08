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
	"github.com/manikorimilli/outlet-owl/internal/replies"
)

type fakeReplies struct {
	err      error
	drafting bool
	calls    int
}

func (f *fakeReplies) Get(context.Context, auth.User, int64) (replies.Detail, error) {
	return replies.Detail{}, f.err
}
func (f *fakeReplies) Draft(context.Context, auth.User, int64) (*replies.Reply, bool, error) {
	f.calls++
	return &replies.Reply{Status: "drafting", UpdatedAt: time.Now()}, f.drafting, f.err
}
func (f *fakeReplies) Save(context.Context, auth.User, int64, string, *time.Time) (*replies.Reply, error) {
	f.calls++
	return &replies.Reply{Status: "draft"}, f.err
}
func (f *fakeReplies) MarkReplied(context.Context, auth.User, int64, string, *time.Time) (*replies.Reply, error) {
	f.calls++
	return &replies.Reply{Status: "replied"}, f.err
}

func repliesHandler(f *fakeReplies) http.Handler {
	return New(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: fakeDB{}, Auth: newFakeAuth(), Replies: f, Brand: testBrand})
}

func code(rec interface{ Bytes() []byte }) string {
	var env fullEnvelope
	_ = json.Unmarshal(rec.Bytes(), &env)
	return env.Error.Code
}

func TestReplyRoutes_StatusesAndCodes(t *testing.T) {
	body := `{"reply_text":"Hi","based_on_updated_at":null}`
	cases := []struct {
		name   string
		f      *fakeReplies
		rq     request
		status int
		code   string
	}{
		{"DraftingIs202", &fakeReplies{drafting: true}, request{method: "POST", path: "/api/v1/reviews/7/draft", cookie: managerCookie}, 202, ""},
		{"BudgetIs503", &fakeReplies{err: replies.ErrBudgetExhausted}, request{method: "POST", path: "/api/v1/reviews/7/draft", cookie: managerCookie}, 503, "budget_exhausted"},
		{"ModelIs503", &fakeReplies{err: replies.ErrModelUnavailable}, request{method: "POST", path: "/api/v1/reviews/7/draft", cookie: managerCookie}, 503, "model_unavailable"},
		{"AdminDraftIs403", &fakeReplies{err: replies.ErrRoleNotAllowed}, request{method: "POST", path: "/api/v1/reviews/7/draft", cookie: adminCookie}, 403, "role_not_allowed"},
		{"AdminSaveRefusedBeforeBody", &fakeReplies{}, request{method: "PUT", path: "/api/v1/reviews/7/reply", cookie: adminCookie, contentType: "text/plain", body: "x"}, 403, "role_not_allowed"},
		{"ConflictIs409", &fakeReplies{err: &replies.ConflictError{Code: "reply_changed"}}, request{method: "POST", path: "/api/v1/reviews/7/replied", cookie: managerCookie, contentType: "application/json", body: body}, 409, "reply_changed"},
		{"BlankIs422", &fakeReplies{err: &replies.ValidationError{Reason: "blank"}}, request{method: "PUT", path: "/api/v1/reviews/7/reply", cookie: managerCookie, contentType: "application/json", body: body}, 422, "validation_failed"},
		{"NotFoundIs404", &fakeReplies{err: replies.ErrNotFound}, request{method: "GET", path: "/api/v1/reviews/7", cookie: managerCookie}, 404, "not_found"},
		{"BadIDIs400", &fakeReplies{}, request{method: "GET", path: "/api/v1/reviews/abc", cookie: managerCookie}, 400, "malformed_request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(repliesHandler(tc.f), tc.rq)
			if rec.Code != tc.status || code(rec.Body) != tc.code {
				t.Fatalf("got %d %q, want %d %q (%s)", rec.Code, code(rec.Body), tc.status, tc.code, rec.Body)
			}
			if tc.status == 202 && rec.Header().Get("Retry-After") != "2" {
				t.Fatal("202 without Retry-After")
			}
			if tc.name == "AdminSaveRefusedBeforeBody" && tc.f.calls != 0 {
				t.Fatal("the service was reached")
			}
		})
	}
}
