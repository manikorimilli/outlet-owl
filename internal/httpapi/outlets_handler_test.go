package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
)

// fakeOutlets records whether the service was reached and answers from a
// fixed list; create returns the configured result.
type fakeOutlets struct {
	list      []outlets.Summary
	createErr error
	created   outlets.Summary
	calls     int
}

func (f *fakeOutlets) List(_ context.Context, _ auth.User) ([]outlets.Summary, error) {
	f.calls++
	return f.list, nil
}

func (f *fakeOutlets) Create(_ context.Context, user auth.User, _ string) (outlets.Summary, error) {
	f.calls++
	if err := outlets.CheckCanCreate(user); err != nil {
		return outlets.Summary{}, err
	}
	return f.created, f.createErr
}

func outletsHandler(o *fakeOutlets) http.Handler {
	return New(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: fakeDB{}, Auth: newFakeAuth(), Outlets: o, Brand: testBrand})
}

var adminCookie = "token-for-" + testAdmin.Email
var managerCookie = "token-for-" + testManager.Email

func createRequest(cookie, body string) request {
	return request{method: http.MethodPost, path: "/api/v1/outlets", contentType: "application/json", body: body, cookie: cookie}
}

func TestCreateOutletHandler_AdminGets201WithLocation(t *testing.T) {
	o := &fakeOutlets{created: outlets.Summary{Outlet: outlets.Outlet{ID: 6, Name: "Electronic City"}}}

	rec := do(outletsHandler(o), createRequest(adminCookie, `{"name":"Electronic City"}`))

	if rec.Code != http.StatusCreated || rec.Header().Get("Location") != "/api/v1/outlets/6" {
		t.Fatalf("got %d Location %q, want 201 /api/v1/outlets/6", rec.Code, rec.Header().Get("Location"))
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for key, want := range map[string]string{"id": "6", "name": `"Electronic City"`, "managers": "[]", "review_count": "0", "untagged_count": "0"} {
		if string(raw[key]) != want {
			t.Fatalf("%s = %s, want %s", key, raw[key], want)
		}
	}
	if _, ok := raw["created_at"]; !ok {
		t.Fatal("OutletSummary lacks created_at")
	}
}

func TestCreateOutletHandler_ManagerGets403BeforeBodyIsRead(t *testing.T) {
	o := &fakeOutlets{}
	// The body is not even JSON: a manager must still get 403, not 400 or 415.
	rq := createRequest(managerCookie, "not json")
	rq.contentType = "text/plain"

	rec := do(outletsHandler(o), rq)

	if body := decodeEnvelope(t, rec); rec.Code != http.StatusForbidden || body.Error.Code != "role_not_allowed" {
		t.Fatalf("got %d %q, want 403 role_not_allowed", rec.Code, body.Error.Code)
	}
	if o.calls != 0 {
		t.Fatal("the outlet service was called for a manager")
	}
}

func TestCreateOutletHandler_DuplicateGets409WithExistingName(t *testing.T) {
	o := &fakeOutlets{createErr: &outlets.NameTakenError{Existing: "Koramangala"}}

	rec := do(outletsHandler(o), createRequest(adminCookie, `{"name":"koramangala"}`))

	body := decodeEnvelope(t, rec)
	if rec.Code != http.StatusConflict || body.Error.Code != "outlet_name_taken" {
		t.Fatalf("got %d %q, want 409 outlet_name_taken", rec.Code, body.Error.Code)
	}
	if len(body.Error.Details) != 1 || body.Error.Details[0] != (Detail{Field: "name", Reason: "Koramangala"}) {
		t.Fatalf("details = %+v, want name Koramangala", body.Error.Details)
	}
}

func TestCreateOutletHandler_Errors(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
		createErr   error
		status      int
		code        string
	}{
		{name: "TextPlainReturns415", contentType: "text/plain", body: `{"name":"X"}`, status: 415, code: "unsupported_media_type"},
		{name: "unknown field", contentType: "application/json", body: `{"name":"X","id":3}`, status: 400, code: "malformed_request"},
		{name: "blank name", contentType: "application/json", body: `{"name":"  "}`, createErr: &outlets.ValidationError{Field: "name", Reason: "blank"}, status: 422, code: "validation_failed"},
		{name: "store failure", contentType: "application/json", body: `{"name":"X"}`, createErr: errors.New("connection refused"), status: 500, code: "internal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rq := createRequest(adminCookie, tc.body)
			rq.contentType = tc.contentType

			rec := do(outletsHandler(&fakeOutlets{createErr: tc.createErr}), rq)

			if body := decodeEnvelope(t, rec); rec.Code != tc.status || body.Error.Code != tc.code {
				t.Fatalf("got %d %q, want %d %s", rec.Code, body.Error.Code, tc.status, tc.code)
			}
		})
	}
}

func TestOutletsHandlers_NeedASession(t *testing.T) {
	for _, rq := range []request{
		{method: http.MethodGet, path: "/api/v1/outlets"},
		createRequest("", `{"name":"X"}`),
	} {
		rec := do(outletsHandler(&fakeOutlets{}), rq)
		if body := decodeEnvelope(t, rec); rec.Code != http.StatusUnauthorized || body.Error.Code != "unauthorized" {
			t.Fatalf("%s %s: got %d %q, want 401 unauthorized", rq.method, rq.path, rec.Code, body.Error.Code)
		}
	}
}

func TestListOutletsHandler_ReturnsData(t *testing.T) {
	o := &fakeOutlets{list: []outlets.Summary{
		{Outlet: outlets.Outlet{ID: 2, Name: "Indiranagar"}, Managers: []outlets.Manager{{ID: 3, Name: "Neha Kulkarni", OutletID: 2}}},
		{Outlet: outlets.Outlet{ID: 6, Name: "Electronic City"}},
	}}

	rec := do(outletsHandler(o), request{method: http.MethodGet, path: "/api/v1/outlets", cookie: adminCookie})

	var body struct {
		Data []outletSummary `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("status %d, err %v", rec.Code, err)
	}
	if len(body.Data) != 2 || body.Data[0].Managers[0].Name != "Neha Kulkarni" || body.Data[1].Managers == nil {
		t.Fatalf("data = %+v, want two outlets, the second with an empty managers list", body.Data)
	}
}
