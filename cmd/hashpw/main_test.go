package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

func TestHashpw_PrintsHashThatChecksAgainstInput(t *testing.T) {
	var out bytes.Buffer

	if err := run(strings.NewReader("correct horse battery\n"), &out); err != nil {
		t.Fatalf("run: %v", err)
	}

	hash := strings.TrimSuffix(out.String(), "\n")
	if !auth.CheckPassword(hash, "correct horse battery") {
		t.Fatalf("printed hash %q does not check against the input", hash)
	}
	if strings.Contains(out.String(), "correct horse battery") {
		t.Fatal("the password appears in the output")
	}
}

var bcryptHash = regexp.MustCompile(`\$2[aby]\$12\$[./A-Za-z0-9]{53}`)

// The make target is how people run hashpw, typing on a terminal; its read
// once trimmed the spaces a password starts or ends with on a terminal only,
// so the hash matched a different password. script(1) gives make a
// pseudo-terminal, so this reaches the same read a person does.
func TestMakeHashPassword_KeepsSpacesWhenTyped(t *testing.T) {
	const password = "  correct horse \t"
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	makeCmd := "make --no-print-directory -s -C '" + root + "' hash-password"
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("script", "-qec", makeCmd, "/dev/null")
	case "darwin", "freebsd":
		cmd = exec.Command("script", "-q", "/dev/null", "sh", "-c", makeCmd)
	default:
		t.Fatalf("no script(1) form known for %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(password + "\r") // Enter on a terminal

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make hash-password on a terminal: %v\n%s", err, out)
	}

	hash := bcryptHash.FindString(string(out))
	if hash == "" {
		t.Fatalf("no bcrypt hash in the terminal output:\n%q", out)
	}
	if !auth.CheckPassword(hash, password) {
		t.Fatalf("the hash typed on a terminal does not check against the password with its spaces")
	}
	if auth.CheckPassword(hash, strings.TrimSpace(password)) {
		t.Fatal("the hash typed on a terminal checks against the trimmed password")
	}
}

// Piped input (a script, CI) reaches the same read.
func TestMakeHashPassword_KeepsLeadingAndTrailingSpaces(t *testing.T) {
	const password = "  correct horse \t"
	cmd := exec.Command("make", "--no-print-directory", "-s", "-C", filepath.Join("..", ".."), "hash-password")
	cmd.Stdin = strings.NewReader(password + "\n")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("make hash-password: %v", err)
	}

	hash := strings.TrimSuffix(string(out), "\n")
	if !auth.CheckPassword(hash, password) {
		t.Fatalf("printed hash %q does not check against the password with its spaces", hash)
	}
	if auth.CheckPassword(hash, strings.TrimSpace(password)) {
		t.Fatal("the hash checks against the trimmed password")
	}
}

func TestHashpw_Refuses(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr error
		wantMsg string
	}{
		{name: "over 72 bytes", input: strings.Repeat("a", auth.MaxPasswordBytes+1) + "\n", wantErr: auth.ErrPasswordTooLong},
		{name: "empty line", input: "\n", wantErr: auth.ErrPasswordEmpty},
		{name: "two lines", input: "one\ntwo\n", wantMsg: "more than one line"},
		{name: "over 1 KiB", input: strings.Repeat("a", maxInput+1), wantMsg: "too long"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			err := run(strings.NewReader(tc.input), &out)

			if err == nil {
				t.Fatal("want an error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.wantMsg)
			}
			if out.Len() != 0 {
				t.Fatalf("printed %q on error", out.String())
			}
		})
	}
}
