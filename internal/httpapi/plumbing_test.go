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

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decodeRequest(contentType, body string) error {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	var dst loginBody
	return decodeJSON(httptest.NewRecorder(), req, &dst)
}

func TestDecodeJSON(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
		wantMedia   bool   // want errUnsupportedMediaType
		wantReason  string // want a malformedError containing this; "" means success
	}{
		{name: "valid", contentType: "application/json", body: `{"email":"a@example.in","password":"pw"}`},
		{name: "charset parameter accepted", contentType: "application/json; charset=utf-8", body: `{"email":"a@example.in"}`},
		{name: "RejectsWrongContentType", contentType: "text/plain", body: `{}`, wantMedia: true},
		{name: "missing content type", body: `{}`, wantMedia: true},
		{name: "form content type", contentType: "application/x-www-form-urlencoded", body: "email=a", wantMedia: true},
		{name: "RejectsUnknownField", contentType: "application/json", body: `{"email":"a","role":"brand_admin"}`, wantReason: `unknown field "role"`},
		{name: "RejectsOver64KiB", contentType: "application/json", body: `{"email":"` + strings.Repeat("a", maxJSONBody) + `"}`, wantReason: "over 64 KiB"},
		{name: "RejectsTrailingData", contentType: "application/json", body: `{"email":"a"} {"email":"b"}`, wantReason: "data after the JSON object"},
		{name: "empty body", contentType: "application/json", body: ``, wantReason: "empty"},
		{name: "invalid JSON", contentType: "application/json", body: `{"email":`, wantReason: "not valid JSON"},
		{name: "wrong field type", contentType: "application/json", body: `{"email":42}`, wantReason: "field email has the wrong type"},
		{name: "an array, not an object", contentType: "application/json", body: `[]`, wantReason: "must be a JSON object"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := decodeRequest(tc.contentType, tc.body)

			switch {
			case tc.wantMedia:
				if !errors.Is(err, errUnsupportedMediaType) {
					t.Fatalf("err = %v, want errUnsupportedMediaType", err)
				}
			case tc.wantReason != "":
				var m *malformedError
				if !errors.As(err, &m) || !strings.Contains(m.reason, tc.wantReason) {
					t.Fatalf("err = %v, want a malformedError mentioning %q", err, tc.wantReason)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

type fullEnvelope struct {
	Error struct {
		Code      string   `json:"code"`
		Message   string   `json:"message"`
		Details   []Detail `json:"details"`
		RequestID string   `json:"request_id"`
	} `json:"error"`
}

func mapError(t *testing.T, logger *slog.Logger, err error) (int, fullEnvelope) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/x", http.NoBody)
	writeDomainError(rec, req, logger, err)
	var body fullEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec.Code, body
}

func TestWriteDomainError_MapsEachDomainError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "unsupported media type", err: errUnsupportedMediaType, status: 415, code: "unsupported_media_type"},
		{name: "malformed", err: &malformedError{reason: "the body is empty"}, status: 400, code: "malformed_request"},
		{name: "validation", err: &validationError{message: "email must not be blank", details: []Detail{{Field: "email", Reason: "blank"}}}, status: 422, code: "validation_failed"},
		{name: "invalid credentials", err: auth.ErrInvalidCredentials, status: 401, code: "invalid_credentials"},
		{name: "unauthenticated, wrapped", err: fmt.Errorf("%w: expired", auth.ErrUnauthenticated), status: 401, code: "unauthorized"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := mapError(t, logger, tc.err)
			if status != tc.status || body.Error.Code != tc.code {
				t.Fatalf("got %d %q, want %d %q", status, body.Error.Code, tc.status, tc.code)
			}
		})
	}

	_, body := mapError(t, logger, &validationError{message: "m", details: []Detail{{Field: "email", Reason: "blank"}}})
	if len(body.Error.Details) != 1 || body.Error.Details[0] != (Detail{Field: "email", Reason: "blank"}) {
		t.Fatalf("details = %+v, want email blank", body.Error.Details)
	}
}

func TestWriteDomainError_UnknownErrorIs500WithoutDetail(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	status, body := mapError(t, logger, errors.New("pq: password authentication failed for user secret"))

	if status != 500 || body.Error.Code != "internal" || body.Error.Message != internalMessage {
		t.Fatalf("got %d %+v, want 500 internal with the fixed message", status, body.Error)
	}
	if strings.Contains(body.Error.Message, "secret") || body.Error.Details != nil {
		t.Fatalf("the response leaks the cause: %+v", body.Error)
	}
	if !strings.Contains(logs.String(), "password authentication failed") {
		t.Fatalf("the cause is not logged: %s", logs.String())
	}
}

func TestWriteError_OmitsEmptyDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody), 404, "not_found", "No such route.")
	if strings.Contains(rec.Body.String(), "details") {
		t.Fatalf("details present when empty: %s", rec.Body.String())
	}
}
