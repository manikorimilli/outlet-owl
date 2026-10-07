package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionTTL is how long a session token and its cookie last (HLD design
// choice 4, ADR-0007).
const SessionTTL = 8 * time.Hour

// minSecretBytes matches the config rule: 32 bytes is the HS256 key size.
const minSecretBytes = 32

// ErrUnauthenticated means the caller has no valid session: the token is
// missing, malformed, signed with another key or algorithm, or expired.
var ErrUnauthenticated = errors.New("auth: not signed in")

// Claims is what a verified token says: the user id and when it was issued.
// Role and outlet are never in the token; they are re-read per request.
type Claims struct {
	UserID   int64
	IssuedAt time.Time
}

// Tokens issues and verifies HS256 session tokens.
type Tokens struct {
	secret []byte
	now    func() time.Time
}

// NewTokens builds a Tokens with the signing secret and a clock (nil means
// time.Now). It refuses a secret shorter than 32 bytes.
func NewTokens(secret []byte, now func() time.Time) (*Tokens, error) {
	if len(secret) < minSecretBytes {
		return nil, fmt.Errorf("auth: signing secret must be at least %d bytes", minSecretBytes)
	}
	if now == nil {
		now = time.Now
	}
	return &Tokens{secret: secret, now: now}, nil
}

// Issue signs a token naming userID, valid for SessionTTL from now.
func (t *Tokens) Issue(userID int64) (string, error) {
	now := t.now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(SessionTTL)),
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign session token: %w", err)
	}
	return s, nil
}

// Verify checks the signature (HS256 only), the expiry and the claims, and
// returns who the token names. Every failure wraps ErrUnauthenticated.
func (t *Tokens) Verify(token string) (Claims, error) {
	var rc jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &rc,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}
	if rc.IssuedAt == nil {
		return Claims{}, fmt.Errorf("%w: token has no iat", ErrUnauthenticated)
	}
	id, err := strconv.ParseInt(rc.Subject, 10, 64)
	if err != nil || id <= 0 {
		return Claims{}, fmt.Errorf("%w: token subject is not a user id", ErrUnauthenticated)
	}
	return Claims{UserID: id, IssuedAt: rc.IssuedAt.Time}, nil
}
