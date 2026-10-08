// Package outlets holds the outlet rules: who may list and add outlets, and
// what a valid name is (US-01-001, phase 1 server LLD section 3). It has no
// SQL and no HTTP types.
package outlets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// maxNameRunes is the API's limit on outlet names (api/openapi.yaml).
const maxNameRunes = 200

// ErrRoleNotAllowed means the caller's role may not add outlets: only the
// brand admin can (AC-US-01-001-2).
var ErrRoleNotAllowed = errors.New("outlets: only the brand admin can add outlets")

// ValidationError is a name that breaks a rule; Reason is blank or too_long.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("outlets: %s is %s", e.Field, e.Reason)
}

// NameTakenError means an outlet with the name exists, ignoring capitals.
// Existing is that outlet's name as stored.
type NameTakenError struct{ Existing string }

func (e *NameTakenError) Error() string {
	return fmt.Sprintf("outlets: the name is taken by %q", e.Existing)
}

// Outlet is one stored outlet.
type Outlet struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	// ReviewCount and UntaggedCount are the outlet's reviews and those with
	// no stored tag result yet (OutletSummary in the API).
	ReviewCount   int
	UntaggedCount int
}

// Manager is an active outlet manager of one outlet.
type Manager struct {
	ID       int64
	Name     string
	OutletID int64
}

// Summary is an outlet as the list shows it (OutletSummary in the API).
type Summary struct {
	Outlet
	Managers []Manager
}

// Store reads and writes outlets. CreateOutlet reports created false when
// the name exists ignoring case.
type Store interface {
	ListOutlets(ctx context.Context, scope auth.Scope) ([]Outlet, error)
	ListActiveManagers(ctx context.Context, outletIDs []int64) ([]Manager, error)
	CreateOutlet(ctx context.Context, name string) (outlet Outlet, created bool, err error)
	OutletByName(ctx context.Context, name string) (Outlet, error)
}

// Service applies the outlet rules.
type Service struct {
	store Store
}

// NewService builds the outlet service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns the outlets the user may see, ordered by name ignoring case:
// every outlet for the brand admin, the manager's own for an outlet manager.
// The scope comes from the user row read on this request (tenet 3).
func (s *Service) List(ctx context.Context, user auth.User) ([]Summary, error) {
	list, err := s.store.ListOutlets(ctx, user.Scope())
	if err != nil {
		return nil, fmt.Errorf("list outlets: %w", err)
	}
	ids := make([]int64, len(list))
	for i, o := range list {
		ids[i] = o.ID
	}
	managers, err := s.store.ListActiveManagers(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list outlet managers: %w", err)
	}
	byOutlet := make(map[int64][]Manager, len(list))
	for _, m := range managers {
		byOutlet[m.OutletID] = append(byOutlet[m.OutletID], m)
	}
	out := make([]Summary, len(list))
	for i, o := range list {
		out[i] = Summary{Outlet: o, Managers: byOutlet[o.ID]}
	}
	return out, nil
}

// CheckCanCreate refuses every role but the brand admin. The handler calls it
// before reading the body, so a manager gets 403 whatever they send.
func CheckCanCreate(user auth.User) error {
	if user.Role != auth.RoleBrandAdmin {
		return ErrRoleNotAllowed
	}
	return nil
}

// ValidateName trims the name and checks it is 1 to 200 characters.
func ValidateName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", &ValidationError{Field: "name", Reason: "blank"}
	case utf8.RuneCountInString(trimmed) > maxNameRunes:
		return "", &ValidationError{Field: "name", Reason: "too_long"}
	}
	return trimmed, nil
}

// Create adds an outlet for the brand admin. A name that exists ignoring
// capitals is a NameTakenError naming the stored outlet, which also answers
// a repeated request (tenet 8).
func (s *Service) Create(ctx context.Context, user auth.User, name string) (Summary, error) {
	if err := CheckCanCreate(user); err != nil {
		return Summary{}, err
	}
	trimmed, err := ValidateName(name)
	if err != nil {
		return Summary{}, err
	}
	created, ok, err := s.store.CreateOutlet(ctx, trimmed)
	if err != nil {
		return Summary{}, fmt.Errorf("create outlet: %w", err)
	}
	if !ok {
		existing, err := s.store.OutletByName(ctx, trimmed)
		if err != nil {
			return Summary{}, fmt.Errorf("read the outlet that holds the name: %w", err)
		}
		return Summary{}, &NameTakenError{Existing: existing.Name}
	}
	return Summary{Outlet: created}, nil
}
