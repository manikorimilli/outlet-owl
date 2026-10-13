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

// crossOriginCase is one request through CrossOrigin: the method, the Host it
// arrives with, and the browser headers (empty means absent).
type crossOriginCase struct {
	name      string
	method    string
	host      string
	fetchSite string
	origin    string
	allowed   bool
}

func serveCrossOrigin(t *testing.T, logger *slog.Logger, tc crossOriginCase) (handlerRan bool, rec *httptest.ResponseRecorder, writtenCode string) {
	t.Helper()
	writeErr := func(w http.ResponseWriter, _ *http.Request, status int, code, _ string) {
		writtenCode = code
		w.WriteHeader(status)
	}
	h := CrossOrigin(logger, writeErr)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerRan = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(context.Background(), tc.method, "http://"+tc.host+"/api/v1/outlets", http.NoBody)
	req.Host = tc.host
	if tc.fetchSite != "" {
		req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
	}
	if tc.origin != "" {
		req.Header.Set("Origin", tc.origin)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return handlerRan, rec, writtenCode
}

func TestCrossOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cases := []crossOriginCase{
		{name: "SameOriginPostAllowed", method: http.MethodPost, host: "localhost:8080", fetchSite: "same-origin", origin: "http://localhost:8080", allowed: true},
		{name: "ViteProxyRequestAllowed", method: http.MethodPost, host: "localhost:5173", fetchSite: "same-origin", origin: "http://localhost:5173", allowed: true},
		{name: "ViteProxyWithoutFetchSiteAllowed", method: http.MethodPost, host: "localhost:5173", origin: "http://localhost:5173", allowed: true},
		{name: "UserInitiatedNoneAllowed", method: http.MethodPost, host: "localhost:8080", fetchSite: "none", allowed: true},
		{name: "OtherLocalPortRefused", method: http.MethodPost, host: "localhost:8080", fetchSite: "same-site", origin: "http://localhost:3000", allowed: false},
		{name: "CrossSiteRefused", method: http.MethodPost, host: "localhost:8080", fetchSite: "cross-site", origin: "https://evil.example", allowed: false},
		{name: "CrossSitePutRefused", method: http.MethodPut, host: "localhost:8080", fetchSite: "cross-site", origin: "https://evil.example", allowed: false},
		{name: "CrossSiteDeleteRefused", method: http.MethodDelete, host: "localhost:8080", fetchSite: "cross-site", origin: "https://evil.example", allowed: false},
		{name: "MissingFetchSiteMismatchedOriginRefused", method: http.MethodPost, host: "localhost:8080", origin: "http://localhost:3000", allowed: false},
		{name: "NoHeadersAllowed", method: http.MethodPost, host: "localhost:8080", allowed: true},
		{name: "SafeMethodGetNeverRefused", method: http.MethodGet, host: "localhost:8080", fetchSite: "cross-site", origin: "https://evil.example", allowed: true},
		{name: "SafeMethodHeadNeverRefused", method: http.MethodHead, host: "localhost:8080", fetchSite: "cross-site", origin: "https://evil.example", allowed: true},
		{name: "SafeMethodOptionsNeverRefused", method: http.MethodOptions, host: "localhost:8080", fetchSite: "cross-site", origin: "https://evil.example", allowed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran, rec, code := serveCrossOrigin(t, logger, tc)

			if tc.allowed {
				if !ran || rec.Code != http.StatusNoContent {
					t.Fatalf("want allowed: handler ran %v, status %d, code %q", ran, rec.Code, code)
				}
				return
			}
			if ran {
				t.Fatal("the handler ran for a refused request")
			}
			if rec.Code != http.StatusForbidden || code != "cross_site_request" {
				t.Fatalf("want 403 cross_site_request, got %d %q", rec.Code, code)
			}
		})
	}
}

func TestCrossOrigin_RefusalIsLoggedWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	writeErr := func(w http.ResponseWriter, _ *http.Request, status int, _, _ string) { w.WriteHeader(status) }
	h := RequestID(CrossOrigin(logger, writeErr)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/api/v1/outlets", http.NoBody)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set(HeaderRequestID, "req-cross-1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	for _, want := range []string{`"msg":"cross-origin write refused"`, `"request_id":"req-cross-1"`, `"method":"POST"`} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("log lacks %s: %s", want, buf.String())
		}
	}
}
