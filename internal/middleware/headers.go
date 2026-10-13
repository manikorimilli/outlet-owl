package middleware

import (
	"net/http"
	"strings"
)

// apiCSP is for JSON: it lets the response load nothing and be framed by
// nothing.
const apiCSP = "default-src 'none'; frame-ancestors 'none'"

// pageCSP is for the built web app when this server serves it: its own
// scripts, styles, fonts and images only. 'unsafe-inline' for styles covers
// React's style attributes (the chart tooltips); no inline script is allowed.
const pageCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; " +
	"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// SecurityHeaders sets the browser protections every response carries
// (threat model T-17): no MIME sniffing, no framing, a strict referrer and a
// content security policy, the API's or the page's. hsts adds
// Strict-Transport-Security, which only makes sense when the app is reached
// over HTTPS (COOKIE_SECURE, ADR-0010).
func SecurityHeaders(hsts bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			if strings.HasPrefix(r.URL.Path, "/api/") {
				h.Set("Content-Security-Policy", apiCSP)
			} else {
				h.Set("Content-Security-Policy", pageCSP)
			}
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=31536000")
			}
			next.ServeHTTP(w, r)
		})
	}
}
