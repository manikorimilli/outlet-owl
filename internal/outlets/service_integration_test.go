//go:build integration

package outlets_test

import (
	"context"
	"errors"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

type env struct {
	ctx   context.Context
	store *store.Store
	svc   *outlets.Service
	admin auth.User
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db := storetest.New(t)
	st := &store.Store{Pool: db.Pool}
	return &env{ctx: context.Background(), store: st, svc: outlets.NewService(st), admin: auth.User{ID: 1, Role: auth.RoleBrandAdmin}}
}

func (e *env) add(t *testing.T, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := e.svc.Create(e.ctx, e.admin, n); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
}

// syncUsers loads accounts the way the server does at start, so managers
// reference real outlets.
func (e *env) syncUsers(t *testing.T, entries ...auth.UsersFileEntry) {
	t.Helper()
	for i := range entries {
		entries[i].PasswordHash = "$2a$12$hash-for-outlet-tests"
	}
	if _, err := e.store.SyncUsers(e.ctx, entries); err != nil {
		t.Fatalf("SyncUsers: %v", err)
	}
}

func adminEntry() auth.UsersFileEntry {
	return auth.UsersFileEntry{Email: "a@example.in", Name: "Ritika Rao", Role: auth.RoleBrandAdmin}
}

func managerEntry(email, name, outlet string) auth.UsersFileEntry {
	return auth.UsersFileEntry{Email: email, Name: name, Role: auth.RoleOutletManager, Outlet: &outlet}
}

func (e *env) managerUser(t *testing.T, email string) auth.User {
	t.Helper()
	rec, err := e.store.UserByEmail(e.ctx, email)
	if err != nil {
		t.Fatalf("UserByEmail: %v", err)
	}
	return rec.User
}

func names(list []outlets.Summary) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Name
	}
	return out
}

func TestCreateOutlet_UniqueIgnoringCase(t *testing.T) {
	e := newEnv(t)
	e.add(t, "Koramangala")

	_, err := e.svc.Create(e.ctx, e.admin, "  koramangala ")

	var taken *outlets.NameTakenError
	if !errors.As(err, &taken) || taken.Existing != "Koramangala" {
		t.Fatalf("err = %v, want NameTakenError with the stored name Koramangala", err)
	}
	list, err := e.svc.List(e.ctx, e.admin)
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %v, %v; want one outlet, no second row", names(list), err)
	}
}

func TestCreateThenListOutlets_NewOutletAppears(t *testing.T) {
	e := newEnv(t)

	created, err := e.svc.Create(e.ctx, e.admin, "Electronic City")
	if err != nil || created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("Create = %+v, %v", created, err)
	}
	list, err := e.svc.List(e.ctx, e.admin)
	if err != nil || len(list) != 1 || list[0].ID != created.ID || list[0].Name != "Electronic City" {
		t.Fatalf("List = %+v, %v; want the new outlet", list, err)
	}
}

func TestListOutlets_AdminSeesAllOrderedByName(t *testing.T) {
	e := newEnv(t)
	e.add(t, "whitefield", "Indiranagar", "HSR Layout", "Koramangala")

	list, err := e.svc.List(e.ctx, e.admin)

	want := []string{"HSR Layout", "Indiranagar", "Koramangala", "whitefield"}
	if err != nil || len(list) != len(want) {
		t.Fatalf("List = %v, %v", names(list), err)
	}
	for i, n := range want {
		if list[i].Name != n {
			t.Fatalf("order = %v, want %v (by name ignoring case)", names(list), want)
		}
	}
}

func TestListOutlets_ManagerSeesOnlyOwnOutlet(t *testing.T) {
	e := newEnv(t)
	e.add(t, "Koramangala", "Indiranagar", "Whitefield")
	e.syncUsers(t, adminEntry(), managerEntry("m@example.in", "Neha Kulkarni", "Indiranagar"))

	list, err := e.svc.List(e.ctx, e.managerUser(t, "m@example.in"))

	if err != nil || len(list) != 1 || list[0].Name != "Indiranagar" {
		t.Fatalf("manager List = %v, %v; want only Indiranagar", names(list), err)
	}
}

func TestListOutlets_ManagersExcludeRemovedUsers(t *testing.T) {
	e := newEnv(t)
	e.add(t, "Koramangala")
	e.syncUsers(t, adminEntry(),
		managerEntry("m1@example.in", "Arjun Mehta", "Koramangala"),
		managerEntry("m2@example.in", "Bina Shah", "Koramangala"))
	e.syncUsers(t, adminEntry(), managerEntry("m2@example.in", "Bina Shah", "Koramangala"))

	list, err := e.svc.List(e.ctx, e.admin)

	if err != nil || len(list) != 1 || len(list[0].Managers) != 1 || list[0].Managers[0].Name != "Bina Shah" {
		t.Fatalf("List = %+v, %v; want only the active manager Bina Shah", list, err)
	}
}

func TestListOutlets_NewOutletHasNoManagers(t *testing.T) {
	e := newEnv(t)
	e.add(t, "Koramangala", "Electronic City")
	e.syncUsers(t, adminEntry(), managerEntry("m@example.in", "Arjun Mehta", "Koramangala"))

	list, err := e.svc.List(e.ctx, e.admin)

	if err != nil || len(list) != 2 {
		t.Fatalf("List = %v, %v", names(list), err)
	}
	if list[0].Name != "Electronic City" || len(list[0].Managers) != 0 {
		t.Fatalf("Electronic City = %+v, want no managers (the S-06 partial state)", list[0])
	}
	if len(list[1].Managers) != 1 {
		t.Fatalf("Koramangala = %+v, want its manager", list[1])
	}
}
