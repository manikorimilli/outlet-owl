package httpapi

import (
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/middleware"
)

// Brand is the one brand per installation (Q-001), from configuration.
type Brand struct {
	Name     string
	Timezone string
}

// maxLoginFieldRunes is the spec's maxLength for email and password.
const maxLoginFieldRunes = 200

// loginEmailShape matches chk_users_email_shape, so a malformed email is a
// 422 and never reaches the database.
var loginEmailShape = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)

// loginRequest is LoginRequest in api/openapi.yaml.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// meResponse is Me in api/openapi.yaml.
type meResponse struct {
	ID     int64         `json:"id"`
	Email  string        `json:"email"`
	Name   string        `json:"name"`
	Role   auth.Role     `json:"role"`
	Outlet *outletRef    `json:"outlet"`
	Brand  brandResponse `json:"brand"`
}

type outletRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type brandResponse struct {
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

func newMe(u auth.User, b Brand) meResponse {
	me := meResponse{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, Brand: brandResponse(b)}
	if u.Outlet != nil {
		me.Outlet = &outletRef{ID: u.Outlet.ID, Name: u.Outlet.Name}
	}
	return me
}

// login answers POST /api/v1/auth/login (operationId login).
func login(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if err := validateLogin(req); err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		user, token, err := d.Auth.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		middleware.SetUserID(r.Context(), user.ID)
		setSessionCookie(w, token)
		WriteJSON(w, http.StatusOK, newMe(user, d.Brand))
	}
}

// validateLogin applies the LoginRequest rules; the password is not trimmed.
func validateLogin(req loginRequest) error {
	var details []Detail
	switch {
	case req.Email == "":
		details = append(details, Detail{Field: "email", Reason: "blank"})
	case utf8.RuneCountInString(req.Email) > maxLoginFieldRunes:
		details = append(details, Detail{Field: "email", Reason: "too_long"})
	case !loginEmailShape.MatchString(req.Email):
		details = append(details, Detail{Field: "email", Reason: "invalid_email"})
	}
	switch {
	case req.Password == "":
		details = append(details, Detail{Field: "password", Reason: "blank"})
	case utf8.RuneCountInString(req.Password) > maxLoginFieldRunes:
		details = append(details, Detail{Field: "password", Reason: "too_long"})
	}
	if len(details) == 0 {
		return nil
	}
	return &validationError{message: "Check the email and password fields.", details: details}
}

// logout answers POST /api/v1/auth/logout (operationId logout): it clears the
// cookie whether or not a session exists, so it is safe to repeat.
func logout() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		clearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	}
}

// me answers GET /api/v1/me (operationId getMe).
func me(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, _ *http.Request, user auth.User) {
		WriteJSON(w, http.StatusOK, newMe(user, d.Brand))
	})
}
