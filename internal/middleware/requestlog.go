package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// logFields carries what inner handlers learn for the request log line.
type logFields struct{ userID int64 }

type logFieldsKey struct{}

// SetUserID records the signed-in user's id for this request's log line.
// Outside RequestLog it does nothing.
func SetUserID(ctx context.Context, id int64) {
	if f, ok := ctx.Value(logFieldsKey{}).(*logFields); ok {
		f.userID = id
	}
}

// RequestLog writes one structured log line per request: route, status,
// duration, request id and, once signed in, the user id (HLD section 10). It
// never logs bodies, because review text and names are personal data.
func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			fields := &logFields{}
			r = r.WithContext(context.WithValue(r.Context(), logFieldsKey{}, fields))
			sw := &StatusWriter{ResponseWriter: w, Status: http.StatusOK}
			next.ServeHTTP(sw, r)
			attrs := []any{
				"method", r.Method, "route", r.Pattern, "status", sw.Status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFrom(r.Context()),
			}
			if fields.userID != 0 {
				attrs = append(attrs, "user_id", fields.userID)
			}
			logger.Info("request", attrs...)
		})
	}
}

// StatusWriter remembers the status a handler wrote so middleware can log it.
// Handlers that never call WriteHeader wrote 200.
type StatusWriter struct {
	http.ResponseWriter
	Status int
}

// WriteHeader records the status, then writes it.
func (s *StatusWriter) WriteHeader(code int) {
	s.Status = code
	s.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *StatusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
