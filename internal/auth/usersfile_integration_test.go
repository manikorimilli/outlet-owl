//go:build integration

package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

// Every character chk_users_email_shape refuses inside an email must make the
// parser skip the entry; otherwise the entry reaches SyncUsers and the server
// does not start. The constraint is read from the database, so the test
// follows any later migration of it.
func TestParseUsersFile_RefusesEveryCharacterTheEmailCheckRefuses(t *testing.T) {
	db := storetest.New(t)
	ctx := context.Background()

	var check string
	if err := db.Pool.QueryRow(ctx, `
		SELECT pg_get_expr(conbin, conrelid) FROM pg_constraint
		WHERE conname = 'chk_users_email_shape'`).Scan(&check); err != nil {
		t.Fatalf("read the email check: %v", err)
	}
	// Each code point (no NUL, no surrogates: chr refuses both) inside the
	// local part of an otherwise valid email, kept where the check fails.
	rows, err := db.Pool.Query(ctx, fmt.Sprintf(`
		SELECT cp FROM (
			SELECT cp, 'a' || chr(cp) || 'b@example.in' AS email
			FROM generate_series(1, 1114111) AS cp
			WHERE cp < 55296 OR cp > 57343
		) AS candidates
		WHERE NOT %s ORDER BY cp`, check))
	if err != nil {
		t.Fatalf("list refused characters: %v", err)
	}
	var refused []rune
	for rows.Next() {
		var cp int32
		if err := rows.Scan(&cp); err != nil {
			t.Fatalf("scan: %v", err)
		}
		refused = append(refused, cp)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list refused characters: %v", err)
	}
	if len(refused) < 7 { // '@', \t, \n, \v, \f, \r and the space at least
		t.Fatalf("the database refuses only %U; the query did not run the check", refused)
	}

	for _, r := range refused {
		email := "a" + string(r) + "b@example.in"
		got, err := parse(t, fileOf(t, admin("a@example.in"), manager(email, "Koramangala")))
		if err != nil {
			t.Fatalf("%U: ParseUsers: %v", r, err)
		}
		if len(got.Skipped) != 1 || got.Skipped[0].Field != "email" {
			t.Errorf("%U: the database refuses it in an email, but the parser kept the entry", r)
		}
	}
}
