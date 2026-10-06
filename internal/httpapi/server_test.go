package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody))
	return rec
}

func handler(db Pinger) http.Handler {
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), db)
}

type envelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

func TestHealthOKWhenDatabaseAnswers(t *testing.T) {
	rec := get(t, handler(fakeDB{}), "/api/v1/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var body struct{ Status string }
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Status != "ok" {
		t.Fatalf("body: got %+v err %v", body, err)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("request id header missing")
	}
}

func TestHealthIsDatabaseUnavailableWhenPingFails(t *testing.T) {
	rec := get(t, handler(fakeDB{err: errors.New("down")}), "/api/v1/health")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d want 503", rec.Code)
	}
	var body envelope
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "database_unavailable" {
		t.Fatalf("code: got %q", body.Error.Code)
	}
	if body.Error.RequestID == "" || body.Error.RequestID != rec.Header().Get("X-Request-Id") {
		t.Fatalf("request_id %q does not match header %q", body.Error.RequestID, rec.Header().Get("X-Request-Id"))
	}
}

func TestUnknownAPIRouteUsesTheErrorEnvelope(t *testing.T) {
	rec := get(t, handler(fakeDB{}), "/api/v1/nowhere")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rec.Code)
	}
	var body envelope
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Error.Code != "not_found" {
		t.Fatalf("body: got %+v err %v", body, err)
	}
}
