package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

// maxJSONBody bounds every JSON request body (phase 1 server LLD, section 3).
const maxJSONBody = 64 << 10

// errUnsupportedMediaType means the body is not application/json (415).
var errUnsupportedMediaType = errors.New("the body is not application/json")

// malformedError means the body could not be read as the expected JSON (400).
// Its text names a field or a position, never a value from the body.
type malformedError struct{ reason string }

func (e *malformedError) Error() string { return e.reason }

// decodeJSON reads one JSON object from the request into dst: the content
// type must be application/json, the body at most 64 KiB, every field known
// and nothing after the object. It is the one place request bodies are read.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errUnsupportedMediaType
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &malformedError{reason: describeDecodeError(err)}
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return &malformedError{reason: describeDecodeError(err)}
		}
		return &malformedError{reason: "the body has data after the JSON object"}
	}
	return nil
}

func describeDecodeError(err error) string {
	var tooLarge *http.MaxBytesError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &tooLarge):
		return "the body is over 64 KiB"
	case errors.Is(err, io.EOF):
		return "the body is empty"
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return "the body is not valid JSON"
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return "the body must be a JSON object"
		}
		return fmt.Sprintf("field %s has the wrong type", typeErr.Field)
	default:
		// encoding/json reports an unknown field only as text.
		if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return "unknown field " + field
		}
		return "the body could not be read"
	}
}
