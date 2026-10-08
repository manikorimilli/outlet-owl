package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/imports"
)

// importStore is the import store in memory: every valid row is new.
type importStore struct {
	saved map[string]imports.Result
}

func (s *importStore) ImportByRequestID(_ context.Context, key string) (imports.Result, bool, error) {
	r, ok := s.saved[key]
	return r, ok, nil
}

func (s *importStore) OutletIDsByName(context.Context) (map[string]int64, error) {
	return map[string]int64{"indiranagar": 1}, nil
}

func (s *importStore) SaveImport(_ context.Context, w imports.Write) (imports.Result, bool, error) {
	r := imports.Result{ID: 12, FileName: w.FileName, ImportedCount: len(w.Reviews), RejectedCount: len(w.Rejections),
		Rejections: w.Rejections, CreatedAt: time.Date(2026, 10, 6, 5, 10, 0, 0, time.UTC)}
	s.saved[w.RequestID] = r
	return r, true, nil
}

type noSignal struct{}

func (noSignal) Signal() {}

func importsHandler() http.Handler {
	svc := imports.NewService(&importStore{saved: map[string]imports.Result{}}, noSignal{})
	return New(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: fakeDB{}, Auth: newFakeAuth(), Imports: svc, Brand: testBrand})
}

const testKey = "0192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b"

// upload posts csv as the file field of a multipart body.
func upload(h http.Handler, cookie, key, fileName, csv string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("note", "ignored") // a field before the file is skipped
	fw, _ := mw.CreateFormFile("file", fileName)
	_, _ = io.WriteString(fw, csv)
	_ = mw.Close()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/api/v1/imports", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const goodCSV = "outlet,source,date,rating,text,reviewer_name\n" +
	"Indiranagar,Google,2026-09-14,4,Great,Asha\n" +
	"Koramangla,Google,2026-09-14,4,Great,Asha\n"

func TestCreateImport_Returns201WithTheResult(t *testing.T) {
	rec := upload(importsHandler(), adminCookie, testKey, `C:\exports\reviews-sept.csv`, goodCSV)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["file_name"] != "reviews-sept.csv" || got["imported_count"] != 1.0 || got["rejected_count"] != 1.0 ||
		got["duplicate_count"] != 0.0 || got["created_at"] != "2026-10-06T05:10:00Z" {
		t.Fatalf("body = %v", got)
	}
	rej := got["rejections"].([]any)[0].(map[string]any)
	if rej["row_number"] != 3.0 || !strings.HasPrefix(rej["reason"].(string), `Unknown outlet "Koramangla"`) {
		t.Fatalf("rejection = %v", rej)
	}
}

func TestCreateImport_Refusals(t *testing.T) {
	cases := []struct {
		name         string
		cookie, key  string
		csv          string
		status       int
		code, detail string
	}{
		{"Manager", managerCookie, testKey, goodCSV, http.StatusForbidden, "role_not_allowed", ""},
		{"NoKey", adminCookie, "", goodCSV, http.StatusBadRequest, "malformed_request", ""},
		{"KeyNotUUID", adminCookie, "abc", goodCSV, http.StatusBadRequest, "malformed_request", ""},
		{"MissingColumn", adminCookie, testKey, "outlet,source,date,text,reviewer_name\n", http.StatusBadRequest, "csv_invalid", "rating"},
		{"TooLarge", adminCookie, testKey, strings.Repeat("x", connector.MaxFileBytes+1), http.StatusRequestEntityTooLarge, "file_too_large", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := upload(importsHandler(), tc.cookie, tc.key, "f.csv", tc.csv)
			var env fullEnvelope
			_ = json.Unmarshal(rec.Body.Bytes(), &env)
			if rec.Code != tc.status || env.Error.Code != tc.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, env.Error.Code, tc.status, tc.code)
			}
			if tc.detail != "" && (len(env.Error.Details) == 0 || env.Error.Details[0].Field != tc.detail) {
				t.Fatalf("details = %+v, want %s first", env.Error.Details, tc.detail)
			}
		})
	}
}

// postImport sends body with the given content type, the test key and the admin
// cookie.
func postImport(contentType string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/api/v1/imports", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Idempotency-Key", testKey)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: adminCookie})
	rec := httptest.NewRecorder()
	importsHandler().ServeHTTP(rec, req)
	return rec
}

func TestCreateImport_NotMultipartIs415(t *testing.T) {
	if rec := postImport("text/csv", strings.NewReader(goodCSV)); rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status %d, want 415", rec.Code)
	}
}

func TestCreateImport_MissingFilePartIs400(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("other", "x")
	_ = mw.Close()
	if rec := postImport(mw.FormDataContentType(), &body); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

// A body cut off before its closing boundary is the client's fault.
func TestCreateImport_TruncatedBodyIs400(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "f.csv")
	_, _ = io.WriteString(fw, goodCSV)
	// no mw.Close(): the final boundary never arrives
	rec := postImport(mw.FormDataContentType(), &body)
	var env fullEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if rec.Code != http.StatusBadRequest || env.Error.Code != "malformed_request" {
		t.Fatalf("got %d %s, want 400 malformed_request", rec.Code, env.Error.Code)
	}
}
