//go:build integration

package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

// The database does not check bcrypt costs, so any non-empty hash serves.
const syncHash = "$2a$12$hash-for-sync-tests-only"

func newSyncStore(t *testing.T, outlets ...string) (*Store, context.Context) {
	t.Helper()
	db := storetest.New(t)
	ctx := context.Background()
	for _, name := range outlets {
		if _, err := db.Pool.Exec(ctx, "INSERT INTO outlets (name) VALUES ($1)", name); err != nil {
			t.Fatalf("insert outlet %s: %v", name, err)
		}
	}
	return &Store{Pool: db.Pool}, ctx
}

func adminEntry(email string) auth.UsersFileEntry {
	return auth.UsersFileEntry{Entry: 1, Email: email, Name: "Ritika Rao", Role: auth.RoleBrandAdmin, PasswordHash: syncHash}
}

func managerEntry(email, outlet string) auth.UsersFileEntry {
	return auth.UsersFileEntry{Entry: 2, Email: email, Name: "Arjun Mehta", Role: auth.RoleOutletManager, Outlet: &outlet, PasswordHash: syncHash}
}

type userRow struct {
	email     string
	name      string
	role      string
	outlet    *string
	removed   bool
	updatedAt time.Time
}

// usersByEmail reads every account, keyed by lower-case email.
func usersByEmail(ctx context.Context, t *testing.T, s *Store) map[string]userRow {
	t.Helper()
	rows, err := s.Pool.Query(ctx, `
		SELECT u.email, u.name, u.role::text, o.name, u.removed_at IS NOT NULL, u.updated_at
		FROM users u LEFT JOIN outlets o ON o.id = u.outlet_id`)
	if err != nil {
		t.Fatalf("read users: %v", err)
	}
	defer rows.Close()
	out := map[string]userRow{}
	for rows.Next() {
		var r userRow
		if err := rows.Scan(&r.email, &r.name, &r.role, &r.outlet, &r.removed, &r.updatedAt); err != nil {
			t.Fatalf("scan user: %v", err)
		}
		out[strings.ToLower(r.email)] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read users: %v", err)
	}
	return out
}

func mustSync(ctx context.Context, t *testing.T, s *Store, entries ...auth.UsersFileEntry) SyncUsersResult {
	t.Helper()
	res, err := s.SyncUsers(ctx, entries)
	if err != nil {
		t.Fatalf("SyncUsers: %v", err)
	}
	return res
}

func TestSyncUsers_InsertsThenUpdatesChangedFields(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala", "Indiranagar")

	res := mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "koramangala"))
	if res.Changed != 2 || res.Removed != 0 || len(res.Skipped) != 0 {
		t.Fatalf("first sync = %+v, want 2 changed", res)
	}
	m := managerEntry("m@example.in", "Indiranagar")
	m.Name = "Arjun M."
	res = mustSync(ctx, t, s, adminEntry("a@example.in"), m)
	if res.Changed != 1 {
		t.Fatalf("second sync changed %d, want 1 (the manager)", res.Changed)
	}
	got := usersByEmail(ctx, t, s)["m@example.in"]
	if got.name != "Arjun M." || got.outlet == nil || *got.outlet != "Indiranagar" || got.role != "outlet_manager" {
		t.Fatalf("manager row = %+v, want the new name and outlet", got)
	}
}

func TestSyncUsers_RepeatChangesNothing(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	entries := []auth.UsersFileEntry{adminEntry("a@example.in"), managerEntry("m@example.in", "Koramangala")}
	mustSync(ctx, t, s, entries...)
	before := usersByEmail(ctx, t, s)

	res := mustSync(ctx, t, s, entries...)

	if res.Changed != 0 || res.Removed != 0 {
		t.Fatalf("repeat = %+v, want nothing changed", res)
	}
	for email, row := range usersByEmail(ctx, t, s) {
		if !row.updatedAt.Equal(before[email].updatedAt) {
			t.Fatalf("%s updated_at moved on a repeat", email)
		}
	}
}

func TestSyncUsers_MixedCaseEmailIsNotRemovedAndRestored(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	entries := []auth.UsersFileEntry{adminEntry("Ritika.Rao@Example.in"), managerEntry("Arjun@Example.in", "Koramangala")}
	mustSync(ctx, t, s, entries...)
	before := usersByEmail(ctx, t, s)

	res := mustSync(ctx, t, s, entries...)

	if res.Changed != 0 || res.Removed != 0 {
		t.Fatalf("repeat with mixed-case emails = %+v, want nothing changed", res)
	}
	if got := usersByEmail(ctx, t, s)["arjun@example.in"]; got.removed || !got.updatedAt.Equal(before["arjun@example.in"].updatedAt) {
		t.Fatalf("mixed-case account was touched: %+v", got)
	}
}

func TestSyncUsers_MarksMissingEmailRemoved(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "Koramangala"))

	res := mustSync(ctx, t, s, adminEntry("a@example.in"))

	if res.Removed != 1 {
		t.Fatalf("removed = %d, want 1", res.Removed)
	}
	if !usersByEmail(ctx, t, s)["m@example.in"].removed {
		t.Fatal("the manager left the file but is not marked removed")
	}
}

func TestSyncUsers_ReturningEmailIsRestored(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "Koramangala"))
	mustSync(ctx, t, s, adminEntry("a@example.in"))

	mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "Koramangala"))

	if usersByEmail(ctx, t, s)["m@example.in"].removed {
		t.Fatal("the returning manager is still marked removed")
	}
}

func TestSyncUsers_UnknownOutletSkipsEntryAndRemovesAccount(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "Koramangala"))

	res := mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "Whitefield"))

	if len(res.Skipped) != 1 || res.Skipped[0].Entry != 2 || res.Skipped[0].Field != "outlet" {
		t.Fatalf("skipped = %+v, want entry 2 for its outlet", res.Skipped)
	}
	if !usersByEmail(ctx, t, s)["m@example.in"].removed {
		t.Fatal("an account whose entry was skipped can still sign in")
	}
}

func TestSyncUsers_AdminEmailChangeKeepsOneActiveAdmin(t *testing.T) {
	s, ctx := newSyncStore(t)
	mustSync(ctx, t, s, adminEntry("old@example.in"))

	mustSync(ctx, t, s, adminEntry("new@example.in"))

	assertOneActiveAdmin(ctx, t, s, "new@example.in")
}

func TestSyncUsers_AdminAndManagerSwapRoles(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("b@example.in", "Koramangala"))

	// The admin entry comes first in the file; SyncUsers still writes it last.
	mustSync(ctx, t, s, adminEntry("b@example.in"), managerEntry("a@example.in", "Koramangala"))

	assertOneActiveAdmin(ctx, t, s, "b@example.in")
	if got := usersByEmail(ctx, t, s)["a@example.in"]; got.role != "outlet_manager" || got.removed {
		t.Fatalf("a@ = %+v, want an active manager", got)
	}
}

func TestSyncUsers_RollsBackOnFailure(t *testing.T) {
	s, ctx := newSyncStore(t, "Koramangala")
	mustSync(ctx, t, s, adminEntry("a@example.in"), managerEntry("m@example.in", "Koramangala"))
	before := usersByEmail(ctx, t, s)

	// A blank name passes no parser but reaches the database here, where
	// chk_users_name_not_blank refuses it after the removal step has run.
	bad := adminEntry("a@example.in")
	bad.Name = " "
	if _, err := s.SyncUsers(ctx, []auth.UsersFileEntry{bad}); err == nil {
		t.Fatal("want an error from the name check")
	}

	after := usersByEmail(ctx, t, s)
	if after["m@example.in"].removed || !after["m@example.in"].updatedAt.Equal(before["m@example.in"].updatedAt) {
		t.Fatalf("the failed sync left a change behind: %+v", after["m@example.in"])
	}
}

func assertOneActiveAdmin(ctx context.Context, t *testing.T, s *Store, want string) {
	t.Helper()
	var active []string
	for email, row := range usersByEmail(ctx, t, s) {
		if row.role == "brand_admin" && !row.removed {
			active = append(active, email)
		}
	}
	if len(active) != 1 || active[0] != want {
		t.Fatalf("active admins = %v, want only %s", active, want)
	}
}
