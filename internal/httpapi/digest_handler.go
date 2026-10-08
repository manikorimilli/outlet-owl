package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/digest"
)

// DigestService generates the digest. *digest.Service satisfies it.
type DigestService interface {
	Generate(ctx context.Context, user auth.User, requestID string) (digest.Digest, error)
}

// digestJSON is Digest in api/openapi.yaml.
type digestJSON struct {
	ID             int64      `json:"id"`
	Status         string     `json:"status"`
	WeekStart      string     `json:"week_start"`
	WeekEnd        string     `json:"week_end"`
	RecipientEmail string     `json:"recipient_email"`
	UntaggedCount  int        `json:"untagged_count"`
	Subject        string     `json:"subject"`
	Body           string     `json:"body"`
	SentAt         *time.Time `json:"sent_at"`
	FailureReason  *string    `json:"failure_reason"`
	CreatedAt      time.Time  `json:"created_at"`
}

// createDigest answers POST /api/v1/digests (operationId createDigest): 201
// sent, 202 a repeat still marked sending, 502 mail_unavailable.
func createDigest(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		if user.Role != auth.RoleBrandAdmin {
			writeDomainError(w, r, d.Logger, digest.ErrRoleNotAllowed)
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if !uuidPattern.MatchString(key) {
			writeDomainError(w, r, d.Logger, &malformedError{reason: "the Idempotency-Key header must be a UUID"})
			return
		}
		dg, err := d.Digests.Generate(r.Context(), user, strings.ToLower(key))
		var mailErr *digest.MailError
		if errors.As(err, &mailErr) {
			d.Logger.WarnContext(r.Context(), "digest not sent", "digest_id", mailErr.Digest.ID, "reason", deref(mailErr.Digest.FailureReason))
			WriteError(w, r, http.StatusBadGateway, "mail_unavailable", "The mail catcher (MailHog) did not answer. Start MailHog and generate the digest again.")
			return
		}
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		status := http.StatusCreated
		if dg.Status == "sending" {
			status = http.StatusAccepted
		}
		WriteJSON(w, status, digestJSON{ID: dg.ID, Status: dg.Status, WeekStart: date(dg.WeekStart), WeekEnd: date(dg.WeekEnd()),
			RecipientEmail: dg.Recipient, UntaggedCount: dg.Untagged, Subject: dg.Subject, Body: dg.Body,
			SentAt: dg.SentAt, FailureReason: dg.FailureReason, CreatedAt: dg.CreatedAt.UTC()})
	})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
