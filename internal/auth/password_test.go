package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// fastHash makes a hash at bcrypt's minimum cost, so the check tests stay
// fast; the cost rule itself is tested on HashPassword and dummyHash.
func fastHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return string(h)
}

func TestCheckPassword(t *testing.T) {
	hash := fastHash(t, "correct-horse-battery")
	at72 := strings.Repeat("a", MaxPasswordBytes)
	hash72 := fastHash(t, at72)

	cases := []struct {
		name     string
		hash     string
		password string
		want     bool
	}{
		{name: "matching password", hash: hash, password: "correct-horse-battery", want: true},
		{name: "wrong password", hash: hash, password: "correct-horse-batterY", want: false},
		{name: "empty password", hash: hash, password: "", want: false},
		{name: "72 bytes matches its own hash", hash: hash72, password: at72, want: true},
		// bcrypt alone would accept this: it compares only the first 72 bytes.
		{name: "73 bytes never matches", hash: hash72, password: at72 + "b", want: false},
		{name: "not a bcrypt hash", hash: "plain-text", password: "plain-text", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckPassword(tc.hash, tc.password); got != tc.want {
				t.Fatalf("CheckPassword = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	t.Parallel() // cost 12 is slow under -race; run beside the other slow test
	h, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(h))
	if err != nil || cost != BcryptCost {
		t.Fatalf("cost = %d (%v), want %d", cost, err, BcryptCost)
	}
	if !CheckPassword(h, "correct-horse-battery") {
		t.Fatal("the hash does not check against its own password")
	}
}

func TestHashPassword_Refuses(t *testing.T) {
	cases := []struct {
		name     string
		password string
		want     error
	}{
		{name: "empty", password: "", want: ErrPasswordEmpty},
		{name: "73 bytes", password: strings.Repeat("a", MaxPasswordBytes+1), want: ErrPasswordTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := HashPassword(tc.password); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestDummyHash_HasCost12(t *testing.T) {
	t.Parallel() // cost 12 is slow under -race; run beside the other slow test
	cost, err := bcrypt.Cost([]byte(dummyHash))
	if err != nil {
		t.Fatalf("dummyHash is not a bcrypt hash: %v", err)
	}
	if cost != BcryptCost {
		t.Fatalf("dummyHash cost = %d, want %d", cost, BcryptCost)
	}
	if CheckDummyPassword("anything") {
		t.Fatal("CheckDummyPassword must always report false")
	}
}
