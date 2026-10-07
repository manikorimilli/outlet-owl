// Package auth signs people in: password checks against bcrypt hashes, the
// session token (ADR-0007), and later the users file and the sign-in service
// (phase 1 server LLD, section 2).
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the one cost every account's hash has. One cost keeps a wrong
// password for a known email as slow as the dummy check for an unknown one,
// so timing does not reveal which emails exist (phase 1 server LLD, section 3).
const BcryptCost = 12

// MaxPasswordBytes is bcrypt's input limit. Longer passwords are refused,
// never truncated.
const MaxPasswordBytes = 72

// ErrPasswordTooLong is returned by HashPassword for more than 72 bytes.
var ErrPasswordTooLong = errors.New("auth: password is longer than 72 bytes")

// ErrPasswordEmpty is returned by HashPassword for an empty password.
var ErrPasswordEmpty = errors.New("auth: password is empty")

// dummyHash is a cost 12 hash of 64 random hex characters nobody kept. A
// sign-in for an unknown email is checked against it, so it takes as long as
// a sign-in for a known email with a wrong password.
const dummyHash = "$2a$12$28FZoNZ7x5hHqyicGsCsYOicdsGWilNAG9iUgN2yd01XIbNoeU/uy"

// HashPassword returns a bcrypt hash of the password at BcryptCost.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrPasswordEmpty
	}
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// CheckPassword reports whether password matches hash. A password over 72
// bytes never matches: bcrypt would compare only its first 72 bytes.
func CheckPassword(hash, password string) bool {
	if len(password) > MaxPasswordBytes {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CheckDummyPassword spends the time of one CheckPassword at BcryptCost and
// always reports false. Call it when no account matched.
func CheckDummyPassword(password string) bool {
	CheckPassword(dummyHash, password)
	return false
}
