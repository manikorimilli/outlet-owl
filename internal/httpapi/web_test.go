package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWeb_ServesFilesAndFallsBackToIndex(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<div id=root></div>"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "assets", "app-1.js"), []byte("console.log(1)"), 0o600)
	h := New(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: fakeDB{}, Auth: newFakeAuth(), WebDir: dir, Brand: testBrand})

	cases := []struct{ path, want string }{
		{"/", "<div id=root>"},
		{"/reviews/7", "<div id=root>"},
		{"/assets/app-1.js", "console.log"},
		{"/etc/passwd", "<div id=root>"}, // outside dir: never served, index instead
	}
	for _, tc := range cases {
		rec := do(h, request{method: http.MethodGet, path: tc.path})
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s: %d %q", tc.path, rec.Code, rec.Body.String())
		}
	}
	if rec := do(h, request{method: http.MethodGet, path: "/assets/app-OLD.js"}); rec.Code != 404 {
		t.Errorf("stale asset = %d, want 404", rec.Code)
	}
	if rec := do(h, request{method: http.MethodGet, path: "/api/v1/nope"}); rec.Code != 404 || !strings.Contains(rec.Body.String(), "not_found") {
		t.Errorf("unknown API route = %d %s", rec.Code, rec.Body)
	}
}
