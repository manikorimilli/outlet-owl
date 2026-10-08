package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

// ReviewService lists reviews. *reviews.Service satisfies it.
type ReviewService interface {
	List(ctx context.Context, user auth.User, f reviews.Filter) (reviews.Page, error)
}

// tagsJSON and reviewSummary are Tags and ReviewSummary in api/openapi.yaml.

type tagsJSON struct {
	Themes        []string `json:"themes"`
	Sentiment     string   `json:"sentiment"`
	IsUrgent      bool     `json:"is_urgent"`
	UrgentReasons []string `json:"urgent_reasons"`
	PromptVersion int      `json:"prompt_version"`
}

type reviewSummary struct {
	ID           int64     `json:"id"`
	Outlet       outletRef `json:"outlet"`
	Source       string    `json:"source"`
	ReviewDate   string    `json:"review_date"`
	Rating       int       `json:"rating"`
	ReviewText   string    `json:"review_text"`
	ReviewerName string    `json:"reviewer_name"`
	Tags         *tagsJSON `json:"tags"`
	ReplyStatus  string    `json:"reply_status"`
}

func newReviewSummary(r reviews.Review) reviewSummary {
	out := reviewSummary{ID: r.ID, Outlet: outletRef{ID: r.OutletID, Name: r.OutletName}, Source: r.Source,
		ReviewDate: r.ReviewDate.Format(time.DateOnly), Rating: r.Rating, ReviewText: r.Text,
		ReviewerName: r.ReviewerName, ReplyStatus: r.ReplyStatus}
	if t := r.Tags; t != nil {
		out.Tags = &tagsJSON{Themes: nonNil(t.Themes), Sentiment: t.Sentiment, IsUrgent: t.IsUrgent,
			UrgentReasons: nonNil(t.UrgentReasons), PromptVersion: t.PromptVersion}
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

var reviewParams = map[string]bool{
	"cursor": true, "limit": true, "q": true, "filter[outlet_id]": true, "filter[theme]": true,
	"filter[sentiment]": true, "filter[is_urgent]": true, "filter[reply_status]": true,
	"filter[review_date][gte]": true, "filter[review_date][lte]": true,
}

// parseReviewFilter is the one place GET /reviews query strings are read; a
// parameter not in the spec or a value that does not parse is 400.
func parseReviewFilter(q url.Values) (reviews.Filter, error) {
	var f reviews.Filter
	bad := func(reason string) (reviews.Filter, error) { return reviews.Filter{}, &malformedError{reason: reason} }
	for k, v := range q {
		if !reviewParams[k] {
			return bad("the parameter " + k + " is not in the spec")
		}
		if len(v) != 1 {
			return bad("the parameter " + k + " is given more than once")
		}
	}
	f.Q = q.Get("q")
	if q.Has("q") && f.Q == "" {
		return bad("q is empty")
	}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > reviews.MaxLimit {
			return bad("limit must be a whole number from 1 to 100")
		}
		f.Limit = n
	}
	if s := q.Get("cursor"); s != "" {
		c, err := reviews.DecodeCursor(s)
		if err != nil || len(s) > 200 {
			return bad("the cursor is not valid")
		}
		f.After = &c
	}
	if s := q.Get("filter[outlet_id]"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < 1 {
			return bad("filter[outlet_id] must be an outlet id")
		}
		f.OutletID = &n
	}
	f.Theme = q.Get("filter[theme]")
	switch f.Sentiment = q.Get("filter[sentiment]"); f.Sentiment {
	case "", "positive", "neutral", "negative":
	default:
		return bad("filter[sentiment] must be positive, neutral or negative")
	}
	if s := q.Get("filter[is_urgent]"); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil || (s != "true" && s != "false") {
			return bad("filter[is_urgent] must be true or false")
		}
		f.Urgent = &b
	}
	switch f.ReplyStatus = q.Get("filter[reply_status]"); f.ReplyStatus {
	case "", "none", "draft", "replied":
	default:
		return bad("filter[reply_status] must be none, draft or replied")
	}
	for key, dst := range map[string]**time.Time{"filter[review_date][gte]": &f.From, "filter[review_date][lte]": &f.To} {
		if s := q.Get(key); s != "" {
			d, err := time.Parse(time.DateOnly, s)
			if err != nil {
				return bad(key + " must be a date, YYYY-MM-DD")
			}
			*dst = &d
		}
	}
	return f, nil
}

// listReviews answers GET /api/v1/reviews (operationId listReviews).
func listReviews(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		f, err := parseReviewFilter(r.URL.Query())
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		page, err := d.Reviews.List(r.Context(), user, f)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		data := make([]reviewSummary, len(page.Reviews))
		for i, rv := range page.Reviews {
			data[i] = newReviewSummary(rv)
		}
		var next *string
		if page.Next != nil {
			s := page.Next.Encode()
			next = &s
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"data":  data,
			"page":  map[string]any{"next_cursor": next, "has_more": next != nil},
			"total": page.Total,
		})
	})
}

// themeJSON is Theme in api/openapi.yaml.
type themeJSON struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

func themesJSON() []themeJSON {
	out := make([]themeJSON, len(tagging.Themes))
	for i, t := range tagging.Themes {
		out[i] = themeJSON{Code: t.Code, Label: t.Label}
	}
	return out
}

// listThemes answers GET /api/v1/themes (operationId listThemes).
func listThemes(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, _ *http.Request, _ auth.User) {
		WriteJSON(w, http.StatusOK, map[string]any{"data": themesJSON()})
	})
}
