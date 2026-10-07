package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The budget set has no Down: no back-out may drop the running total
// (HLD section 4, REQ-031). A Down section added to any file would let
// goose roll the total away.
func TestBudgetMigrations_HaveNoDown(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "db", "migrations", "budget", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no budget migrations found")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "+goose Down") {
			t.Errorf("%s has a Down section; the budget set must never roll back", f)
		}
	}
}
