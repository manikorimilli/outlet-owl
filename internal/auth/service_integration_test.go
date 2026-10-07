//go:build integration

package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

// The password every account in these tests has, hashed at a low cost: the
// database and CheckPassword do not check the cost (the users file does).
const testPassword = "correct-horse-battery"

type env struct {
	ctx   context.Context
	db    *storetest.DB
	store *store.Store
	svc   *auth.Service
	hash  string
}

func newEnv(t *testing.T, now func() time.Time, outlets ...string) *env {
	t.Helper()
	db := storetest.New(t)
	ctx := context.Background()
	for _, name := range outlets {
		if _, err := db.Pool.Exec(ctx, "INSERT INTO outlets (name) VALUES ($1)", name); err != nil {
			t.Fatalf("insert outlet: %v", err)
		}
	}
	tokens, err := auth.NewTokens([]byte("0123456789abcdef0123456789abcdef"), now)
	if err != nil {
		t.Fatalf("NewTokens: %v", err)
	}
	st := &store.Store{Pool: db.Pool}
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return &env{ctx: ctx, db: db, store: st, svc: auth.NewService(st, tokens), hash: string(hash)}
}

func (e *env) sync(t *testing.T, entries ...auth.UsersFileEntry) {
	t.Helper()
	for i := range entries {
		entries[i].PasswordHash = e.hash
	}
	if _, err := e.store.SyncUsers(e.ctx, entries); err != nil {
		t.Fatalf("SyncUsers: %v", err)
	}
}

func admin(email string) auth.UsersFileEntry {
	return auth.UsersFileEntry{Email: email, Name: "Ritika Rao", Role: auth.RoleBrandAdmin}
}

func manager(email, outlet string) auth.UsersFileEntry {
	return auth.UsersFileEntry{Email: email, Name: "Arjun Mehta", Role: auth.RoleOutletManager, Outlet: &outlet}
}

func TestLogin_EmailMatchesIgnoringCase(t *testing.T) {
	e := newEnv(t, nil)
	e.sync(t, admin("Ritika.Rao@Example.in"))

	u, token, err := e.svc.Login(e.ctx, "ritika.rao@EXAMPLE.IN", testPassword)

	if err != nil || token == "" || u.Role != auth.RoleBrandAdmin {
		t.Fatalf("Login = %+v, token %q, %v", u, token, err)
	}
}

func TestLogin_RemovedUserIsInvalidCredentials(t *testing.T) {
	e := newEnv(t, nil, "Koramangala")
	e.sync(t, admin("a@example.in"), manager("m@example.in", "Koramangala"))
	e.sync(t, admin("a@example.in"))

	if _, _, err := e.svc.Login(e.ctx, "m@example.in", testPassword); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_RereadsRoleAndOutletEachRequest(t *testing.T) {
	e := newEnv(t, nil, "Koramangala", "Indiranagar")
	e.sync(t, admin("a@example.in"), manager("m@example.in", "Koramangala"))
	_, token, err := e.svc.Login(e.ctx, "m@example.in", testPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// The users file moves the manager; the same token now sees the new outlet.
	e.sync(t, admin("a@example.in"), manager("m@example.in", "Indiranagar"))

	u, err := e.svc.Authenticate(e.ctx, token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if u.Outlet == nil || u.Outlet.Name != "Indiranagar" || u.Scope().OutletID != u.Outlet.ID {
		t.Fatalf("user = %+v, want the new outlet Indiranagar and its scope", u)
	}
}

func TestAuthenticate_RemovedUserIsUnauthenticated(t *testing.T) {
	e := newEnv(t, nil, "Koramangala")
	e.sync(t, admin("a@example.in"), manager("m@example.in", "Koramangala"))
	_, token, err := e.svc.Login(e.ctx, "m@example.in", testPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	e.sync(t, admin("a@example.in"))

	if _, err := e.svc.Authenticate(e.ctx, token); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated", err)
	}
}

func TestAuthenticate_TokenOlderThanUserRowIsRefused(t *testing.T) {
	// The token is issued an hour ago, against a row made two hours ago, so
	// it is valid; then the tables are dropped and recreated, as goose down
	// and up or make db-reset do, and id 1 now names a new account.
	e := newEnv(t, func() time.Time { return time.Now().Add(-time.Hour) })
	e.sync(t, admin("old-admin@example.in"))
	if _, err := e.db.Pool.Exec(e.ctx, "UPDATE users SET created_at = now() - interval '2 hours'"); err != nil {
		t.Fatalf("age the row: %v", err)
	}
	_, token, err := e.svc.Login(e.ctx, "old-admin@example.in", testPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := e.svc.Authenticate(e.ctx, token); err != nil {
		t.Fatalf("before the reset the token must work: %v", err)
	}

	e.db.Goose(t, "down-to", "0")
	e.db.Goose(t, "up")
	// The recreated user_role enum has a new type id; pooled connections
	// still hold the old one, as a server left running would.
	e.db.Pool.Reset()
	e.sync(t, admin("new-admin@example.in"))

	if _, err := e.svc.Authenticate(e.ctx, token); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated for a token older than the new row", err)
	}
}
