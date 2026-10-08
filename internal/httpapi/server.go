// Package httpapi owns routing, the middleware chain and the JSON response
// helpers. Every route lives under /api/v1 and follows api/openapi.yaml
// (ADR-0005, tenet 7). Product routes arrive with their stories.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manikorimilli/outlet-owl/internal/middleware"
)

// Pinger reports whether the database answers. The store satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps is what the routes need. main builds it; tests fill it with fakes.
type Deps struct {
	Logger  *slog.Logger
	DB      Pinger
	Auth    Authenticator
	Outlets OutletService
	Imports ImportService
	Brand   Brand
}

// New builds the HTTP handler. The chain, outermost first: request id, panic
// recovery, request log, the cross-origin write check, then the mux. The
// check sits inside the request log so a refusal is logged with its 403.
func New(d Deps) http.Handler {
	logger := d.Logger
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", health(logger, d.DB))
	mux.HandleFunc("POST /api/v1/auth/login", login(d))
	mux.HandleFunc("POST /api/v1/auth/logout", logout())
	mux.HandleFunc("GET /api/v1/me", me(d))
	mux.HandleFunc("GET /api/v1/outlets", listOutlets(d))
	mux.HandleFunc("POST /api/v1/outlets", createOutlet(d))
	mux.HandleFunc("POST /api/v1/imports", createImport(d))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusNotFound, "not_found", "No such route.")
	})

	var handler http.Handler = mux
	handler = middleware.CrossOrigin(logger, WriteError)(handler)
	handler = middleware.RequestLog(logger)(handler)
	handler = middleware.Recover(logger, WriteError)(handler)
	return middleware.RequestID(handler)
}

// health answers GET /api/v1/health (operationId getHealth): 200 when
// PostgreSQL answers, 503 database_unavailable when it does not.
func health(logger *slog.Logger, db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			logger.WarnContext(r.Context(), "health: database ping failed", "err", err,
				"request_id", middleware.RequestIDFrom(r.Context()))
			WriteError(w, r, http.StatusServiceUnavailable, "database_unavailable", "The database is not reachable.")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// WriteJSON is the single way a handler writes a JSON body.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the client has gone if this fails; nothing to do
}

// WriteError writes the error envelope from api/openapi.yaml: a stable code,
// a message for people and the request id. Never leak internal detail here.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	WriteErrorDetails(w, r, status, code, message, nil)
}
