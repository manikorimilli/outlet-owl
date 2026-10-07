package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/middleware"
)

// Detail is one entry of the error envelope's details: the field and the
// rule it broke (api/openapi.yaml, Error).
type Detail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// validationError is a request that parsed but breaks a rule (422).
type validationError struct {
	message string
	details []Detail
}

func (e *validationError) Error() string { return e.message }

// internalMessage is the fixed 500 text; no internal detail reaches a client.
const internalMessage = "Something failed on our side. Quote the request id."

// WriteErrorDetails writes the error envelope with details. Every error
// response goes through it or WriteError.
func WriteErrorDetails(w http.ResponseWriter, r *http.Request, status int, code, message string, details []Detail) {
	body := struct {
		Code      string   `json:"code"`
		Message   string   `json:"message"`
		Details   []Detail `json:"details,omitempty"`
		RequestID string   `json:"request_id"`
	}{Code: code, Message: message, Details: details, RequestID: middleware.RequestIDFrom(r.Context())}
	WriteJSON(w, status, map[string]any{"error": body})
}

// writeDomainError maps an error from decoding or a service to its status and
// code, the one place this mapping happens (phase 1 server LLD, section 6).
// An error it does not know is logged with the request id and becomes a 500.
func writeDomainError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	var malformed *malformedError
	var invalid *validationError
	switch {
	case errors.Is(err, errUnsupportedMediaType):
		WriteError(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send application/json.")
	case errors.As(err, &malformed):
		WriteError(w, r, http.StatusBadRequest, "malformed_request", "The request could not be read: "+malformed.reason+".")
	case errors.As(err, &invalid):
		WriteErrorDetails(w, r, http.StatusUnprocessableEntity, "validation_failed", invalid.message, invalid.details)
	case errors.Is(err, auth.ErrInvalidCredentials):
		WriteError(w, r, http.StatusUnauthorized, "invalid_credentials", "That email and password do not match an account. Check both and try again.")
	case errors.Is(err, auth.ErrUnauthenticated):
		WriteError(w, r, http.StatusUnauthorized, "unauthorized", "Sign in again.")
	default:
		logger.ErrorContext(r.Context(), "request failed", "err", err,
			"method", r.Method, "path", r.URL.Path,
			"request_id", middleware.RequestIDFrom(r.Context()))
		WriteError(w, r, http.StatusInternalServerError, "internal", internalMessage)
	}
}
