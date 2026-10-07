package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// fakeAuth signs in one password and maps tokens to users. err, when set, is
// what both methods return (a store failure).
type fakeAuth struct {
	password string
	users    map[string]auth.User // by email for Login, by token for Authenticate
	err      error
}

func (f *fakeAuth) Login(_ context.Context, email, password string) (auth.User, string, error) {
	if f.err != nil {
		return auth.User{}, "", f.err
	}
	u, ok := f.users[email]
	if !ok || password != f.password {
		return auth.User{}, "", auth.ErrInvalidCredentials
	}
	return u, "token-for-" + email, nil
}

func (f *fakeAuth) Authenticate(_ context.Context, token string) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	u, ok := f.users[token]
	if !ok {
		return auth.User{}, fmt.Errorf("%w: unknown token", auth.ErrUnauthenticated)
	}
	return u, nil
}

var (
	testManager = auth.User{ID: 2, Email: "arjun.mehta@example.in", Name: "Arjun Mehta", Role: auth.RoleOutletManager, Outlet: &auth.OutletRef{ID: 1, Name: "Koramangala"}}
	testAdmin   = auth.User{ID: 1, Email: "ritika.rao@example.in", Name: "Ritika Rao", Role: auth.RoleBrandAdmin}
	testBrand   = Brand{Name: "Neem Tree Kitchens", Timezone: "Asia/Kolkata"}
)

func newFakeAuth() *fakeAuth {
	return &fakeAuth{password: "correct-horse", users: map[string]auth.User{
		testManager.Email:                testManager,
		testAdmin.Email:                  testAdmin,
		"token-for-" + testManager.Email: testManager,
		"token-for-" + testAdmin.Email:   testAdmin,
	}}
}

func authHandler(a *fakeAuth, logs io.Writer) http.Handler {
	if logs == nil {
		logs = io.Discard
	}
	return New(Deps{Logger: slog.New(slog.NewJSONHandler(logs, nil)), DB: fakeDB{}, Auth: a, Brand: testBrand})
}

type request struct {
	method      string
	path        string
	contentType string
	body        string
	cookie      string
	fetchSite   string
}

func do(h http.Handler, rq request) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), rq.method, "http://localhost:8080"+rq.path, strings.NewReader(rq.body))
	if rq.contentType != "" {
		req.Header.Set("Content-Type", rq.contentType)
	}
	if rq.cookie != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: rq.cookie})
	}
	if rq.fetchSite != "" {
		req.Header.Set("Sec-Fetch-Site", rq.fetchSite)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func loginRequestOf(email, password string) request {
	b, _ := json.Marshal(map[string]string{"email": email, "password": password}) // two strings always marshal
	return request{method: http.MethodPost, path: "/api/v1/auth/login", contentType: "application/json", body: string(b)}
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) fullEnvelope {
	t.Helper()
	var body fullEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode envelope: %v (body %q)", err, rec.Body.String())
	}
	return body
}

func TestLoginHandler_SetsHttpOnlyStrictCookie(t *testing.T) {
	rec := do(authHandler(newFakeAuth(), nil), loginRequestOf("arjun.mehta@example.in", "correct-horse"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v, want one", cookies)
	}
	c := cookies[0]
	if c.Name != sessionCookie || c.Value != "token-for-arjun.mehta@example.in" || !c.HttpOnly ||
		c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.MaxAge != 28800 || c.Secure {
		t.Fatalf("cookie = %+v, want outletowl_session HttpOnly SameSite=Strict Path=/ Max-Age=28800", c)
	}
	var me meResponse
	if err := json.NewDecoder(rec.Body).Decode(&me); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if me.ID != 2 || me.Outlet == nil || me.Outlet.Name != "Koramangala" || me.Brand.Name != testBrand.Name {
		t.Fatalf("me = %+v", me)
	}
}

func TestLoginHandler_TrimsEmail(t *testing.T) {
	rec := do(authHandler(newFakeAuth(), nil), loginRequestOf("  arjun.mehta@example.in ", "correct-horse"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 for a padded email", rec.Code)
	}
}

func TestLoginHandler_WrongPasswordReturns401InvalidCredentials(t *testing.T) {
	rec := do(authHandler(newFakeAuth(), nil), loginRequestOf("arjun.mehta@example.in", "wrong"))

	body := decodeEnvelope(t, rec)
	if rec.Code != http.StatusUnauthorized || body.Error.Code != "invalid_credentials" {
		t.Fatalf("got %d %q, want 401 invalid_credentials", rec.Code, body.Error.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("a refused sign-in set a cookie")
	}
}

func TestLoginHandler_ValidationReturns422(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		password string
		want     []Detail
	}{
		{name: "BlankEmailReturns422", email: "  ", password: "pw", want: []Detail{{Field: "email", Reason: "blank"}}},
		{name: "email not shaped like an email", email: "arjun", password: "pw", want: []Detail{{Field: "email", Reason: "invalid_email"}}},
		{name: "email over 200 characters", email: strings.Repeat("a", 190) + "@example.in", password: "pw", want: []Detail{{Field: "email", Reason: "too_long"}}},
		{name: "blank password", email: "a@example.in", password: "", want: []Detail{{Field: "password", Reason: "blank"}}},
		{name: "password over 200 characters", email: "a@example.in", password: strings.Repeat("p", 201), want: []Detail{{Field: "password", Reason: "too_long"}}},
		{name: "both blank", email: "", password: "", want: []Detail{{Field: "email", Reason: "blank"}, {Field: "password", Reason: "blank"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(authHandler(newFakeAuth(), nil), loginRequestOf(tc.email, tc.password))

			body := decodeEnvelope(t, rec)
			if rec.Code != http.StatusUnprocessableEntity || body.Error.Code != "validation_failed" {
				t.Fatalf("got %d %q, want 422 validation_failed", rec.Code, body.Error.Code)
			}
			if fmt.Sprint(body.Error.Details) != fmt.Sprint(tc.want) {
				t.Fatalf("details = %v, want %v", body.Error.Details, tc.want)
			}
		})
	}
}

func TestLoginHandler_UnknownFieldReturns400(t *testing.T) {
	rq := loginRequestOf("a@example.in", "pw")
	rq.body = `{"email":"a@example.in","password":"pw","role":"brand_admin"}`

	rec := do(authHandler(newFakeAuth(), nil), rq)

	if body := decodeEnvelope(t, rec); rec.Code != http.StatusBadRequest || body.Error.Code != "malformed_request" {
		t.Fatalf("got %d %q, want 400 malformed_request", rec.Code, body.Error.Code)
	}
}

func TestLoginHandler_TextPlainReturns415(t *testing.T) {
	rq := loginRequestOf("a@example.in", "pw")
	rq.contentType = "text/plain"

	rec := do(authHandler(newFakeAuth(), nil), rq)

	if body := decodeEnvelope(t, rec); rec.Code != http.StatusUnsupportedMediaType || body.Error.Code != "unsupported_media_type" {
		t.Fatalf("got %d %q, want 415 unsupported_media_type", rec.Code, body.Error.Code)
	}
}

func TestLoginHandler_StoreFailureReturns500(t *testing.T) {
	a := newFakeAuth()
	a.err = errors.New("connection refused")

	rec := do(authHandler(a, nil), loginRequestOf("a@example.in", "pw"))

	if body := decodeEnvelope(t, rec); rec.Code != http.StatusInternalServerError || body.Error.Code != "internal" {
		t.Fatalf("got %d %q, want 500 internal", rec.Code, body.Error.Code)
	}
}

func TestLogoutHandler_ClearsCookieWithoutSession(t *testing.T) {
	rec := do(authHandler(newFakeAuth(), nil), request{method: http.MethodPost, path: "/api/v1/auth/logout"})

	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("got %d with body %q, want 204 and no body", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookie || cookies[0].MaxAge >= 0 || cookies[0].Value != "" {
		t.Fatalf("cookies = %+v, want outletowl_session cleared with Max-Age=0", cookies)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want Max-Age=0", rec.Header().Get("Set-Cookie"))
	}
}

func TestMeHandler_NoCookieReturns401(t *testing.T) {
	rec := do(authHandler(newFakeAuth(), nil), request{method: http.MethodGet, path: "/api/v1/me"})

	if body := decodeEnvelope(t, rec); rec.Code != http.StatusUnauthorized || body.Error.Code != "unauthorized" {
		t.Fatalf("got %d %q, want 401 unauthorized", rec.Code, body.Error.Code)
	}
}

func TestMeHandler_ExpiredCookieReturns401(t *testing.T) {
	// The fake refuses any token it did not issue, as Verify refuses an
	// expired one: both wrap ErrUnauthenticated.
	rec := do(authHandler(newFakeAuth(), nil), request{method: http.MethodGet, path: "/api/v1/me", cookie: "expired-token"})

	if body := decodeEnvelope(t, rec); rec.Code != http.StatusUnauthorized || body.Error.Code != "unauthorized" {
		t.Fatalf("got %d %q, want 401 unauthorized", rec.Code, body.Error.Code)
	}
}

func TestMeHandler_ReturnsBrandAndOutlet(t *testing.T) {
	cases := []struct {
		name       string
		token      string
		wantOutlet string // "null" for the brand admin
	}{
		{name: "outlet manager", token: "token-for-" + testManager.Email, wantOutlet: `{"id":1,"name":"Koramangala"}`},
		{name: "brand admin has outlet null", token: "token-for-" + testAdmin.Email, wantOutlet: "null"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(authHandler(newFakeAuth(), nil), request{method: http.MethodGet, path: "/api/v1/me", cookie: tc.token})

			if rec.Code != http.StatusOK {
				t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("decode: %v", err)
			}
			for _, key := range []string{"id", "email", "name", "role", "outlet", "brand"} {
				if _, ok := raw[key]; !ok {
					t.Fatalf("Me lacks %s: %s", key, rec.Body.String())
				}
			}
			if string(raw["outlet"]) != tc.wantOutlet {
				t.Fatalf("outlet = %s, want %s", raw["outlet"], tc.wantOutlet)
			}
			if string(raw["brand"]) != `{"name":"Neem Tree Kitchens","timezone":"Asia/Kolkata"}` {
				t.Fatalf("brand = %s", raw["brand"])
			}
		})
	}
}

func TestMeHandler_LogsUserID(t *testing.T) {
	var logs bytes.Buffer
	do(authHandler(newFakeAuth(), &logs), request{method: http.MethodGet, path: "/api/v1/me", cookie: "token-for-" + testManager.Email})

	if !strings.Contains(logs.String(), `"user_id":2`) || !strings.Contains(logs.String(), `"route":"GET /api/v1/me"`) {
		t.Fatalf("request log lacks the user id or route: %s", logs.String())
	}
}

func TestCrossOrigin_LoginAndLogoutAreProtected(t *testing.T) {
	h := authHandler(newFakeAuth(), nil)
	for _, rq := range []request{
		loginRequestOf("arjun.mehta@example.in", "correct-horse"),
		{method: http.MethodPost, path: "/api/v1/auth/logout"},
	} {
		rq.fetchSite = "cross-site"
		rec := do(h, rq)
		if body := decodeEnvelope(t, rec); rec.Code != http.StatusForbidden || body.Error.Code != "cross_site_request" {
			t.Fatalf("%s: got %d %q, want 403 cross_site_request", rq.path, rec.Code, body.Error.Code)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Fatalf("%s: a refused request changed the cookie", rq.path)
		}
	}
}
