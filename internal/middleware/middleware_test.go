package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDEchoesOrMints(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = RequestIDFrom(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set(HeaderRequestID, "abc-123")
	h.ServeHTTP(rec, req)
	if seen != "abc-123" || rec.Header().Get(HeaderRequestID) != "abc-123" {
		t.Fatalf("echo: got ctx %q header %q", seen, rec.Header().Get(HeaderRequestID))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody))
	if got := rec.Header().Get(HeaderRequestID); len(got) != 32 {
		t.Fatalf("minted id: got %q", got)
	}
}

func TestRecoverWrites500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	called := false
	writeErr := func(w http.ResponseWriter, _ *http.Request, status int, _, _ string) {
		called = true
		w.WriteHeader(status)
	}
	h := Recover(logger, writeErr)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody))

	if !called || rec.Code != http.StatusInternalServerError {
		t.Fatalf("got called=%v status=%d", called, rec.Code)
	}
}

func TestRequestLogRecordsStatusWithoutBody(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := RequestLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("secret review text"))
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", http.NoBody))

	if !strings.Contains(buf.String(), `"status":418`) {
		t.Fatalf("log lacks the status: %s", buf.String())
	}
	if strings.Contains(buf.String(), "secret review text") {
		t.Fatalf("log contains the response body: %s", buf.String())
	}
	if strings.Contains(buf.String(), "user_id") {
		t.Fatalf("a request with no signed-in user logs a user_id: %s", buf.String())
	}
}

func TestRequestLog_IncludesUserID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := RequestLog(logger)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		SetUserID(r.Context(), 42)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", http.NoBody))

	if !strings.Contains(buf.String(), `"user_id":42`) {
		t.Fatalf("log lacks the user id: %s", buf.String())
	}
}

func TestSetUserID_OutsideRequestLogDoesNothing(t *testing.T) {
	SetUserID(context.Background(), 42) // must not panic
}
