package httpapi

import (
	"context"
	"net/http"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/middleware"
)

// sessionCookie is the cookie api/openapi.yaml names (cookieAuth).
const sessionCookie = "outletowl_session"

// Authenticator signs people in and resolves the caller of a request.
// *auth.Service satisfies it.
type Authenticator interface {
	Login(ctx context.Context, email, password string) (auth.User, string, error)
	Authenticate(ctx context.Context, token string) (auth.User, error)
}

// setSessionCookie sets the session cookie for SessionTTL: HttpOnly so no
// script reads it, SameSite=Strict so other sites never send it (ADR-0007).
// No Secure flag: the product runs on http://localhost (PRD section 6).
func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.SessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// clearSessionCookie tells the browser to drop the session cookie
// (Max-Age=0). A copied token stays valid until it expires (ADR-0007).
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// requireUser runs next only for a signed-in caller, with the user re-read
// from the database on this request (tenet 3); otherwise 401 unauthorized.
// It records the user id for the request log.
func requireUser(d Deps, next func(w http.ResponseWriter, r *http.Request, user auth.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || cookie.Value == "" {
			WriteError(w, r, http.StatusUnauthorized, "unauthorized", "Sign in again.")
			return
		}
		user, err := d.Auth.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		middleware.SetUserID(r.Context(), user.ID)
		next(w, r, user)
	}
}
