package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// The sign-in service reads accounts through the store.
var _ auth.UserStore = (*Store)(nil)

// UserByEmail returns the active account with this email, ignoring case, or
// auth.ErrUserNotFound. It implements auth.UserStore.
func (s *Store) UserByEmail(ctx context.Context, email string) (auth.UserRecord, error) {
	row, err := New(s.Pool).GetActiveUserByEmail(ctx, email)
	if err != nil {
		return auth.UserRecord{}, userLookupError(err)
	}
	return userRecord(row.ID, row.Email, row.Name, row.Role, row.OutletID, row.OutletName, row.PasswordHash, row.CreatedAt), nil
}

// UserByID returns the active account with this id, or auth.ErrUserNotFound.
// It implements auth.UserStore.
func (s *Store) UserByID(ctx context.Context, id int64) (auth.UserRecord, error) {
	row, err := New(s.Pool).GetActiveUserByID(ctx, id)
	if err != nil {
		return auth.UserRecord{}, userLookupError(err)
	}
	return userRecord(row.ID, row.Email, row.Name, row.Role, row.OutletID, row.OutletName, row.PasswordHash, row.CreatedAt), nil
}

func userLookupError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrUserNotFound
	}
	return fmt.Errorf("read user: %w", err)
}

func userRecord(id int64, email, name string, role UserRole, outletID *int64, outletName *string, hash string, createdAt time.Time) auth.UserRecord {
	u := auth.User{ID: id, Email: email, Name: name, Role: auth.Role(role), CreatedAt: createdAt}
	if outletID != nil && outletName != nil {
		u.Outlet = &auth.OutletRef{ID: *outletID, Name: *outletName}
	}
	return auth.UserRecord{User: u, PasswordHash: hash}
}
