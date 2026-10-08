package digest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
)

// ErrRoleNotAllowed means the caller is not the brand admin (HLD section 9).
var ErrRoleNotAllowed = errors.New("digest: only the brand admin can generate the digest")

// Digest is one generated digest (Digest in the API).
type Digest struct {
	ID            int64
	Status        string // sending, sent or failed
	WeekStart     time.Time
	Recipient     string
	Untagged      int
	Subject       string
	Body          string
	FailureReason *string
	SentAt        *time.Time
	CreatedAt     time.Time
}

// WeekEnd is the Sunday of the digest week.
func (d Digest) WeekEnd() time.Time { return d.WeekStart.AddDate(0, 0, 6) }

// MailError is a digest MailHog did not accept; it is stored as failed (502).
type MailError struct{ Digest Digest }

func (e *MailError) Error() string { return "digest: the mail catcher did not accept the digest" }

// Store keeps digests and reads what one holds.
type Store interface {
	DigestByRequestID(ctx context.Context, requestID string) (Digest, bool, error)
	ActiveBrandAdminEmail(ctx context.Context) (string, error)
	ClaimDigest(ctx context.Context, requestID string, d Digest) (Digest, bool, error)
	MarkDigestSent(ctx context.Context, id int64) (time.Time, error)
	MarkDigestFailed(ctx context.Context, id int64, reason string) error
	ListReviews(ctx context.Context, scope auth.Scope, f reviews.Filter, limit int) ([]reviews.Review, error)
}

// Reports computes the movers exactly as the dashboard does (AC-US-01-009-2).
type Reports interface {
	Movers(ctx context.Context, user auth.User) (dashboard.Movers, error)
}

// Mailer sends one message.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Service generates digests.
type Service struct {
	store   Store
	reports Reports
	mail    Mailer
}

// NewService builds the digest service.
func NewService(store Store, reports Reports, mail Mailer) *Service {
	return &Service{store: store, reports: reports, mail: mail}
}

// maxUrgent bounds the urgent reviews one digest lists; a week holds a few.
const maxUrgent = 200

// Generate builds the digest for the latest complete week, records it as
// sending under the request id, sends one email to the brand admin and marks
// it sent or failed. A repeat of the request id returns the recorded digest
// and never sends again (tenet 8).
func (s *Service) Generate(ctx context.Context, user auth.User, requestID string) (Digest, error) {
	if user.Role != auth.RoleBrandAdmin {
		return Digest{}, ErrRoleNotAllowed
	}
	if d, ok, err := s.store.DigestByRequestID(ctx, requestID); err != nil {
		return Digest{}, fmt.Errorf("look up digest %s: %w", requestID, err)
	} else if ok {
		return repeat(d)
	}

	m, err := s.reports.Movers(ctx, user)
	if err != nil {
		return Digest{}, fmt.Errorf("digest movers: %w", err)
	}
	urgent := true
	list, err := s.store.ListReviews(ctx, user.Scope(), reviews.Filter{Urgent: &urgent, From: &m.Week.Start, To: &m.Week.End}, maxUrgent)
	if err != nil {
		return Digest{}, fmt.Errorf("digest urgent reviews: %w", err)
	}
	to, err := s.store.ActiveBrandAdminEmail(ctx)
	if err != nil {
		return Digest{}, fmt.Errorf("digest recipient: %w", err)
	}
	subject, body := Compose(m, list)
	d, created, err := s.store.ClaimDigest(ctx, requestID, Digest{WeekStart: m.Week.Start, Recipient: to, Untagged: m.Untagged, Subject: subject, Body: body})
	if err != nil {
		return Digest{}, fmt.Errorf("claim digest: %w", err)
	}
	if !created {
		return repeat(d) // the same request id committed meanwhile
	}

	// The send and the status write run even if the browser goes away, so
	// a sent email is always recorded as sent.
	wctx := context.WithoutCancel(ctx)
	if err := s.mail.Send(wctx, to, subject, body); err != nil {
		reason := "The mail catcher did not accept the digest: " + err.Error()
		if ferr := s.store.MarkDigestFailed(wctx, d.ID, reason); ferr != nil {
			return Digest{}, fmt.Errorf("record failed digest %d: %w", d.ID, ferr)
		}
		d.Status, d.FailureReason = "failed", &reason
		return Digest{}, &MailError{Digest: d}
	}
	sent, err := s.store.MarkDigestSent(wctx, d.ID)
	if err != nil {
		return Digest{}, fmt.Errorf("record sent digest %d: %w", d.ID, err)
	}
	d.Status, d.SentAt = "sent", &sent
	return d, nil
}

func repeat(d Digest) (Digest, error) {
	if d.Status == "failed" {
		return Digest{}, &MailError{Digest: d}
	}
	return d, nil
}
