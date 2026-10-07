package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeUsers is an in-memory UserStore. It holds only active accounts, as the
// real queries return only those; err, when set, is returned by every call.
type fakeUsers struct {
	byEmail map[string]UserRecord
	byID    map[int64]UserRecord
	err     error
}

func newFakeUsers(recs ...UserRecord) *fakeUsers {
	f := &fakeUsers{byEmail: map[string]UserRecord{}, byID: map[int64]UserRecord{}}
	for _, r := range recs {
		f.byEmail[r.Email] = r
		f.byID[r.ID] = r
	}
	return f
}

func (f *fakeUsers) UserByEmail(_ context.Context, email string) (UserRecord, error) {
	if f.err != nil {
		return UserRecord{}, f.err
	}
	r, ok := f.byEmail[email]
	if !ok {
		return UserRecord{}, ErrUserNotFound
	}
	return r, nil
}

func (f *fakeUsers) UserByID(_ context.Context, id int64) (UserRecord, error) {
	if f.err != nil {
		return UserRecord{}, f.err
	}
	r, ok := f.byID[id]
	if !ok {
		return UserRecord{}, ErrUserNotFound
	}
	return r, nil
}

func newService(t *testing.T, c *clock, recs ...UserRecord) (*Service, *fakeUsers) {
	t.Helper()
	users := newFakeUsers(recs...)
	return NewService(users, newTestTokens(t, c)), users
}

func managerRecord(t *testing.T, password string) UserRecord {
	t.Helper()
	return UserRecord{
		User: User{
			ID: 2, Email: "arjun.mehta@example.in", Name: "Arjun Mehta", Role: RoleOutletManager,
			Outlet: &OutletRef{ID: 1, Name: "Koramangala"}, CreatedAt: issuedAt.Add(-time.Hour),
		},
		PasswordHash: fastHash(t, password),
	}
}

func TestLogin_Succeeds(t *testing.T) {
	c := &clock{t: issuedAt}
	svc, _ := newService(t, c, managerRecord(t, "correct-horse"))

	u, token, err := svc.Login(context.Background(), "arjun.mehta@example.in", "correct-horse")

	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if u.ID != 2 || u.Outlet == nil || u.Outlet.Name != "Koramangala" {
		t.Fatalf("user = %+v", u)
	}
	got, err := svc.Authenticate(context.Background(), token)
	if err != nil || got.ID != 2 {
		t.Fatalf("the issued token does not authenticate: %+v, %v", got, err)
	}
}

func TestLogin_WrongPasswordIsInvalidCredentials(t *testing.T) {
	svc, _ := newService(t, &clock{t: issuedAt}, managerRecord(t, "correct-horse"))

	_, token, err := svc.Login(context.Background(), "arjun.mehta@example.in", "wrong-horse")

	if !errors.Is(err, ErrInvalidCredentials) || token != "" {
		t.Fatalf("err = %v, token %q; want ErrInvalidCredentials and no token", err, token)
	}
}

func TestLogin_UnknownAndRemovedEmailGetSameError(t *testing.T) {
	// A removed account is not returned by the store, exactly like an
	// unknown one, so both take the dummy-check path.
	svc, _ := newService(t, &clock{t: issuedAt})

	_, _, err := svc.Login(context.Background(), "nobody@example.in", "anything")

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_StoreFailureIsNotInvalidCredentials(t *testing.T) {
	svc, users := newService(t, &clock{t: issuedAt})
	users.err = errors.New("connection refused")

	_, _, err := svc.Login(context.Background(), "arjun.mehta@example.in", "x")

	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want a wrapped store error, not invalid credentials", err)
	}
}

func TestAuthenticate(t *testing.T) {
	rec := managerRecord(t, "pw")
	c := &clock{t: issuedAt}
	svc, users := newService(t, c, rec)
	valid, err := svc.tokens.Issue(rec.ID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	unknown, err := svc.tokens.Issue(99)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	t.Run("valid token returns the current row", func(t *testing.T) {
		u, err := svc.Authenticate(context.Background(), valid)
		if err != nil || u.Outlet == nil || u.Outlet.ID != 1 {
			t.Fatalf("Authenticate = %+v, %v", u, err)
		}
	})
	t.Run("invalid token", func(t *testing.T) {
		if _, err := svc.Authenticate(context.Background(), "nope"); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, want ErrUnauthenticated", err)
		}
	})
	t.Run("user no longer active", func(t *testing.T) {
		if _, err := svc.Authenticate(context.Background(), unknown); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, want ErrUnauthenticated", err)
		}
	})
	t.Run("token issued before the row was created", func(t *testing.T) {
		newer := rec
		newer.CreatedAt = issuedAt.Add(time.Minute)
		users.byID[rec.ID] = newer
		defer func() { users.byID[rec.ID] = rec }()
		if _, err := svc.Authenticate(context.Background(), valid); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, want ErrUnauthenticated", err)
		}
	})
	t.Run("row created within the token's second is accepted", func(t *testing.T) {
		same := rec
		same.CreatedAt = issuedAt.Add(400 * time.Millisecond)
		users.byID[rec.ID] = same
		defer func() { users.byID[rec.ID] = rec }()
		if _, err := svc.Authenticate(context.Background(), valid); err != nil {
			t.Fatalf("err = %v, want accepted (iat has whole seconds)", err)
		}
	})
	t.Run("store failure is not unauthenticated", func(t *testing.T) {
		users.err = errors.New("connection refused")
		defer func() { users.err = nil }()
		if _, err := svc.Authenticate(context.Background(), valid); err == nil || errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("err = %v, want a wrapped store error", err)
		}
	})
}

func TestUserScope(t *testing.T) {
	cases := []struct {
		name string
		user User
		want Scope
	}{
		{name: "brand admin sees all", user: User{Role: RoleBrandAdmin}, want: Scope{All: true}},
		{name: "manager sees their outlet", user: User{Role: RoleOutletManager, Outlet: &OutletRef{ID: 7}}, want: Scope{OutletID: 7}},
		{name: "manager without outlet sees nothing", user: User{Role: RoleOutletManager}, want: Scope{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.user.Scope(); got != tc.want {
				t.Fatalf("Scope = %+v, want %+v", got, tc.want)
			}
		})
	}
}
