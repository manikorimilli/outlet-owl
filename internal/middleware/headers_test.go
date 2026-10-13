package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	cases := []struct {
		name string
		path string
		hsts bool
		csp  string
	}{
		{"api", "/api/v1/me", false, apiCSP},
		{"page", "/reviews/7", false, pageCSP},
		{"asset", "/assets/index.js", true, pageCSP},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := SecurityHeaders(tc.hsts)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, tc.path, http.NoBody))

			want := map[string]string{
				"X-Content-Type-Options":  "nosniff",
				"X-Frame-Options":         "DENY",
				"Referrer-Policy":         "strict-origin-when-cross-origin",
				"Content-Security-Policy": tc.csp,
			}
			for name, value := range want {
				if got := rec.Header().Get(name); got != value {
					t.Errorf("%s = %q, want %q", name, got, value)
				}
			}
			hsts := rec.Header().Get("Strict-Transport-Security")
			if tc.hsts != (hsts != "") {
				t.Errorf("Strict-Transport-Security = %q with hsts %v", hsts, tc.hsts)
			}
		})
	}
}
