package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
)

// The outlet service reads and writes outlets through the store.
var _ outlets.Store = (*Store)(nil)

// ListOutlets returns the outlets in the scope, ordered by name ignoring case.
func (s *Store) ListOutlets(ctx context.Context, scope auth.Scope) ([]outlets.Outlet, error) {
	rows, err := New(s.Pool).ListOutlets(ctx, ListOutletsParams{AllOutlets: scope.All, OutletID: scope.OutletID})
	if err != nil {
		return nil, fmt.Errorf("query outlets: %w", err)
	}
	out := make([]outlets.Outlet, len(rows))
	for i, r := range rows {
		out[i] = outlets.Outlet{ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt,
			ReviewCount: int(r.ReviewCount), UntaggedCount: int(r.UntaggedCount)}
	}
	return out, nil
}

// ListActiveManagers returns the active managers of the given outlets.
func (s *Store) ListActiveManagers(ctx context.Context, outletIDs []int64) ([]outlets.Manager, error) {
	if len(outletIDs) == 0 {
		return nil, nil
	}
	rows, err := New(s.Pool).ListActiveManagers(ctx, outletIDs)
	if err != nil {
		return nil, fmt.Errorf("query managers: %w", err)
	}
	out := make([]outlets.Manager, 0, len(rows))
	for _, r := range rows {
		if r.OutletID == nil {
			continue // the query filters on outlet_id; a manager always has one
		}
		out = append(out, outlets.Manager{ID: r.ID, Name: r.Name, OutletID: *r.OutletID})
	}
	return out, nil
}

// CreateOutlet inserts the outlet; created is false when the name exists
// ignoring case (uq_outlets_name_lower), and nothing is written.
func (s *Store) CreateOutlet(ctx context.Context, name string) (outlets.Outlet, bool, error) {
	r, err := New(s.Pool).CreateOutlet(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return outlets.Outlet{}, false, nil
	}
	if err != nil {
		return outlets.Outlet{}, false, fmt.Errorf("insert outlet: %w", err)
	}
	return outlets.Outlet{ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt}, true, nil
}

// OutletByName returns the outlet with this name, ignoring case.
func (s *Store) OutletByName(ctx context.Context, name string) (outlets.Outlet, error) {
	r, err := New(s.Pool).GetOutletByName(ctx, name)
	if err != nil {
		return outlets.Outlet{}, fmt.Errorf("query outlet by name: %w", err)
	}
	return outlets.Outlet{ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt}, nil
}
