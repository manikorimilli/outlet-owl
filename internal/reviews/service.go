// Package reviews holds the review list rules: the filters, the outlet scope
// and paging (US-01-008, US-00-001, phase 4 LLD section 3). It has no SQL and
// no HTTP types.
package reviews

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

// ErrNotFound means the caller asked for an outlet outside their scope; the
// API answers 404 so it does not say whether the outlet exists (tenet 3).
var ErrNotFound = errors.New("reviews: no such outlet in the caller's scope")

// ValidationError is a filter that parsed but breaks a rule (422).
type ValidationError struct{ Field, Reason string }

func (e *ValidationError) Error() string { return fmt.Sprintf("reviews: %s is %s", e.Field, e.Reason) }

// Tags are a review's stored tag result.
type Tags struct {
	Themes        []string
	Sentiment     string
	IsUrgent      bool
	UrgentReasons []string
	PromptVersion int
}

// Review is one review as the list shows it (ReviewSummary).
type Review struct {
	ID           int64
	OutletID     int64
	OutletName   string
	Source       string
	ReviewDate   time.Time
	Rating       int
	Text         string
	ReviewerName string
	// Tags is nil while the review is untagged.
	Tags *Tags
	// ReplyStatus is none until the replies table arrives in phase 5.
	ReplyStatus string
}

// Filter is the validated query of GET /reviews.
type Filter struct {
	Q           string
	OutletID    *int64
	Theme       string
	Sentiment   string
	Urgent      *bool
	ReplyStatus string
	From, To    *time.Time
	Limit       int
	After       *Cursor
}

// Page is one page of results.
type Page struct {
	Reviews []Review
	Next    *Cursor
	Total   int
}

// Store reads reviews within a scope.
type Store interface {
	ListReviews(ctx context.Context, scope auth.Scope, f Filter, limit int) ([]Review, error)
	CountReviews(ctx context.Context, scope auth.Scope, f Filter) (int, error)
}

// Service applies the list rules.
type Service struct{ store Store }

// NewService builds the review service.
func NewService(store Store) *Service { return &Service{store: store} }

// DefaultLimit and MaxLimit bound a page (api/openapi.yaml, Limit).
const (
	DefaultLimit = 50
	MaxLimit     = 100
	maxQRunes    = 200
)

// List returns one page of the caller's reviews, newest first.
func (s *Service) List(ctx context.Context, user auth.User, f Filter) (Page, error) {
	scope := user.Scope()
	if f.OutletID != nil && !scope.All && *f.OutletID != scope.OutletID {
		return Page{}, ErrNotFound
	}
	f.Q = strings.TrimSpace(f.Q)
	if utf8.RuneCountInString(f.Q) > maxQRunes {
		return Page{}, &ValidationError{Field: "q", Reason: "too_long"}
	}
	if f.Theme != "" && !slices.ContainsFunc(tagging.Themes, func(t tagging.Theme) bool { return t.Code == f.Theme }) {
		return Page{}, &ValidationError{Field: "filter[theme]", Reason: "unknown_theme"}
	}
	if f.Limit == 0 {
		f.Limit = DefaultLimit
	}
	// Until phase 5 adds replies, every review is "none": a filter for a
	// draft or a replied review matches nothing.
	if f.ReplyStatus == "draft" || f.ReplyStatus == "replied" {
		return Page{Reviews: []Review{}}, nil
	}
	rows, err := s.store.ListReviews(ctx, scope, f, f.Limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("list reviews: %w", err)
	}
	total, err := s.store.CountReviews(ctx, scope, f)
	if err != nil {
		return Page{}, fmt.Errorf("count reviews: %w", err)
	}
	p := Page{Reviews: rows, Total: total}
	if len(rows) > f.Limit {
		p.Reviews = rows[:f.Limit]
		last := p.Reviews[f.Limit-1]
		p.Next = &Cursor{Date: last.ReviewDate, ID: last.ID}
	}
	return p, nil
}

// LikePattern turns a search term into an ILIKE pattern that matches it as a
// substring, with %, _ and the escape character taken literally.
func LikePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}
