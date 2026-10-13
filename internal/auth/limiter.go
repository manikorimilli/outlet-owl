package auth

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// Sign-in attempt limits (threat model T-01): MaxFailedLogins wrong
// passwords for one email inside FailureWindow lock that email for
// LockDuration. A successful sign-in clears the count.
const (
	MaxFailedLogins = 5
	FailureWindow   = 15 * time.Minute
	LockDuration    = 15 * time.Minute
)

// TooManyAttemptsError is what Login returns while an email is locked.
// RetryAfter is how long until the lock ends, rounded up to whole seconds.
type TooManyAttemptsError struct {
	RetryAfter time.Duration
}

func (e *TooManyAttemptsError) Error() string {
	return fmt.Sprintf("auth: too many failed sign-ins, retry in %s", e.RetryAfter)
}

// attempts is one email's recent failures and its lock, if any.
type attempts struct {
	failures    []time.Time
	lockedUntil time.Time
}

// loginLimiter counts failed sign-ins per email in memory. It is per
// process: the demo runs one replica (ADR-0010); a second replica would keep
// its own count, so each email could try MaxFailedLogins times per replica.
// The key is the lowercased email, so case changes do not reset the count.
type loginLimiter struct {
	mu      sync.Mutex
	byEmail map[string]*attempts
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{byEmail: map[string]*attempts{}}
}

func limiterKey(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// check returns a TooManyAttemptsError while email is locked at now.
func (l *loginLimiter) check(email string, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	a, ok := l.byEmail[limiterKey(email)]
	if !ok || !now.Before(a.lockedUntil) {
		return nil
	}
	// Rounded up, so Retry-After never says "try now" while the lock holds.
	wait := time.Duration(math.Ceil(a.lockedUntil.Sub(now).Seconds())) * time.Second
	return &TooManyAttemptsError{RetryAfter: wait}
}

// fail records a failed sign-in and locks the email on the
// MaxFailedLogins-th failure inside FailureWindow.
func (l *loginLimiter) fail(email string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)
	key := limiterKey(email)
	a, ok := l.byEmail[key]
	if !ok {
		a = &attempts{}
		l.byEmail[key] = a
	}
	recent := a.failures[:0]
	for _, t := range a.failures {
		if now.Sub(t) < FailureWindow {
			recent = append(recent, t)
		}
	}
	a.failures = append(recent, now)
	if len(a.failures) >= MaxFailedLogins {
		a.lockedUntil = now.Add(LockDuration)
		a.failures = nil
	}
}

// succeed clears an email's failures after a correct password.
func (l *loginLimiter) succeed(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.byEmail, limiterKey(email))
}

// sweep drops emails with no recent failure and no lock, so guessing many
// different emails cannot grow the map without bound. The caller holds mu.
func (l *loginLimiter) sweep(now time.Time) {
	for key, a := range l.byEmail {
		if now.Before(a.lockedUntil) {
			continue
		}
		if n := len(a.failures); n == 0 || now.Sub(a.failures[n-1]) >= FailureWindow {
			delete(l.byEmail, key)
		}
	}
}
