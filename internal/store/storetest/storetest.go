//go:build integration

package storetest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is one per-test database, migrated to the latest version.
type DB struct {
	Name string
	URL  string
	Pool *pgxpool.Pool
}

// New creates a database named outlet_owl_test_<pid>_<random>, applies every
// migration in db/migrations and the budget set with the goose binary, and drops the database
// when the test ends. DATABASE_URL names the server and a database to connect
// to for CREATE DATABASE; nothing is written there.
func New(t testing.TB) *DB {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Fatal("storetest: DATABASE_URL is not set (make test-integration sets it; make db starts PostgreSQL)")
	}
	name := "outlet_owl_test_" + fmt.Sprint(os.Getpid()) + "_" + randomHex(t, 4)
	testURL, err := withDatabase(base, name)
	if err != nil {
		t.Fatalf("storetest: %v", err)
	}

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("storetest: connect to DATABASE_URL: %v", err)
	}
	ident := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+ident); err != nil {
		_ = admin.Close(ctx) // the create failed; the close error adds nothing
		t.Fatalf("storetest: create database %s: %v", name, err)
	}
	_ = admin.Close(ctx) // a failed close leaks one local connection at worst

	t.Cleanup(func() { dropDatabase(t, base, ident) })

	db := &DB{Name: name, URL: testURL}
	db.Goose(t, "up")
	db.gooseBudget(t, "up")

	pool, err := pgxpool.New(ctx, testURL)
	if err != nil {
		t.Fatalf("storetest: open pool on %s: %v", name, err)
	}
	t.Cleanup(pool.Close) // registered last, so it runs before the drop

	var current string
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&current); err != nil {
		t.Fatalf("storetest: read current_database: %v", err)
	}
	if err := CheckNotDemo(current); err != nil {
		t.Fatal(err)
	}
	db.Pool = pool
	return db
}

// Goose runs the goose binary against this database with the repository's
// migrations, for example db.Goose(t, "down-to", "0").
func (d *DB) Goose(t testing.TB, args ...string) {
	t.Helper()
	if _, err := exec.LookPath("goose"); err != nil {
		t.Fatal("storetest: goose is not on PATH; install v3.28.0 (AGENTS.md, Toolchain) or run through make test-integration")
	}
	cmdArgs := append([]string{"-dir", migrationsDir(t), "postgres", d.URL}, args...)
	out, err := exec.Command("goose", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("storetest: goose %v on %s: %v\n%s", args, d.Name, err, out)
	}
}

// gooseBudget runs goose on the budget set, which has its own version table
// and no Down (HLD section 4); tests only ever apply it.
func (d *DB) gooseBudget(t testing.TB, args ...string) {
	t.Helper()
	dir := filepath.Join(migrationsDir(t), "budget")
	cmdArgs := append([]string{"-dir", dir, "-table", "goose_budget_version", "postgres", d.URL}, args...)
	out, err := exec.Command("goose", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("storetest: goose budget %v on %s: %v\n%s", args, d.Name, err, out)
	}
}

// migrationsDir finds db/migrations from this file's own path, so it works
// whichever package directory go test runs in.
func migrationsDir(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("storetest: cannot locate the storetest source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "migrations")
}

// withDatabase returns the URL with its database path replaced.
func withDatabase(base, name string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", fmt.Errorf("DATABASE_URL must be a postgres:// URL")
	}
	u.Path = "/" + name
	return u.String(), nil
}

func dropDatabase(t testing.TB, base, ident string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Errorf("storetest: connect to drop %s: %v", ident, err)
		return
	}
	defer func() { _ = admin.Close(ctx) }() // cleanup path; nothing left to report to
	if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+ident+" WITH (FORCE)"); err != nil {
		t.Errorf("storetest: drop database %s: %v", ident, err)
	}
}

func randomHex(t testing.TB, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("storetest: random name: %v", err)
	}
	return hex.EncodeToString(b)
}
