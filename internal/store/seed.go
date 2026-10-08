package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// SeedReview is one generated demo review.
type SeedReview struct {
	OutletID     int64
	Source       string
	Date         time.Time
	Rating       int
	Text         string
	ReviewerName string
}

// ResetDomain empties every domain table and restarts review ids at 1. The
// budget schema is never touched (HLD section 4). The caller holds the
// tagging lock (data model section 5).
func (s *Store) ResetDomain(ctx context.Context) error {
	q := New(s.Pool)
	if err := q.ResetDomainTables(ctx); err != nil {
		return fmt.Errorf("truncate domain tables: %w", err)
	}
	if err := q.RestartReviewIDs(ctx); err != nil {
		return fmt.Errorf("restart review ids: %w", err)
	}
	return nil
}

// InsertSeedReviews stores the reviews in order, with no import, and
// returns how many were stored.
func (s *Store) InsertSeedReviews(ctx context.Context, rs []SeedReview) (int, error) {
	var p InsertSeedReviewsParams
	for _, r := range rs {
		p.OutletIds = append(p.OutletIds, r.OutletID)
		p.Sources = append(p.Sources, r.Source)
		p.ReviewDates = append(p.ReviewDates, pgtype.Date{Time: r.Date, Valid: true})
		p.Ratings = append(p.Ratings, int16(r.Rating))
		p.ReviewTexts = append(p.ReviewTexts, r.Text)
		p.ReviewerNames = append(p.ReviewerNames, r.ReviewerName)
	}
	n, err := New(s.Pool).InsertSeedReviews(ctx, p)
	if err != nil {
		return 0, fmt.Errorf("insert seed reviews: %w", err)
	}
	return int(n), nil
}
