package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

// The tagging worker reads reviews and stores results through the store.
var _ tagging.Store = (*Store)(nil)

// LockTagging takes the tagging advisory lock on a connection of its own and
// holds it until unlock closes that connection. A session lock taken through
// the pool would go back to the pool with its connection and could be taken
// again by a second holder on it (HLD section 16, MINOR advisory lock).
func (s *Store) LockTagging(ctx context.Context) (func(), error) {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire a connection: %w", err)
	}
	if err := New(conn).LockTagging(ctx, tagging.LockKey); err != nil {
		closeAndRelease(conn.Conn(), conn.Release)
		return nil, fmt.Errorf("advisory lock: %w", err)
	}
	return func() { closeAndRelease(conn.Conn(), conn.Release) }, nil
}

// closeAndRelease ends the session, which drops its advisory locks, then
// hands the closed connection back for the pool to discard.
func closeAndRelease(c *pgx.Conn, release func()) {
	_ = c.Close(context.Background()) // a failed close also ends the session
	release()
}

// UntaggedReviewIDs returns the ids of reviews with no tag result above
// afterID, in id order.
func (s *Store) UntaggedReviewIDs(ctx context.Context, afterID int64) ([]int64, error) {
	ids, err := New(s.Pool).UntaggedReviewIDs(ctx, afterID)
	if err != nil {
		return nil, fmt.Errorf("query untagged reviews: %w", err)
	}
	return ids, nil
}

// ReviewTexts returns the id and text of each review in ids, by id.
func (s *Store) ReviewTexts(ctx context.Context, ids []int64) ([]tagging.Review, error) {
	rows, err := New(s.Pool).ReviewTexts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("query review texts: %w", err)
	}
	out := make([]tagging.Review, len(rows))
	for i, r := range rows {
		out[i] = tagging.Review{ID: r.ID, Text: r.ReviewText}
	}
	return out, nil
}

// SaveResults stores each result unless its review already has one
// (REQ-011, tenet 4), in one transaction, and returns how many it stored.
func (s *Store) SaveResults(ctx context.Context, results []tagging.Result, promptVersion int) (int, error) {
	if len(results) == 0 {
		return 0, nil
	}
	stored := 0
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := New(s.Pool).WithTx(tx)
		for _, r := range results {
			n, err := q.InsertReviewTag(ctx, InsertReviewTagParams{
				ReviewID: r.ReviewID, Themes: r.Themes, Sentiment: Sentiment(r.Sentiment),
				IsUrgent: r.IsUrgent(), UrgentReasons: r.UrgentReasons, PromptVersion: int32(promptVersion),
			})
			if err != nil {
				return fmt.Errorf("insert the tags of review %d: %w", r.ReviewID, err)
			}
			stored += int(n)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return stored, nil
}
