// Package replies holds the reply rules: who may draft, edit and approve,
// the drafting claim, and the transitions none to drafting to draft to
// replied (US-00-002, US-00-003, phase 5 LLD section 3). It has no SQL and
// no HTTP types.
package replies

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// Errors the API maps to statuses (phase 5 LLD).
var (
	ErrRoleNotAllowed = errors.New("replies: only the outlet's manager can draft, edit or approve a reply")
	ErrNotFound       = errors.New("replies: no such review in the caller's outlet")
	// ErrBudgetExhausted covers the USD 8 stop and a provider 402 (503 budget_exhausted).
	ErrBudgetExhausted = errors.New("replies: drafting is unavailable, the model budget is used up")
	// ErrModelUnavailable covers every other drafting failure (503 model_unavailable).
	ErrModelUnavailable = errors.New("replies: drafting is unavailable, the model did not answer")
)

// ConflictError is a write refused by the reply's state (409); Code is
// reply_changed, draft_in_progress or already_replied.
type ConflictError struct{ Code string }

func (e *ConflictError) Error() string { return "replies: " + e.Code }

// ValidationError is reply text that breaks a rule (422).
type ValidationError struct{ Reason string }

func (e *ValidationError) Error() string { return "replies: reply_text is " + e.Reason }

// DraftDeadline bounds one draft request, every gateway attempt included
// (HLD section 8).
const DraftDeadline = 45 * time.Second

const maxReplyRunes = 5000

// Reply is a review's reply (Reply in the API).
type Reply struct {
	Status        string
	DraftText     *string
	PromptVersion *int
	ReplyText     *string
	RepliedBy     *outlets.Manager
	RepliedAt     *time.Time
	UpdatedAt     time.Time
}

// Edited is true when the text differs from the draft.
func (r Reply) Edited() bool {
	if r.ReplyText == nil {
		return false
	}
	return r.DraftText == nil || *r.DraftText != *r.ReplyText
}

// Detail is one review with its reply (ReviewDetail in the API).
type Detail struct {
	Review   reviews.Review
	Reply    *Reply
	Managers []outlets.Manager
	CanReply bool
}

// Store reads and writes replies. Writes take the manager's outlet and
// change nothing for a review outside it.
type Store interface {
	ReviewDetail(ctx context.Context, scope auth.Scope, reviewID int64) (reviews.Review, bool, error)
	Reply(ctx context.Context, reviewID int64) (*Reply, error)
	ListActiveManagers(ctx context.Context, outletIDs []int64) ([]outlets.Manager, error)
	ClaimDraft(ctx context.Context, reviewID, outletID int64) (claimed time.Time, ok bool, err error)
	TakeOverClaim(ctx context.Context, reviewID int64) (claimed time.Time, ok bool, err error)
	LandDraft(ctx context.Context, reviewID int64, claimed time.Time, text string, promptVersion int) (bool, error)
	ReleaseClaim(ctx context.Context, reviewID int64, claimed time.Time) error
	InsertHandReply(ctx context.Context, reviewID, outletID int64, text string) (bool, error)
	SaveReplyText(ctx context.Context, reviewID, outletID int64, text string, seen time.Time) (bool, error)
	MarkReplied(ctx context.Context, reviewID, outletID, userID int64, text string, seen time.Time) (bool, error)
}

// Model is the one door to the model (tenet 1).
type Model interface {
	Complete(ctx context.Context, r gateway.Request) (gateway.Response, error)
}

// Service applies the reply rules.
type Service struct {
	store  Store
	model  Model
	prompt prompts.Version
	logger *slog.Logger
}

// NewService builds the reply service with the current reply prompt.
func NewService(store Store, model Model, prompt prompts.Version, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{store: store, model: model, prompt: prompt, logger: logger}
}

// Get returns one review in the caller's scope with its reply and managers.
// It never calls the model.
func (s *Service) Get(ctx context.Context, user auth.User, reviewID int64) (Detail, error) {
	rv, ok, err := s.store.ReviewDetail(ctx, user.Scope(), reviewID)
	if err != nil {
		return Detail{}, fmt.Errorf("read review %d: %w", reviewID, err)
	}
	if !ok {
		return Detail{}, ErrNotFound
	}
	reply, err := s.store.Reply(ctx, reviewID)
	if err != nil {
		return Detail{}, fmt.Errorf("read reply %d: %w", reviewID, err)
	}
	managers, err := s.store.ListActiveManagers(ctx, []int64{rv.OutletID})
	if err != nil {
		return Detail{}, fmt.Errorf("read managers: %w", err)
	}
	if reply != nil {
		rv.ReplyStatus = reply.Status
	}
	return Detail{Review: rv, Reply: reply, Managers: managers, CanReply: canReply(user, rv.OutletID)}, nil
}

func canReply(user auth.User, outletID int64) bool {
	return user.Role == auth.RoleOutletManager && user.Outlet != nil && user.Outlet.ID == outletID
}

// manager checks the caller may write replies for the review and returns
// the review.
func (s *Service) manager(ctx context.Context, user auth.User, reviewID int64) (reviews.Review, error) {
	if user.Role != auth.RoleOutletManager || user.Outlet == nil {
		return reviews.Review{}, ErrRoleNotAllowed
	}
	rv, ok, err := s.store.ReviewDetail(ctx, user.Scope(), reviewID)
	if err != nil {
		return reviews.Review{}, fmt.Errorf("read review %d: %w", reviewID, err)
	}
	if !ok {
		return reviews.Review{}, ErrNotFound
	}
	return rv, nil
}

// Draft returns the stored reply, or drafts one through the gateway once
// (Q-007). drafting is true when another request holds the claim: the
// caller answers 202 and the client asks again.
func (s *Service) Draft(ctx context.Context, user auth.User, reviewID int64) (reply *Reply, drafting bool, err error) {
	rv, err := s.manager(ctx, user, reviewID)
	if err != nil {
		return nil, false, err
	}
	claimed, ok, err := s.store.ClaimDraft(ctx, reviewID, user.Outlet.ID)
	if err != nil {
		return nil, false, fmt.Errorf("claim draft %d: %w", reviewID, err)
	}
	if !ok {
		existing, err := s.store.Reply(ctx, reviewID)
		if err != nil {
			return nil, false, fmt.Errorf("read reply %d after a lost claim: %w", reviewID, err)
		}
		if existing == nil {
			// The other request released its claim in between: ask again.
			return nil, true, nil
		}
		if existing.Status != "drafting" {
			return existing, false, nil // AC-US-00-002-2: no model call
		}
		if claimed, ok, err = s.store.TakeOverClaim(ctx, reviewID); err != nil {
			return nil, false, fmt.Errorf("take over claim %d: %w", reviewID, err)
		}
		if !ok {
			return existing, true, nil
		}
	}

	dctx, cancel := context.WithTimeout(ctx, DraftDeadline)
	defer cancel()
	resp, err := s.model.Complete(dctx, gateway.Request{
		Purpose: gateway.Drafting,
		Prompt:  s.prompt,
		User: Message(Input{Outlet: rv.OutletName, ManagerName: user.Name, ReviewerName: rv.ReviewerName,
			Rating: rv.Rating, Text: rv.Text}),
	})
	text := strings.TrimSpace(resp.Text)
	if err == nil && (text == "" || resp.FinishReason == "length" || utf8.RuneCountInString(text) > maxReplyRunes) {
		err = fmt.Errorf("%w: the draft was blank, cut off or too long", gateway.ErrModelUnavailable)
	}
	if err != nil {
		s.release(ctx, reviewID, claimed)
		s.logger.WarnContext(ctx, "draft unavailable", "review_id", reviewID, "err", err)
		if errors.Is(err, gateway.ErrBudgetExhausted) || errors.Is(err, gateway.ErrProviderCreditExhausted) {
			return nil, false, ErrBudgetExhausted
		}
		return nil, false, ErrModelUnavailable
	}
	if _, err := s.store.LandDraft(context.WithoutCancel(ctx), reviewID, claimed, text, s.prompt.Number); err != nil {
		return nil, false, fmt.Errorf("store draft %d: %w", reviewID, err)
	}
	stored, err := s.store.Reply(ctx, reviewID)
	if err != nil {
		return nil, false, fmt.Errorf("read draft %d: %w", reviewID, err)
	}
	if stored == nil {
		return nil, true, nil // released by a newer claim in between: ask again
	}
	return stored, stored.Status == "drafting", nil
}

// release deletes the request's own claim so the manager can write by hand
// or try again; it runs even if the caller has gone.
func (s *Service) release(ctx context.Context, reviewID int64, claimed time.Time) {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.store.ReleaseClaim(rctx, reviewID, claimed); err != nil {
		s.logger.Error("draft claim not released; it expires in 60 seconds", "review_id", reviewID, "err", err)
	}
}

func checkText(text string) (string, error) {
	t := strings.TrimSpace(text)
	switch {
	case t == "":
		return "", &ValidationError{Reason: "blank"}
	case utf8.RuneCountInString(t) > maxReplyRunes:
		return "", &ValidationError{Reason: "too_long"}
	}
	return t, nil
}

// Save stores an edit, or a hand-written reply when seen is nil and there
// is none (AC-US-00-003-1, AC-US-00-002-7).
func (s *Service) Save(ctx context.Context, user auth.User, reviewID int64, text string, seen *time.Time) (*Reply, error) {
	if _, err := s.manager(ctx, user, reviewID); err != nil {
		return nil, err
	}
	t, err := checkText(text)
	if err != nil {
		return nil, err
	}
	var ok bool
	if seen == nil {
		ok, err = s.store.InsertHandReply(ctx, reviewID, user.Outlet.ID, t)
	} else {
		ok, err = s.store.SaveReplyText(ctx, reviewID, user.Outlet.ID, t, *seen)
	}
	if err != nil {
		return nil, fmt.Errorf("save reply %d: %w", reviewID, err)
	}
	return s.after(ctx, reviewID, ok, false)
}

// MarkReplied approves the text and marks the review replied (Q-008). A
// repeat on a replied review returns it unchanged.
func (s *Service) MarkReplied(ctx context.Context, user auth.User, reviewID int64, text string, seen *time.Time) (*Reply, error) {
	if _, err := s.manager(ctx, user, reviewID); err != nil {
		return nil, err
	}
	t, err := checkText(text)
	if err != nil {
		return nil, err
	}
	ok := false
	if seen != nil {
		if ok, err = s.store.MarkReplied(ctx, reviewID, user.Outlet.ID, user.ID, t, *seen); err != nil {
			return nil, fmt.Errorf("mark replied %d: %w", reviewID, err)
		}
	}
	return s.after(ctx, reviewID, ok, true)
}

// after reads the reply once a write ran: the new row when it matched, or
// the conflict that explains why it did not.
func (s *Service) after(ctx context.Context, reviewID int64, ok, replying bool) (*Reply, error) {
	r, err := s.store.Reply(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("read reply %d: %w", reviewID, err)
	}
	if ok && r != nil {
		return r, nil
	}
	switch {
	case r != nil && r.Status == "replied" && replying:
		return r, nil
	case r != nil && r.Status == "replied":
		return nil, &ConflictError{Code: "already_replied"}
	case r != nil && r.Status == "drafting":
		return nil, &ConflictError{Code: "draft_in_progress"}
	}
	return nil, &ConflictError{Code: "reply_changed"}
}
