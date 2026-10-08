package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/replies"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
)

// The reply service reads and writes replies through the store.
var _ replies.Store = (*Store)(nil)

// ReviewDetail returns one review in the scope; ok is false when it is not.
func (s *Store) ReviewDetail(ctx context.Context, scope auth.Scope, reviewID int64) (reviews.Review, bool, error) {
	r, err := New(s.Pool).GetReviewDetail(ctx, GetReviewDetailParams{ID: reviewID, AllOutlets: scope.All, ScopeOutletID: scope.OutletID})
	if errors.Is(err, pgx.ErrNoRows) {
		return reviews.Review{}, false, nil
	}
	if err != nil {
		return reviews.Review{}, false, fmt.Errorf("query review %d: %w", reviewID, err)
	}
	out := reviews.Review{ID: r.ID, OutletID: r.OutletID, OutletName: r.OutletName, Source: r.Source,
		ReviewDate: r.ReviewDate.Time, Rating: int(r.Rating), Text: r.ReviewText, ReviewerName: r.ReviewerName,
		ReplyStatus: "none"}
	if r.Tagged {
		out.Tags = &reviews.Tags{Themes: r.Themes, Sentiment: r.Sentiment, IsUrgent: r.IsUrgent,
			UrgentReasons: r.UrgentReasons, PromptVersion: int(r.TagPromptVersion)}
	}
	return out, true, nil
}

// Reply returns the review's reply, or nil when it has none.
func (s *Store) Reply(ctx context.Context, reviewID int64) (*replies.Reply, error) {
	r, err := New(s.Pool).GetReply(ctx, reviewID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query reply %d: %w", reviewID, err)
	}
	out := &replies.Reply{Status: r.Status, DraftText: r.DraftText, ReplyText: r.ReplyText, RepliedAt: r.RepliedAt, UpdatedAt: r.UpdatedAt}
	if r.PromptVersion != nil {
		v := int(*r.PromptVersion)
		out.PromptVersion = &v
	}
	if r.RepliedByID != nil && r.RepliedByName != nil {
		out.RepliedBy = &outlets.Manager{ID: *r.RepliedByID, Name: *r.RepliedByName}
	}
	return out, nil
}

func returned(t time.Time, err error, what string) (time.Time, bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%s: %w", what, err)
	}
	return t, true, nil
}

// ClaimDraft inserts the drafting claim; ok is false when a reply exists.
func (s *Store) ClaimDraft(ctx context.Context, reviewID, outletID int64) (time.Time, bool, error) {
	t, err := New(s.Pool).ClaimDraft(ctx, ClaimDraftParams{ReviewID: reviewID, OutletID: outletID})
	return returned(t, err, "claim draft")
}

// TakeOverClaim takes a drafting claim older than 60 seconds.
func (s *Store) TakeOverClaim(ctx context.Context, reviewID int64) (time.Time, bool, error) {
	t, err := New(s.Pool).TakeOverClaim(ctx, reviewID)
	return returned(t, err, "take over claim")
}

// LandDraft stores the draft only while the claim is still this request's.
func (s *Store) LandDraft(ctx context.Context, reviewID int64, claimed time.Time, text string, promptVersion int) (bool, error) {
	v := int32(promptVersion)
	n, err := New(s.Pool).LandDraft(ctx, LandDraftParams{ReviewID: reviewID, Claimed: claimed, Text: text, PromptVersion: &v})
	if err != nil {
		return false, fmt.Errorf("land draft: %w", err)
	}
	return n == 1, nil
}

// ReleaseClaim deletes the request's own drafting claim.
func (s *Store) ReleaseClaim(ctx context.Context, reviewID int64, claimed time.Time) error {
	if err := New(s.Pool).ReleaseClaim(ctx, ReleaseClaimParams{ReviewID: reviewID, Claimed: claimed}); err != nil {
		return fmt.Errorf("release claim: %w", err)
	}
	return nil
}

// InsertHandReply stores a hand-written reply when the review has none.
func (s *Store) InsertHandReply(ctx context.Context, reviewID, outletID int64, text string) (bool, error) {
	n, err := New(s.Pool).InsertHandReply(ctx, InsertHandReplyParams{ReviewID: reviewID, OutletID: outletID, Text: text})
	if err != nil {
		return false, fmt.Errorf("insert reply: %w", err)
	}
	return n == 1, nil
}

// SaveReplyText stores an edit when the draft is unchanged since seen.
func (s *Store) SaveReplyText(ctx context.Context, reviewID, outletID int64, text string, seen time.Time) (bool, error) {
	n, err := New(s.Pool).SaveReplyText(ctx, SaveReplyTextParams{ReviewID: reviewID, OutletID: outletID, Text: text, Seen: seen})
	if err != nil {
		return false, fmt.Errorf("save reply: %w", err)
	}
	return n == 1, nil
}

// MarkReplied approves the text when the draft is unchanged since seen.
func (s *Store) MarkReplied(ctx context.Context, reviewID, outletID, userID int64, text string, seen time.Time) (bool, error) {
	n, err := New(s.Pool).MarkReplied(ctx, MarkRepliedParams{ReviewID: reviewID, OutletID: outletID, UserID: &userID, Text: text, Seen: seen})
	if err != nil {
		return false, fmt.Errorf("mark replied: %w", err)
	}
	return n == 1, nil
}
