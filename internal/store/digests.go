package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/manikorimilli/outlet-owl/internal/digest"
)

// The digest service keeps digests through the store.
var _ digest.Store = (*Store)(nil)

// DigestByRequestID returns the digest stored under the request id.
func (s *Store) DigestByRequestID(ctx context.Context, requestID string) (digest.Digest, bool, error) {
	r, err := New(s.Pool).GetDigestByRequestID(ctx, requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return digest.Digest{}, false, nil
	}
	if err != nil {
		return digest.Digest{}, false, fmt.Errorf("query digest: %w", err)
	}
	return digest.Digest{ID: r.ID, Status: r.Status, WeekStart: r.WeekStart.Time, Recipient: r.RecipientEmail,
		Untagged: int(r.UntaggedCount), Subject: r.Subject, Body: r.Body, FailureReason: r.FailureReason,
		SentAt: r.SentAt, CreatedAt: r.CreatedAt}, true, nil
}

// ActiveBrandAdminEmail returns the brand admin's address.
func (s *Store) ActiveBrandAdminEmail(ctx context.Context) (string, error) {
	email, err := New(s.Pool).ActiveBrandAdminEmail(ctx)
	if err != nil {
		return "", fmt.Errorf("query brand admin: %w", err)
	}
	return email, nil
}

// ClaimDigest records the digest as sending; created is false when the
// request id already holds one, which is then returned.
func (s *Store) ClaimDigest(ctx context.Context, requestID string, d digest.Digest) (digest.Digest, bool, error) {
	r, err := New(s.Pool).ClaimDigest(ctx, ClaimDigestParams{RequestID: requestID, WeekStart: pgtype.Date{Time: d.WeekStart, Valid: true},
		RecipientEmail: d.Recipient, UntaggedCount: int32(d.Untagged), Subject: d.Subject, Body: d.Body})
	if errors.Is(err, pgx.ErrNoRows) {
		prev, ok, err := s.DigestByRequestID(ctx, requestID)
		if err != nil {
			return digest.Digest{}, false, fmt.Errorf("read the digest that holds the request id: %w", err)
		}
		if !ok {
			return digest.Digest{}, false, errors.New("the digest that holds the request id is gone")
		}
		return prev, false, nil
	}
	if err != nil {
		return digest.Digest{}, false, fmt.Errorf("claim digest: %w", err)
	}
	d.ID, d.CreatedAt, d.Status = r.ID, r.CreatedAt, "sending"
	return d, true, nil
}

// MarkDigestSent records the send.
func (s *Store) MarkDigestSent(ctx context.Context, id int64) (time.Time, error) {
	t, err := New(s.Pool).MarkDigestSent(ctx, id)
	if err != nil {
		return time.Time{}, fmt.Errorf("mark digest sent: %w", err)
	}
	if t == nil {
		return time.Time{}, errors.New("mark digest sent: no sent time")
	}
	return *t, nil
}

// MarkDigestFailed records why the send failed.
func (s *Store) MarkDigestFailed(ctx context.Context, id int64, reason string) error {
	if err := New(s.Pool).MarkDigestFailed(ctx, MarkDigestFailedParams{ID: id, Reason: reason}); err != nil {
		return fmt.Errorf("mark digest failed: %w", err)
	}
	return nil
}
