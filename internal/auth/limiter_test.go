package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

const lockedEmail = "arjun.mehta@example.in"

// failLogins makes n wrong-password sign-ins for email.
func failLogins(t *testing.T, svc *Service, email string, n int) {
	t.Helper()
	for i := range n {
		if _, _, err := svc.Login(context.Background(), email, "wrong-horse"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCredentials", i+1, err)
		}
	}
}

func TestLogin_LocksAfterMaxFailures(t *testing.T) {
	c := &clock{t: issuedAt}
	svc, _ := newService(t, c, managerRecord(t, "correct-horse"))

	failLogins(t, svc, lockedEmail, MaxFailedLogins)

	// Even the right password is refused while the lock holds.
	_, token, err := svc.Login(context.Background(), "Arjun.Mehta@Example.in", "correct-horse")
	var locked *TooManyAttemptsError
	if !errors.As(err, &locked) || token != "" {
		t.Fatalf("err = %v, token %q; want TooManyAttemptsError and no token", err, token)
	}
	if locked.RetryAfter != LockDuration {
		t.Fatalf("RetryAfter = %s, want %s", locked.RetryAfter, LockDuration)
	}

	// The lock ends after LockDuration.
	c.t = c.t.Add(LockDuration)
	if _, _, err := svc.Login(context.Background(), lockedEmail, "correct-horse"); err != nil {
		t.Fatalf("after the lock: err = %v, want a sign-in", err)
	}
}

func TestLogin_FewerFailuresDoNotLock(t *testing.T) {
	svc, _ := newService(t, &clock{t: issuedAt}, managerRecord(t, "correct-horse"))

	failLogins(t, svc, lockedEmail, MaxFailedLogins-1)

	if _, _, err := svc.Login(context.Background(), lockedEmail, "correct-horse"); err != nil {
		t.Fatalf("err = %v, want a sign-in", err)
	}
}

func TestLogin_SuccessClearsFailures(t *testing.T) {
	svc, _ := newService(t, &clock{t: issuedAt}, managerRecord(t, "correct-horse"))

	failLogins(t, svc, lockedEmail, MaxFailedLogins-1)
	if _, _, err := svc.Login(context.Background(), lockedEmail, "correct-horse"); err != nil {
		t.Fatalf("sign-in: %v", err)
	}
	// A fresh count: four more failures still do not lock.
	failLogins(t, svc, lockedEmail, MaxFailedLogins-1)
	if _, _, err := svc.Login(context.Background(), lockedEmail, "correct-horse"); err != nil {
		t.Fatalf("err = %v, want a sign-in", err)
	}
}

func TestLogin_FailuresOutsideTheWindowAreForgotten(t *testing.T) {
	c := &clock{t: issuedAt}
	svc, _ := newService(t, c, managerRecord(t, "correct-horse"))

	failLogins(t, svc, lockedEmail, MaxFailedLogins-1)
	c.t = c.t.Add(FailureWindow)
	failLogins(t, svc, lockedEmail, 1)

	if _, _, err := svc.Login(context.Background(), lockedEmail, "correct-horse"); err != nil {
		t.Fatalf("err = %v, want a sign-in", err)
	}
}

func TestLogin_UnknownEmailsLockLikeKnownOnes(t *testing.T) {
	// The lock answers the same for an email that has no account, so it
	// never tells which emails exist.
	svc, _ := newService(t, &clock{t: issuedAt})

	failLogins(t, svc, "nobody@example.in", MaxFailedLogins)

	var locked *TooManyAttemptsError
	if _, _, err := svc.Login(context.Background(), "nobody@example.in", "x"); !errors.As(err, &locked) {
		t.Fatalf("err = %v, want TooManyAttemptsError", err)
	}
}

func TestLimiter_RetryAfterRoundsUp(t *testing.T) {
	l := newLoginLimiter()
	now := issuedAt
	for range MaxFailedLogins {
		l.fail(lockedEmail, now)
	}
	err := l.check(lockedEmail, now.Add(LockDuration-1500*time.Millisecond))
	var locked *TooManyAttemptsError
	if !errors.As(err, &locked) || locked.RetryAfter != 2*time.Second {
		t.Fatalf("err = %v, want RetryAfter 2s", err)
	}
}

func TestLimiter_SweepDropsStaleEmails(t *testing.T) {
	l := newLoginLimiter()
	l.fail("a@example.in", issuedAt)
	l.fail("b@example.in", issuedAt.Add(FailureWindow))

	if _, ok := l.byEmail["a@example.in"]; ok {
		t.Fatal("a stale email was kept")
	}
	if len(l.byEmail) != 1 {
		t.Fatalf("map holds %d emails, want 1", len(l.byEmail))
	}
}
