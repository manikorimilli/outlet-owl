package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// SyncUsersResult says what one users file sync did.
type SyncUsersResult struct {
	// Changed counts accounts inserted or updated; an entry that already
	// matched its row is not counted.
	Changed int64
	// Removed counts accounts marked removed because the file no longer
	// holds them (or holds them only in a skipped entry).
	Removed int64
	// Skipped lists manager entries whose outlet does not exist.
	Skipped []auth.EntryProblem
}

// upsert is one account to write, with its entry number in the users file.
type upsert struct {
	entry  int
	params UpsertUserParams
}

// SyncUsers applies the parsed users file in one transaction (phase 1 server
// LLD, section 5): it resolves each manager's outlet by name and skips the
// managers whose outlet does not exist, marks every account missing from the
// rest removed, then upserts the managers first and the brand admin last, so
// neither an admin email change nor a role swap ever leaves two active admins.
// A failure rolls everything back. entries must hold the one brand admin
// auth.ParseUsers guarantees.
func (s *Store) SyncUsers(ctx context.Context, entries []auth.UsersFileEntry) (SyncUsersResult, error) {
	var res SyncUsersResult
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := New(tx)

		var names []string
		for _, e := range entries {
			if e.Role == auth.RoleOutletManager && e.Outlet != nil {
				names = append(names, *e.Outlet)
			}
		}
		outlets, err := q.ResolveOutletsByName(ctx, names)
		if err != nil {
			return fmt.Errorf("resolve outlets: %w", err)
		}
		outletIDs := make(map[string]int64, len(outlets))
		for _, o := range outlets {
			outletIDs[strings.ToLower(o.Name)] = o.ID
		}

		var managers, admins []upsert
		for _, e := range entries {
			p := UpsertUserParams{Email: e.Email, Name: e.Name, Role: UserRole(e.Role), PasswordHash: e.PasswordHash}
			if e.Role != auth.RoleOutletManager {
				admins = append(admins, upsert{entry: e.Entry, params: p})
				continue
			}
			id, ok := outletIDs[strings.ToLower(*e.Outlet)]
			if !ok {
				res.Skipped = append(res.Skipped, auth.EntryProblem{
					Entry: e.Entry, Field: "outlet",
					Reason: "no outlet with this name exists yet; add it on the Outlets screen, then restart",
				})
				continue
			}
			p.OutletID = &id
			managers = append(managers, upsert{entry: e.Entry, params: p})
		}
		kept := append(managers, admins...) // managers first, the admin last
		if len(admins) == 0 {
			return errors.New("the users file holds no brand admin")
		}

		emails := make([]string, len(kept))
		for i, u := range kept {
			emails[i] = u.params.Email
		}
		if res.Removed, err = q.MarkUsersRemovedExcept(ctx, emails); err != nil {
			return fmt.Errorf("mark removed users: %w", err)
		}
		for _, u := range kept {
			n, err := q.UpsertUser(ctx, u.params)
			if err != nil {
				// The entry number finds the line; no field value goes in
				// the message, and the database error names only the
				// constraint, never the row.
				return fmt.Errorf("upsert users file entry %d (role %s): %w", u.entry, u.params.Role, err)
			}
			res.Changed += n
		}
		return nil
	})
	if err != nil {
		return SyncUsersResult{}, fmt.Errorf("sync users: %w", err)
	}
	return res, nil
}
