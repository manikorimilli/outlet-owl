//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

// TestSignIn_EndToEnd runs the real service, store and PostgreSQL behind the
// handler: sign in, read /me with the cookie, then remove the account from
// the users file and see the same cookie refused (AC-US-00-001-1, ADR-0007).
func TestSignIn_EndToEnd(t *testing.T) {
	db := storetest.New(t)
	ctx := context.Background()
	if _, err := db.Pool.Exec(ctx, "INSERT INTO outlets (name) VALUES ('Koramangala')"); err != nil {
		t.Fatalf("insert outlet: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-horse"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	outlet := "Koramangala"
	admin := auth.UsersFileEntry{Email: "ritika.rao@example.in", Name: "Ritika Rao", Role: auth.RoleBrandAdmin, PasswordHash: string(hash)}
	manager := auth.UsersFileEntry{Email: "arjun.mehta@example.in", Name: "Arjun Mehta", Role: auth.RoleOutletManager, Outlet: &outlet, PasswordHash: string(hash)}
	st := &store.Store{Pool: db.Pool}
	if _, err := st.SyncUsers(ctx, []auth.UsersFileEntry{admin, manager}); err != nil {
		t.Fatalf("SyncUsers: %v", err)
	}
	tokens, err := auth.NewTokens([]byte("0123456789abcdef0123456789abcdef"), nil)
	if err != nil {
		t.Fatalf("NewTokens: %v", err)
	}
	h := New(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: st, Auth: auth.NewService(st, tokens), Brand: testBrand})

	if rec := do(h, loginRequestOf("ARJUN.MEHTA@example.in", "wrong")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: status %d, want 401", rec.Code)
	}
	rec := do(h, loginRequestOf("ARJUN.MEHTA@example.in", "correct-horse"))
	if rec.Code != http.StatusOK || len(rec.Result().Cookies()) != 1 {
		t.Fatalf("sign in: status %d, cookies %v", rec.Code, rec.Result().Cookies())
	}
	cookie := rec.Result().Cookies()[0].Value

	rec = do(h, request{method: http.MethodGet, path: "/api/v1/me", cookie: cookie})
	var me meResponse
	if err := json.NewDecoder(rec.Body).Decode(&me); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("/me: status %d, err %v", rec.Code, err)
	}
	if me.Role != auth.RoleOutletManager || me.Outlet == nil || me.Outlet.Name != "Koramangala" {
		t.Fatalf("/me = %+v, want the manager of Koramangala", me)
	}

	if _, err := st.SyncUsers(ctx, []auth.UsersFileEntry{admin}); err != nil {
		t.Fatalf("SyncUsers: %v", err)
	}
	if rec := do(h, request{method: http.MethodGet, path: "/api/v1/me", cookie: cookie}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("/me after removal: status %d, want 401", rec.Code)
	}
}
