package httpapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/connector"
	"github.com/manikorimilli/outlet-owl/internal/imports"
)

// ImportService runs a CSV import. *imports.Service satisfies it.
type ImportService interface {
	Import(ctx context.Context, user auth.User, requestID, fileName string, src connector.Connector) (imports.Result, error)
}

// maxUploadBody bounds the whole multipart body: the 5 MB file plus room for
// the part headers. The file itself is capped by the connector.
const maxUploadBody = connector.MaxFileBytes + 64<<10

// maxFileNameRunes is the API's limit on file_name (ImportResult).
const maxFileNameRunes = 200

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// errNotMultipart means the import body is not multipart/form-data (415).
var errNotMultipart = errors.New("the body is not multipart/form-data")

// importRejection and importResult are ImportRejection and ImportResult in
// api/openapi.yaml.
type importRejection struct {
	RowNumber int    `json:"row_number"`
	Reason    string `json:"reason"`
}

type importResult struct {
	ID             int64             `json:"id"`
	FileName       string            `json:"file_name"`
	ImportedCount  int               `json:"imported_count"`
	DuplicateCount int               `json:"duplicate_count"`
	RejectedCount  int               `json:"rejected_count"`
	Rejections     []importRejection `json:"rejections"`
	CreatedAt      time.Time         `json:"created_at"`
}

// createImport answers POST /api/v1/imports (operationId createImport). The
// role is checked before the body is read; the file streams straight into
// the CSV connector, and a repeat of the Idempotency-Key returns the first
// result without reading it.
func createImport(d Deps) http.HandlerFunc {
	return requireUser(d, func(w http.ResponseWriter, r *http.Request, user auth.User) {
		if err := imports.CheckCanImport(user); err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if !uuidPattern.MatchString(key) {
			writeDomainError(w, r, d.Logger, &malformedError{reason: "the Idempotency-Key header must be a UUID"})
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			writeDomainError(w, r, d.Logger, errNotMultipart)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)
		mr, err := r.MultipartReader()
		if err != nil {
			writeDomainError(w, r, d.Logger, &malformedError{reason: "the multipart body could not be read"})
			return
		}
		part, err := filePart(mr)
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		defer func() { _ = part.Close() }() // the body is discarded with the request

		res, err := d.Imports.Import(r.Context(), user, strings.ToLower(key), fileName(part.FileName()), connector.NewCSV(part))
		if err != nil {
			writeDomainError(w, r, d.Logger, err)
			return
		}
		out := importResult{
			ID: res.ID, FileName: res.FileName, CreatedAt: res.CreatedAt.UTC(),
			ImportedCount: res.ImportedCount, DuplicateCount: res.DuplicateCount, RejectedCount: res.RejectedCount,
			Rejections: make([]importRejection, len(res.Rejections)), // [] in JSON, never null
		}
		for i, rej := range res.Rejections {
			out.Rejections[i] = importRejection{RowNumber: rej.Number, Reason: rej.Reason}
		}
		WriteJSON(w, http.StatusCreated, out)
	})
}

// filePart returns the part named file, skipping any other field.
func filePart(mr *multipart.Reader) (*multipart.Part, error) {
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, &malformedError{reason: "the file field is missing"}
		}
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, connector.ErrFileTooLarge
		}
		if err != nil {
			return nil, &malformedError{reason: "the multipart body could not be read"}
		}
		if p.FormName() == "file" {
			return p, nil
		}
		_ = p.Close()
	}
}

// fileName keeps the base name the browser sent, at most 200 characters; a
// file sent with no name is called upload.csv.
func fileName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, `\`, "/")))
	if name == "" || name == "." || name == "/" || !utf8.ValidString(name) {
		return "upload.csv"
	}
	if utf8.RuneCountInString(name) > maxFileNameRunes {
		name = string([]rune(name)[:maxFileNameRunes])
	}
	return name
}
