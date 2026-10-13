package middleware

import (
	"log/slog"
	"net/http"
)

// crossSiteMessage is the message api/openapi.yaml gives for
// cross_site_request.
const crossSiteMessage = "This request came from another site and was refused. Open OutletOwl at its own address and try again."

// CrossOrigin refuses a write (any method but GET, HEAD and OPTIONS) that the
// browser marks as coming from another origin, with 403 cross_site_request,
// before any handler runs (phase 1 server LLD, section 4.7). It uses
// net/http.CrossOriginProtection, with no trusted origins unless the
// deployment names some (TRUSTED_ORIGINS, ADR-0010):
//
//   - Sec-Fetch-Site same-origin or none: allowed;
//   - Sec-Fetch-Site same-site or cross-site: refused, which covers another
//     port on localhost that SameSite=Strict would let through;
//   - no Sec-Fetch-Site: refused when Origin's host differs from Host;
//   - neither header (curl, tests): allowed.
//
// The UI is always same-origin: the Go binary serves it, the Vite proxy
// forwards the browser's Sec-Fetch-Site, Origin and Host unchanged, and on
// the demo Vercel forwards /api to this server. A trusted origin covers a
// browser that sends no Sec-Fetch-Site through that proxy, whose Origin
// (Vercel) then differs from Host (this server). config.Load has already
// checked each origin; one that still fails is logged and left out.
func CrossOrigin(logger *slog.Logger, writeError func(w http.ResponseWriter, r *http.Request, status int, code, message string), trusted ...string) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range trusted {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			logger.Error("trusted origin left out", "origin", origin, "err", err.Error())
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := protection.Check(r); err != nil {
				logger.InfoContext(r.Context(), "cross-origin write refused",
					"reason", err.Error(),
					"method", r.Method,
					"path", r.URL.Path,
					"origin", r.Header.Get("Origin"),
					"request_id", RequestIDFrom(r.Context()))
				writeError(w, r, http.StatusForbidden, "cross_site_request", crossSiteMessage)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
