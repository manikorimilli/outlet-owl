package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLog writes one structured log line per request: route, status,
// duration and request id (HLD section 10). It never logs bodies, because
// review text and names are personal data.
func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &StatusWriter{ResponseWriter: w, Status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("request",
				"method", r.Method, "route", r.Pattern, "status", sw.Status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFrom(r.Context()))
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
