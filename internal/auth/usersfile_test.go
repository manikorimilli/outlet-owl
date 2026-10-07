package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// validHash is a cost 12 bcrypt hash; dummyHash serves, so the tests hash
// nothing at cost 12.
const validHash = dummyHash

// entry is one users file entry as JSON; nil Outlet writes "outlet": null.
type entry map[string]any

func admin(email string) entry {
	return entry{"email": email, "name": "Ritika Rao", "role": "brand_admin", "outlet": nil, "password_hash": validHash}
}

func manager(email, outlet string) entry {
	return entry{"email": email, "name": "Arjun Mehta", "role": "outlet_manager", "outlet": outlet, "password_hash": validHash}
}

func with(e entry, key string, value any) entry {
	out := entry{}
	for k, v := range e {
		out[k] = v
	}
	out[key] = value
	return out
}

func without(e entry, key string) entry {
	out := with(e, key, nil)
	delete(out, key)
	return out
}

func fileOf(t *testing.T, entries ...entry) string {
	t.Helper()
	b, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func parse(t *testing.T, content string) (UsersFile, error) {
	t.Helper()
	return ParseUsers(strings.NewReader(content))
}

// fileError asserts err is a *FileError mentioning contains and returns a
// copy, so a caller that needs only the assertion can drop it.
func fileError(t *testing.T, err error, contains string) FileError {
	t.Helper()
	var fe *FileError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want a *FileError", err)
	}
	if !strings.Contains(fe.Error(), contains) {
		t.Fatalf("FileError %q does not mention %q", fe.Error(), contains)
	}
	return *fe
}

// wantFileError is fileError for tests that need only the assertion.
func wantFileError(t *testing.T, err error, contains string) {
	t.Helper()
	fileError(t, err, contains)
}

func TestParseUsersFile_ValidFile(t *testing.T) {
	got, err := parse(t, fileOf(t,
		with(admin(" Ritika.Rao@Example.in "), "name", "  Ritika Rao "),
		manager("arjun.mehta@example.in", " Koramangala "),
		manager("neha.k@example.in", "Indiranagar"),
	))
	if err != nil {
		t.Fatalf("ParseUsers: %v", err)
	}
	if len(got.Entries) != 3 || len(got.Skipped) != 0 {
		t.Fatalf("entries %d, skipped %v; want 3 and none", len(got.Entries), got.Skipped)
	}
	a := got.Entries[0]
	if a.Email != "Ritika.Rao@Example.in" || a.Name != "Ritika Rao" || a.Role != RoleBrandAdmin || a.Outlet != nil {
		t.Fatalf("admin entry = %+v, want trimmed values and no outlet", a)
	}
	if got.Entries[0].Entry != 1 || got.Entries[2].Entry != 3 {
		t.Fatalf("entry numbers = %d, %d; want 1 and 3", got.Entries[0].Entry, got.Entries[2].Entry)
	}
	if m := got.Entries[1]; m.Outlet == nil || *m.Outlet != "Koramangala" {
		t.Fatalf("manager outlet = %v, want Koramangala trimmed", m.Outlet)
	}
}

func TestParseUsersFile_MissingFileIsFileError(t *testing.T) {
	_, err := ReadUsersFile(filepath.Join(t.TempDir(), "users.json"))
	wantFileError(t, err, "does not exist")
}

func TestReadUsersFile_ReadsFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	if err := os.WriteFile(path, []byte(fileOf(t, admin("a@example.in"))), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadUsersFile(path)
	if err != nil || len(got.Entries) != 1 {
		t.Fatalf("ReadUsersFile = %d entries, %v; want 1, nil", len(got.Entries), err)
	}
}

func TestParseUsersFile_FileErrors(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		contains string
	}{
		{name: "empty file", content: "", contains: "empty"},
		{name: "an object, not an array", content: `{"email": "a@example.in"}`, contains: "JSON array"},
		{name: "invalid JSON", content: `[{"email": }]`, contains: "invalid JSON at byte"},
		{name: "data after the array", content: fileOf(t, admin("a@example.in")) + `[]`, contains: "after the closing"},
		{name: "an entry that is not an object", content: `[42]`, contains: "entry 1"},
		{name: "wrong field type", content: fileOf(t, with(admin("a@example.in"), "name", 7)), contains: "entry 1, field name: wrong type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parse(t, tc.content)
			wantFileError(t, err, tc.contains)
		})
	}
}

func TestParseUsersFile_UnknownFieldIsFileError(t *testing.T) {
	_, err := parse(t, fileOf(t,
		admin("a@example.in"),
		with(manager("m@example.in", "Koramangala"), "password", "plain"),
	))
	fe := fileError(t, err, "unknown field")
	if fe.Entry != 2 || fe.Field != "password" {
		t.Fatalf("FileError = %+v, want entry 2 field password", fe)
	}
}

func TestParseUsersFile_DuplicateEmailIgnoringCaseIsFileError(t *testing.T) {
	_, err := parse(t, fileOf(t,
		admin("a@example.in"),
		manager("m@example.in", "Koramangala"),
		manager("M@Example.IN", "Indiranagar"),
	))
	fe := fileError(t, err, "the same email as entry 2")
	if fe.Entry != 3 {
		t.Fatalf("FileError entry = %d, want 3", fe.Entry)
	}
}

func TestParseUsersFile_NoAdminIsFileError(t *testing.T) {
	_, err := parse(t, fileOf(t, manager("m@example.in", "Koramangala")))
	wantFileError(t, err, "exactly one valid brand_admin entry; it holds 0")
}

func TestParseUsersFile_TwoAdminsIsFileError(t *testing.T) {
	_, err := parse(t, fileOf(t, admin("a@example.in"), admin("b@example.in")))
	wantFileError(t, err, "it holds 2")
}

func TestParseUsersFile_AdminWithOutletLeavesNoAdmin(t *testing.T) {
	got, err := parse(t, fileOf(t, with(admin("a@example.in"), "outlet", "Koramangala")))
	wantFileError(t, err, "it holds 0")
	// The caller logs why: the skipped entry comes back with the error.
	if len(got.Skipped) != 1 || got.Skipped[0].Field != "outlet" {
		t.Fatalf("skipped = %+v, want the admin entry skipped for its outlet", got.Skipped)
	}
}

func TestParseUsersFile_SkippedEntries(t *testing.T) {
	lowCost, err := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	m := manager("m@example.in", "Koramangala")
	cases := []struct {
		name  string
		entry entry
		field string
	}{
		{name: "ManagerWithoutOutletIsSkipped", entry: without(m, "outlet"), field: "outlet"},
		{name: "manager with null outlet", entry: with(m, "outlet", nil), field: "outlet"},
		{name: "manager with blank outlet", entry: with(m, "outlet", "   "), field: "outlet"},
		{name: "outlet over 200 characters", entry: with(m, "outlet", strings.Repeat("o", 201)), field: "outlet"},
		{name: "CostOtherThan12IsSkipped", entry: with(m, "password_hash", string(lowCost)), field: "password_hash"},
		{name: "UnreadableHashIsSkipped", entry: with(m, "password_hash", "REPLACE with the output of make hash-password"), field: "password_hash"},
		{name: "BadEmailShapeIsSkipped", entry: with(m, "email", "arjun at example.in"), field: "email"},
		{name: "email over 200 characters", entry: with(m, "email", strings.Repeat("e", 190)+"@example.in"), field: "email"},
		{name: "blank email", entry: with(m, "email", " "), field: "email"},
		{name: "BlankNameIsSkipped", entry: with(m, "name", "  "), field: "name"},
		{name: "name over 200 characters", entry: with(m, "name", strings.Repeat("न", 201)), field: "name"},
		{name: "unknown role", entry: with(m, "role", "owner"), field: "role"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parse(t, fileOf(t, admin("a@example.in"), tc.entry))
			if err != nil {
				t.Fatalf("ParseUsers: %v; a bad manager entry must be skipped, not stop the file", err)
			}
			if len(got.Entries) != 1 || got.Entries[0].Role != RoleBrandAdmin {
				t.Fatalf("entries = %+v, want only the admin", got.Entries)
			}
			if len(got.Skipped) != 1 || got.Skipped[0].Entry != 2 || got.Skipped[0].Field != tc.field {
				t.Fatalf("skipped = %+v, want entry 2 field %s", got.Skipped, tc.field)
			}
		})
	}
}

func TestParseUsersFile_200CharactersAreAllowed(t *testing.T) {
	_, err := parse(t, fileOf(t,
		with(admin("a@example.in"), "name", strings.Repeat("न", 200)),
		manager("m@example.in", strings.Repeat("o", 200)),
	))
	if err != nil {
		t.Fatalf("ParseUsers: %v", err)
	}
}

func TestParseUsersFile_ErrorsNeverContainHash(t *testing.T) {
	const secretLooking = "$2a$12$SECRETSECRETSECRETSECRETSECRETSECRETSECRETSECRETSECR"
	files := []string{
		fileOf(t, with(admin("a@example.in"), "password_hash", 12345)),
		fileOf(t, admin("a@example.in"), with(admin("b@example.in"), "password_hash", secretLooking)),
		`[{"email": "a@example.in", "password_hash": "` + secretLooking + `", }]`,
	}
	for i, content := range files {
		got, err := parse(t, content)
		texts := []string{}
		if err != nil {
			texts = append(texts, err.Error())
		}
		for _, p := range got.Skipped {
			texts = append(texts, p.Field+" "+p.Reason)
		}
		for _, text := range texts {
			if strings.Contains(text, "SECRET") || strings.Contains(text, dummyHash) || strings.Contains(text, "12345") {
				t.Fatalf("file %d: %q carries a hash value", i, text)
			}
		}
	}
}

func TestUsersExampleFile_DocumentsTheFormat(t *testing.T) {
	got, err := ReadUsersFile(filepath.Join("..", "..", "users.example.json"))
	// The placeholders are not hashes, so both entries are skipped for that
	// reason alone and no admin is left; any other problem means the example
	// shows a wrong format.
	wantFileError(t, err, "it holds 0")
	if len(got.Skipped) != 2 {
		t.Fatalf("skipped = %+v, want both example entries", got.Skipped)
	}
	for _, p := range got.Skipped {
		if p.Field != "password_hash" {
			t.Fatalf("example entry %d skipped for %s (%s), want only the placeholder hash", p.Entry, p.Field, p.Reason)
		}
	}
}
