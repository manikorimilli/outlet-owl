//go:build integration

package storetest

import (
	"context"
	"testing"
)

// TestNew_SeparateDatabasesPerCall proves two callers (two test packages
// running in parallel) never share a database, and that each is migrated.
func TestNew_SeparateDatabasesPerCall(t *testing.T) {
	a := New(t)
	b := New(t)
	if a.Name == b.Name {
		t.Fatalf("both calls got database %q", a.Name)
	}
	for _, db := range []*DB{a, b} {
		var current string
		var hasUsers bool
		err := db.Pool.QueryRow(context.Background(),
			"SELECT current_database(), to_regclass('public.users') IS NOT NULL").Scan(&current, &hasUsers)
		if err != nil {
			t.Fatalf("query %s: %v", db.Name, err)
		}
		if current != db.Name {
			t.Fatalf("pool is on %q, want %q", current, db.Name)
		}
		if !hasUsers {
			t.Fatalf("%s is not migrated: no users table", db.Name)
		}
	}
}
