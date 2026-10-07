// Command hashpw prints a bcrypt hash (cost 12) of a password read from
// stdin, for the password_hash field of the users file (phase 1 server LLD,
// section 3). `make hash-password` runs it with the terminal echo off. The
// password is never printed or logged.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/manikorimilli/outlet-owl/internal/auth"
)

// maxInput bounds what is read from stdin: a password is at most 72 bytes,
// plus a line ending.
const maxInput = 1024

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "hashpw:", err)
		os.Exit(1)
	}
}

// run reads one password from in and writes its hash and a newline to out.
// Only a trailing line ending is removed; spaces are part of the password.
func run(in io.Reader, out io.Writer) error {
	raw, err := io.ReadAll(io.LimitReader(in, maxInput+1))
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	if len(raw) > maxInput {
		return errors.New("input is too long; send one password")
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if strings.ContainsAny(password, "\r\n") {
		return errors.New("input has more than one line; send one password")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, hash); err != nil {
		return fmt.Errorf("write hash: %w", err)
	}
	return nil
}
