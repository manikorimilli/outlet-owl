package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/digest"
	"github.com/manikorimilli/outlet-owl/internal/imports"
	"github.com/manikorimilli/outlet-owl/internal/middleware"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/replies"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
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
	var badName *outlets.ValidationError
	var taken *outlets.NameTakenError
	var fileErr *connector.FileError
	var badFilter *reviews.ValidationError
	var conflict *replies.ConflictError
	var badReply *replies.ValidationError
	var bodyTooLarge *http.MaxBytesError
	switch {
	case errors.Is(err, errUnsupportedMediaType):
		WriteError(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send application/json.")
	case errors.Is(err, errNotMultipart):
		WriteError(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the file as multipart/form-data.")
	case errors.Is(err, connector.ErrFileTooLarge), errors.As(err, &bodyTooLarge):
		WriteError(w, r, http.StatusRequestEntityTooLarge, "file_too_large", "The CSV file is over 5 MB. Split it into smaller files and import each.")
	case errors.As(err, &fileErr):
		details := make([]Detail, len(fileErr.Problems))
		for i, p := range fileErr.Problems {
			details[i] = Detail{Field: p.Field, Reason: p.Reason}
		}
		WriteErrorDetails(w, r, http.StatusBadRequest, "csv_invalid", fileErr.Message, details)
	case errors.Is(err, digest.ErrRoleNotAllowed):
		WriteError(w, r, http.StatusForbidden, "role_not_allowed", "Only the brand admin can generate the digest.")
	case errors.Is(err, replies.ErrRoleNotAllowed):
		WriteError(w, r, http.StatusForbidden, "role_not_allowed", "Only this outlet's manager can draft, edit or mark this reply.")
	case errors.Is(err, replies.ErrNotFound):
		WriteError(w, r, http.StatusNotFound, "not_found", "No such review among the ones you can see.")
	case errors.Is(err, replies.ErrBudgetExhausted):
		WriteError(w, r, http.StatusServiceUnavailable, "budget_exhausted", "Drafting is unavailable. The model budget is used up. Write the reply yourself.")
	case errors.Is(err, replies.ErrModelUnavailable):
		WriteError(w, r, http.StatusServiceUnavailable, "model_unavailable", "Drafting is unavailable right now. Write the reply yourself or try again later.")
	case errors.As(err, &conflict):
		WriteError(w, r, http.StatusConflict, conflict.Code, conflictMessages[conflict.Code])
	case errors.As(err, &badReply):
		msg := "Write the reply before saving."
		if badReply.Reason == "too_long" {
			msg = "Use at most 5,000 characters for the reply."
		}
		WriteErrorDetails(w, r, http.StatusUnprocessableEntity, "validation_failed", msg,
			[]Detail{{Field: "reply_text", Reason: badReply.Reason}})
	case errors.Is(err, reviews.ErrNotFound):
		WriteError(w, r, http.StatusNotFound, "not_found", "No such outlet among the ones you can see.")
	case errors.As(err, &badFilter):
		msg := "That theme is not in the theme list."
		if badFilter.Field == "q" {
			msg = "Use at most 200 characters in the search."
		}
		WriteErrorDetails(w, r, http.StatusUnprocessableEntity, "validation_failed", msg,
			[]Detail{{Field: badFilter.Field, Reason: badFilter.Reason}})
	case errors.Is(err, imports.ErrRoleNotAllowed):
		WriteError(w, r, http.StatusForbidden, "role_not_allowed", "Only the brand admin can import reviews.")
	case errors.As(err, &malformed):
		WriteError(w, r, http.StatusBadRequest, "malformed_request", "The request could not be read: "+malformed.reason+".")
	case errors.As(err, &invalid):
		WriteErrorDetails(w, r, http.StatusUnprocessableEntity, "validation_failed", invalid.message, invalid.details)
	case errors.Is(err, auth.ErrInvalidCredentials):
		WriteError(w, r, http.StatusUnauthorized, "invalid_credentials", "That email and password do not match an account. Check both and try again.")
	case errors.Is(err, auth.ErrUnauthenticated):
		WriteError(w, r, http.StatusUnauthorized, "unauthorized", "Sign in again.")
	case errors.Is(err, outlets.ErrRoleNotAllowed):
		WriteError(w, r, http.StatusForbidden, "role_not_allowed", "Only the brand admin can add outlets.")
	case errors.As(err, &badName):
		WriteErrorDetails(w, r, http.StatusUnprocessableEntity, "validation_failed", outletNameMessage(badName.Reason),
			[]Detail{{Field: badName.Field, Reason: badName.Reason}})
	case errors.As(err, &taken):
		// details[0].reason carries the stored name (api/openapi.yaml 1.1.1).
		WriteErrorDetails(w, r, http.StatusConflict, "outlet_name_taken",
			"An outlet named "+taken.Existing+" already exists. Names are matched without regard to capitals, so use a different name.",
			[]Detail{{Field: "name", Reason: taken.Existing}})
	default:
		logger.ErrorContext(r.Context(), "request failed", "err", err,
			"method", r.Method, "path", r.URL.Path,
			"request_id", middleware.RequestIDFrom(r.Context()))
		WriteError(w, r, http.StatusInternalServerError, "internal", internalMessage)
	}
}

func outletNameMessage(reason string) string {
	if reason == "too_long" {
		return "Use at most 200 characters for the outlet name."
	}
	return "Enter the outlet name."
}

var conflictMessages = map[string]string{
	"reply_changed":     "This reply changed since you opened it. Reload to see the newer text.",
	"draft_in_progress": "The model is still drafting this reply. Wait a moment and reload.",
	"already_replied":   "This reply is approved and its text can no longer change.",
}
