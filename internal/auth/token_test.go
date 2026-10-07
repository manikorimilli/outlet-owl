package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	testSecret = []byte("0123456789abcdef0123456789abcdef") // 32 bytes, test only
	issuedAt   = time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
)

// clock is a settable time source for Tokens.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestTokens(t *testing.T, c *clock) *Tokens {
	t.Helper()
	tk, err := NewTokens(testSecret, c.now)
	if err != nil {
		t.Fatalf("NewTokens: %v", err)
	}
	return tk
}

// signRaw signs arbitrary claims, for tokens Issue would never make.
func signRaw(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestNewTokens_RefusesShortSecret(t *testing.T) {
	if _, err := NewTokens(testSecret[:31], nil); err == nil {
		t.Fatal("want an error for a 31 byte secret")
	}
}

func TestTokens_IssueVerifyRoundTrip(t *testing.T) {
	c := &clock{t: issuedAt}
	tk := newTestTokens(t, c)

	s, err := tk.Issue(42)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	got, err := tk.Verify(s)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.UserID != 42 || !got.IssuedAt.Equal(issuedAt) {
		t.Fatalf("claims = %+v, want user 42 issued at %v", got, issuedAt)
	}
}

func TestTokens_ExpiredAfter8Hours(t *testing.T) {
	c := &clock{t: issuedAt}
	tk := newTestTokens(t, c)
	s, err := tk.Issue(42)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	c.t = issuedAt.Add(SessionTTL - time.Minute)
	if _, err := tk.Verify(s); err != nil {
		t.Fatalf("at 7h59m: %v, want valid", err)
	}
	c.t = issuedAt.Add(SessionTTL)
	if _, err := tk.Verify(s); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("at 8h: err = %v, want ErrUnauthenticated", err)
	}
}

func TestTokens_Rejects(t *testing.T) {
	c := &clock{t: issuedAt}
	tk := newTestTokens(t, c)
	valid, err := tk.Issue(42)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	other, err := NewTokens([]byte(strings.Repeat("z", 32)), c.now)
	if err != nil {
		t.Fatalf("NewTokens: %v", err)
	}
	fromOther, err := other.Issue(42)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	full := jwt.RegisteredClaims{
		Subject:   "42",
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(SessionTTL)),
	}
	// Change the first signature character: the last one can carry only
	// unused padding bits, so changing it may leave the signature intact.
	sig := strings.LastIndexByte(valid, '.') + 1
	flipped := byte('A')
	if valid[sig] == 'A' {
		flipped = 'B'
	}
	tampered := valid[:sig] + string(flipped) + valid[sig+1:]

	cases := []struct {
		name  string
		token string
	}{
		{name: "alg none", token: signRaw(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, full)},
		{name: "HS512 with the same secret", token: signRaw(t, jwt.SigningMethodHS512, testSecret, full)},
		{name: "tampered signature", token: tampered},
		{name: "signed with another secret", token: fromOther},
		{name: "no exp", token: signRaw(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: "42", IssuedAt: full.IssuedAt})},
		{name: "no iat", token: signRaw(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: "42", ExpiresAt: full.ExpiresAt})},
		{name: "iat in the future", token: signRaw(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: "42", IssuedAt: jwt.NewNumericDate(issuedAt.Add(time.Hour)), ExpiresAt: full.ExpiresAt})},
		{name: "subject not a number", token: signRaw(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: "admin", IssuedAt: full.IssuedAt, ExpiresAt: full.ExpiresAt})},
		{name: "subject zero", token: signRaw(t, jwt.SigningMethodHS256, testSecret, jwt.RegisteredClaims{Subject: "0", IssuedAt: full.IssuedAt, ExpiresAt: full.ExpiresAt})},
		{name: "not a token", token: "not-a-token"},
		{name: "empty", token: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tk.Verify(tc.token); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("err = %v, want ErrUnauthenticated", err)
			}
		})
	}
}
