package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrInvalidCredentials means the email and password match no active
// account. A wrong password and an unknown or removed email get the same
// error, so the answer never says which emails exist.
var ErrInvalidCredentials = errors.New("auth: email or password is wrong")

// ErrUserNotFound is what a UserStore returns when no active account
// matches.
var ErrUserNotFound = errors.New("auth: no active user")

// OutletRef names an outlet.
type OutletRef struct {
	ID   int64
	Name string
}

// User is the signed-in caller as the database says on this request: role
// and outlet are never taken from the token (ADR-0007, tenet 3).
type User struct {
	ID        int64
	Email     string
	Name      string
	Role      Role
	Outlet    *OutletRef // nil for the brand admin
	CreatedAt time.Time
}

// Scope is which outlets a query may touch: every outlet, or one.
type Scope struct {
	All      bool
	OutletID int64
}

// Scope is the only way to build a Scope, so it always comes from the user
// row read on this request, never from a URL, a claim or the UI (tenet 3).
// A manager row without an outlet (the database forbids it) scopes to
// outlet 0, which matches nothing.
func (u User) Scope() Scope {
	if u.Role == RoleBrandAdmin {
		return Scope{All: true}
	}
	if u.Outlet == nil {
		return Scope{}
	}
	return Scope{OutletID: u.Outlet.ID}
}

// UserRecord is a user with its password hash, as the store returns it.
type UserRecord struct {
	User
	PasswordHash string
}

// UserStore reads active accounts. Both methods return ErrUserNotFound when
// no active account matches; the email lookup ignores case.
type UserStore interface {
	UserByEmail(ctx context.Context, email string) (UserRecord, error)
	UserByID(ctx context.Context, id int64) (UserRecord, error)
}

// Service signs people in and resolves the caller of each request.
type Service struct {
	store  UserStore
	tokens *Tokens
}

// NewService builds the sign-in service.
func NewService(store UserStore, tokens *Tokens) *Service {
	return &Service{store: store, tokens: tokens}
}

// Login checks the password and returns the user and a new session token.
// An unknown email still spends one bcrypt check, so it takes as long as a
// wrong password.
func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	rec, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		CheckDummyPassword(password)
		return User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return User{}, "", fmt.Errorf("sign in: look up user: %w", err)
	}
	if !CheckPassword(rec.PasswordHash, password) {
		return User{}, "", ErrInvalidCredentials
	}
	token, err := s.tokens.Issue(rec.ID)
	if err != nil {
		return User{}, "", fmt.Errorf("sign in: %w", err)
	}
	return rec.User, token, nil
}

// Authenticate verifies a session token and re-reads its user. It returns
// ErrUnauthenticated (wrapped) when the token is invalid or expired, when the
// account is gone or removed, and when the token was issued before the
// account's row existed: user ids restart after `goose down` or
// `make db-reset`, so an old token must not name a new account.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	claims, err := s.tokens.Verify(token)
	if err != nil {
		return User{}, err
	}
	rec, err := s.store.UserByID(ctx, claims.UserID)
	if errors.Is(err, ErrUserNotFound) {
		return User{}, fmt.Errorf("%w: user %d is not active", ErrUnauthenticated, claims.UserID)
	}
	if err != nil {
		return User{}, fmt.Errorf("authenticate: look up user: %w", err)
	}
	// iat has whole seconds; compare against the row's second.
	if claims.IssuedAt.Before(rec.CreatedAt.Truncate(time.Second)) {
		return User{}, fmt.Errorf("%w: token is older than user %d", ErrUnauthenticated, claims.UserID)
	}
	return rec.User, nil
}
