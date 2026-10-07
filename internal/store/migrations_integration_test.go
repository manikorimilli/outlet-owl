//go:build integration

package store

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

// TestMigration00001_UpDownUp applies the migrations, takes 00001 down and
// applies it again, checking the tables, the enum and the four indexes each
// way (phase 1 server LLD, work item 1).
func TestMigration00001_UpDownUp(t *testing.T) {
	db := storetest.New(t)
	ctx := context.Background()

	wantIndexes := []string{"idx_users_outlet_id", "uq_outlets_name_lower", "uq_users_email_lower", "uq_users_one_brand_admin"}

	assertSchema(ctx, t, db.Pool, true, wantIndexes)
	db.Goose(t, "down-to", "0")
	assertSchema(ctx, t, db.Pool, false, nil)
	db.Goose(t, "up")
	assertSchema(ctx, t, db.Pool, true, wantIndexes)
}

func assertSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool, present bool, wantIndexes []string) {
	t.Helper()
	for _, table := range []string{"public.outlets", "public.users"} {
		var exists bool
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil {
			t.Fatalf("look up %s: %v", table, err)
		}
		if exists != present {
			t.Fatalf("table %s exists = %v, want %v", table, exists, present)
		}
	}

	var typeExists bool
	if err := pool.QueryRow(ctx, "SELECT to_regtype('public.user_role') IS NOT NULL").Scan(&typeExists); err != nil {
		t.Fatalf("look up user_role: %v", err)
	}
	if typeExists != present {
		t.Fatalf("type user_role exists = %v, want %v", typeExists, present)
	}

	rows, err := pool.Query(ctx,
		"SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename IN ('outlets', 'users') AND indexname NOT LIKE '%_pkey' ORDER BY indexname")
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan index name: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	if !slices.Equal(got, wantIndexes) {
		t.Fatalf("indexes = %v, want %v", got, wantIndexes)
	}
}
