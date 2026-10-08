package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
)

// The review service reads the list through the store.
var _ reviews.Store = (*Store)(nil)

func optDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ListReviews returns at most limit reviews in the scope matching f, newest
// first, after f.After when it is set.
func (s *Store) ListReviews(ctx context.Context, scope auth.Scope, f reviews.Filter, limit int) ([]reviews.Review, error) {
	p := ListReviewsParams{
		AllOutlets: scope.All, ScopeOutletID: scope.OutletID, OutletID: f.OutletID,
		Theme: optString(f.Theme), Sentiment: optString(f.Sentiment), IsUrgent: f.Urgent,
		DateFrom: optDate(f.From), DateTo: optDate(f.To), RowLimit: int32(limit), ReplyStatus: optString(f.ReplyStatus),
	}
	if f.Q != "" {
		q := reviews.LikePattern(f.Q)
		p.Q = &q
	}
	if f.After != nil {
		p.AfterDate = pgtype.Date{Time: f.After.Date, Valid: true}
		p.AfterID = &f.After.ID
	}
	rows, err := New(s.Pool).ListReviews(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("query reviews: %w", err)
	}
	out := make([]reviews.Review, len(rows))
	for i, r := range rows {
		out[i] = reviews.Review{
			ID: r.ID, OutletID: r.OutletID, OutletName: r.OutletName, Source: r.Source,
			ReviewDate: r.ReviewDate.Time, Rating: int(r.Rating), Text: r.ReviewText, ReviewerName: r.ReviewerName,
			ReplyStatus: r.ReplyStatus,
		}
		if r.Tagged {
			out[i].Tags = &reviews.Tags{Themes: r.Themes, Sentiment: r.Sentiment, IsUrgent: r.IsUrgent,
				UrgentReasons: r.UrgentReasons, PromptVersion: int(r.PromptVersion)}
		}
	}
	return out, nil
}

// CountReviews counts every review in the scope matching f.
func (s *Store) CountReviews(ctx context.Context, scope auth.Scope, f reviews.Filter) (int, error) {
	p := CountReviewsParams{
		AllOutlets: scope.All, ScopeOutletID: scope.OutletID, OutletID: f.OutletID,
		Theme: optString(f.Theme), Sentiment: optString(f.Sentiment), IsUrgent: f.Urgent,
		DateFrom: optDate(f.From), DateTo: optDate(f.To), ReplyStatus: optString(f.ReplyStatus),
	}
	if f.Q != "" {
		q := reviews.LikePattern(f.Q)
		p.Q = &q
	}
	n, err := New(s.Pool).CountReviews(ctx, p)
	if err != nil {
		return 0, fmt.Errorf("count reviews: %w", err)
	}
	return int(n), nil
}
