package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// Role is what an account may do (Q-002); the users table stores it as the
// user_role enum.
type Role string

// The two roles. The brand admin sees every outlet; an outlet manager sees
// and acts on one.
const (
	RoleBrandAdmin    Role = "brand_admin"
	RoleOutletManager Role = "outlet_manager"
)

// maxFieldRunes bounds email, name and outlet name, as the API does
// (api/openapi.yaml conventions).
const maxFieldRunes = 200

// maxUsersFileBytes bounds the file read at start; a handful of accounts is
// a few kilobytes.
const maxUsersFileBytes = 1 << 20

// emailShape matches chk_users_email_shape in the database.
var emailShape = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)

// UsersFileEntry is one account in the users file, trimmed and checked.
type UsersFileEntry struct {
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	Role         Role    `json:"role"`
	Outlet       *string `json:"outlet"`
	PasswordHash string  `json:"password_hash"`
}

// UsersFile is the parse result: the entries to load, and the entries skipped
// with the reason for each.
type UsersFile struct {
	Entries []UsersFileEntry
	Skipped []EntryProblem
}

// FileError is a problem with the whole file; the server refuses to start.
// Entry is the 1-based entry number, or 0 when the problem is not one entry.
// It never carries a field's value.
type FileError struct {
	Entry  int
	Field  string
	Reason string
}

func (e *FileError) Error() string {
	switch {
	case e.Entry > 0 && e.Field != "":
		return fmt.Sprintf("entry %d, field %s: %s", e.Entry, e.Field, e.Reason)
	case e.Entry > 0:
		return fmt.Sprintf("entry %d: %s", e.Entry, e.Reason)
	default:
		return e.Reason
	}
}

// EntryProblem is why one entry was skipped. The account it describes cannot
// sign in. It never carries a field's value.
type EntryProblem struct {
	Entry  int
	Field  string
	Reason string
}

// ReadUsersFile opens the users file at path and parses it.
func ReadUsersFile(path string) (UsersFile, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UsersFile{}, &FileError{Reason: "the users file does not exist (copy users.example.json and fill in hashes from make hash-password)"}
		}
		return UsersFile{}, &FileError{Reason: "the users file cannot be read: " + err.Error()}
	}
	defer func() { _ = f.Close() }() // read-only file; a close error changes nothing
	return ParseUsers(f)
}

// ParseUsers reads a users file from r. A *FileError means the server must not
// start; the returned UsersFile then still lists the skipped entries, so the
// caller can say why no valid brand admin was left.
func ParseUsers(r io.Reader) (UsersFile, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxUsersFileBytes+1))
	if err != nil {
		return UsersFile{}, &FileError{Reason: "the users file cannot be read: " + err.Error()}
	}
	if len(data) > maxUsersFileBytes {
		return UsersFile{}, &FileError{Reason: "the users file is over 1 MiB"}
	}

	raws, err := decodeArray(data)
	if err != nil {
		return UsersFile{}, err
	}

	var out UsersFile
	seen := make(map[string]int, len(raws)) // lower-case email to entry number
	admins := 0
	for i, raw := range raws {
		n := i + 1
		e, err := decodeEntry(n, raw)
		if err != nil {
			return out, err
		}
		key := strings.ToLower(e.Email)
		if key != "" {
			if first, dup := seen[key]; dup {
				return out, &FileError{Entry: n, Field: "email", Reason: fmt.Sprintf("the same email as entry %d, ignoring case", first)}
			}
			seen[key] = n
		}
		if p := checkEntry(n, e); p != nil {
			out.Skipped = append(out.Skipped, *p)
			continue
		}
		if e.Role == RoleBrandAdmin {
			admins++
		}
		out.Entries = append(out.Entries, e)
	}
	if admins != 1 {
		return out, &FileError{Reason: fmt.Sprintf("the users file must hold exactly one valid brand_admin entry; it holds %d", admins)}
	}
	return out, nil
}

// decodeArray splits the file into its entries without decoding them, so each
// entry's problems can name its number.
func decodeArray(data []byte) ([]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var raws []json.RawMessage
	if err := dec.Decode(&raws); err != nil {
		return nil, &FileError{Reason: "the users file is not a JSON array of entries: " + jsonProblem(err)}
	}
	if dec.More() {
		return nil, &FileError{Reason: "the users file has data after the closing ]"}
	}
	return raws, nil
}

// decodeEntry decodes one entry, refusing unknown fields and wrong types.
func decodeEntry(n int, raw json.RawMessage) (UsersFileEntry, error) {
	var e UsersFileEntry
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return e, &FileError{Entry: n, Field: typeErr.Field, Reason: "wrong type, want " + typeErr.Type.String()}
		}
		if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return e, &FileError{Entry: n, Field: strings.Trim(field, `"`), Reason: "unknown field"}
		}
		return e, &FileError{Entry: n, Reason: "not a JSON object: " + jsonProblem(err)}
	}
	e.Email = strings.TrimSpace(e.Email)
	e.Name = strings.TrimSpace(e.Name)
	if e.Outlet != nil {
		trimmed := strings.TrimSpace(*e.Outlet)
		e.Outlet = &trimmed
	}
	return e, nil
}

// checkEntry returns why an entry must be skipped, or nil.
func checkEntry(n int, e UsersFileEntry) *EntryProblem {
	problem := func(field, reason string) *EntryProblem {
		return &EntryProblem{Entry: n, Field: field, Reason: reason}
	}
	switch {
	case e.Email == "":
		return problem("email", "blank")
	case utf8.RuneCountInString(e.Email) > maxFieldRunes:
		return problem("email", "over 200 characters")
	case !emailShape.MatchString(e.Email):
		return problem("email", "not an email address")
	case e.Name == "":
		return problem("name", "blank")
	case utf8.RuneCountInString(e.Name) > maxFieldRunes:
		return problem("name", "over 200 characters")
	}
	switch e.Role {
	case RoleBrandAdmin:
		if e.Outlet != nil {
			return problem("outlet", "a brand_admin has no outlet; use null")
		}
	case RoleOutletManager:
		switch {
		case e.Outlet == nil || *e.Outlet == "":
			return problem("outlet", "an outlet_manager needs the name of an existing outlet")
		case utf8.RuneCountInString(*e.Outlet) > maxFieldRunes:
			return problem("outlet", "over 200 characters")
		}
	default:
		return problem("role", "must be brand_admin or outlet_manager")
	}
	cost, err := bcrypt.Cost([]byte(e.PasswordHash))
	switch {
	case err != nil:
		return problem("password_hash", "not a bcrypt hash (make one with make hash-password)")
	case cost != BcryptCost:
		return problem("password_hash", fmt.Sprintf("bcrypt cost %d, want %d (make one with make hash-password)", cost, BcryptCost))
	}
	return nil
}

// jsonProblem describes a decode error by position only, never by content.
func jsonProblem(err error) string {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("invalid JSON at byte %d", syntaxErr.Offset)
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return fmt.Sprintf("a %s where an array was expected", typeErr.Value)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "the file is empty or ends early"
	}
	return "unreadable JSON"
}
