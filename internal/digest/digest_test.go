package digest

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
)

func d(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t }

var movers = dashboard.Movers{
	Week:        dashboard.Week{Start: d("2026-09-28"), End: d("2026-10-04")},
	Previous:    dashboard.Week{Start: d("2026-09-21"), End: d("2026-09-27")},
	ReviewCount: 66,
	Movers: []dashboard.Mover{
		{Outlet: outlets.Outlet{Name: "Koramangala"}, Theme: tagging.Theme{Code: "wait_time", Label: "Wait time"}, Previous: 3, Current: 11, Change: 8},
		{Outlet: outlets.Outlet{Name: "Indiranagar"}, Theme: tagging.Theme{Code: "staff", Label: "Staff"}, Previous: 4, Current: 2, Change: -2},
	},
}

var urgentReview = reviews.Review{OutletName: "Jayanagar", ReviewDate: d("2026-10-02"), Rating: 1, Text: "<b>Found a cockroach</b>",
	Tags: &reviews.Tags{UrgentReasons: []string{"food_safety", "legal_threat"}}}

// AC-US-01-009-2, -3, -4: the first line names the top mover with both
// counts; movers are listed in rank order; urgent reviews carry reasons,
// outlet, date and text, as plain text.
func TestCompose(t *testing.T) {
	subject, body := Compose(movers, []reviews.Review{urgentReview})
	lines := strings.Split(body, "\n")
	if lines[0] != "Biggest mover: Koramangala, wait time. Negative reviews 3 last week, 11 this week (up 8)." {
		t.Fatalf("first line = %q", lines[0])
	}
	if subject != "Weekly digest, 28 Sep to 4 Oct 2026: Koramangala wait time up 8" {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{"1. Koramangala, wait time: 3 to 11 (up 8)", "2. Indiranagar, staff: 4 to 2 (down 2)",
		"Urgent reviews this week: 1", "Jayanagar, 2 Oct 2026, 1/5. Urgent: food safety, legal threat.", "<b>Found a cockroach</b>"} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
}

// HLD flow C: untagged reviews in the two weeks are named on the first line.
func TestCompose_UntaggedFirst(t *testing.T) {
	m := movers
	m.Untagged = 12
	_, body := Compose(m, nil)
	if !strings.HasPrefix(body, "12 reviews in these weeks are not tagged yet; movers and urgent reviews may be incomplete.\nBiggest mover:") {
		t.Fatalf("body = %q", body)
	}
}

func TestCompose_SubjectStaysOneLine(t *testing.T) {
	m := movers
	m.Movers = []dashboard.Mover{{Outlet: outlets.Outlet{Name: "Evil\r\nBcc: x@y"}, Theme: tagging.Theme{Label: "Food"}, Change: 1}}
	if s, _ := Compose(m, nil); strings.ContainsAny(s, "\r\n") {
		t.Fatalf("subject %q holds a line break", s)
	}
}

type memStore struct {
	digests map[string]Digest
	failed  string
}

func (m *memStore) DigestByRequestID(_ context.Context, k string) (Digest, bool, error) {
	x, ok := m.digests[k]
	return x, ok, nil
}
func (m *memStore) ActiveBrandAdminEmail(context.Context) (string, error) {
	return "ritika@example.in", nil
}
func (m *memStore) ClaimDigest(_ context.Context, k string, x Digest) (Digest, bool, error) {
	if prev, ok := m.digests[k]; ok {
		return prev, false, nil
	}
	x.ID, x.Status = int64(len(m.digests)+1), "sending"
	m.digests[k] = x
	return x, true, nil
}
func (m *memStore) MarkDigestSent(_ context.Context, id int64) (time.Time, error) {
	for k, x := range m.digests {
		if x.ID == id {
			x.Status = "sent"
			m.digests[k] = x
		}
	}
	return time.Now(), nil
}
func (m *memStore) MarkDigestFailed(_ context.Context, id int64, reason string) error {
	m.failed = reason
	for k, x := range m.digests {
		if x.ID == id {
			x.Status = "failed"
			m.digests[k] = x
		}
	}
	return nil
}
func (m *memStore) ListReviews(context.Context, auth.Scope, reviews.Filter, int) ([]reviews.Review, error) {
	return []reviews.Review{urgentReview}, nil
}

type fixedReports struct{}

func (fixedReports) Movers(context.Context, auth.User) (dashboard.Movers, error) { return movers, nil }

type countMail struct {
	sent int
	to   string
	err  error
}

func (c *countMail) Send(_ context.Context, to, _, _ string) error {
	c.sent++
	c.to = to
	return c.err
}

var admin = auth.User{Role: auth.RoleBrandAdmin}

const key = "0192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b"

// AC-US-01-009-1, -5 and tenet 8: generated on request, one email to the
// brand admin; a repeat of the key sends nothing more.
func TestGenerate_SendsOnceToTheAdmin(t *testing.T) {
	st, mail := &memStore{digests: map[string]Digest{}}, &countMail{}
	svc := NewService(st, fixedReports{}, mail)
	got, err := svc.Generate(context.Background(), admin, key)
	if err != nil || got.Status != "sent" || got.Recipient != "ritika@example.in" || mail.sent != 1 || mail.to != "ritika@example.in" {
		t.Fatalf("digest %+v, %v, sent %d", got, err, mail.sent)
	}
	if again, err := svc.Generate(context.Background(), admin, key); err != nil || again.ID != got.ID || mail.sent != 1 {
		t.Fatalf("repeat sent again: %d, %v", mail.sent, err)
	}
}

func TestGenerate_MailDownIsRecordedFailed(t *testing.T) {
	st := &memStore{digests: map[string]Digest{}}
	svc := NewService(st, fixedReports{}, &countMail{err: errors.New("connection refused")})
	var me *MailError
	if _, err := svc.Generate(context.Background(), admin, key); !errors.As(err, &me) || !strings.Contains(st.failed, "connection refused") {
		t.Fatalf("err %v, failed %q", err, st.failed)
	}
	if _, err := svc.Generate(context.Background(), admin, key); !errors.As(err, &me) {
		t.Fatalf("a repeat of a failed digest must answer the failure, got %v", err)
	}
}

func TestGenerate_ManagerRefused(t *testing.T) {
	mail := &countMail{}
	_, err := NewService(&memStore{digests: map[string]Digest{}}, fixedReports{}, mail).Generate(context.Background(), auth.User{Role: auth.RoleOutletManager}, key)
	if !errors.Is(err, ErrRoleNotAllowed) || mail.sent != 0 {
		t.Fatalf("err %v sent %d", err, mail.sent)
	}
}

// fakeSMTP accepts one message on a loopback port and hands back what it got.
func fakeSMTP(t *testing.T) (string, <-chan string) {
	t.Helper()
	return fakeSMTPWith(t, false)
}

// fakeSMTPWith can drop the connection at QUIT, after it has accepted the
// message.
func fakeSMTPWith(t *testing.T, dropAtQuit bool) (string, <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		write("220 fake")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					write("250 ok")
					got <- data.String()
					continue
				}
				data.WriteString(line)
				continue
			}
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				write("250 fake")
			case cmd == "DATA":
				inData = true
				write("354 go")
			case cmd == "QUIT":
				if !dropAtQuit {
					write("221 bye")
				}
				return
			default:
				write("250 ok")
			}
		}
	}()
	return ln.Addr().String(), got
}

func TestSMTP_SendsOnePlainTextMessage(t *testing.T) {
	addr, got := fakeSMTP(t)
	err := SMTP{Addr: addr, From: "digest@outletowl.local", Timeout: 5 * time.Second}.
		Send(context.Background(), "ritika@example.in", "Weekly digest: खाना", "Line one\n.hidden dot\nलाइन")
	if err != nil {
		t.Fatal(err)
	}
	msg := <-got
	for _, want := range []string{"To: ritika@example.in", "Subject: =?utf-8?q?", "Content-Type: text/plain; charset=UTF-8", "Line one\r\n..hidden dot\r\nलाइन"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
}

// Tenet 8: a message the server accepted is delivered, even when QUIT
// fails, so a retry never sends it twice.
func TestSMTP_AcceptedThenQuitFailsIsDelivered(t *testing.T) {
	addr, got := fakeSMTPWith(t, true)
	if err := (SMTP{Addr: addr, From: "a@b", Timeout: 5 * time.Second}).Send(context.Background(), "x@y", "s", "b"); err != nil {
		t.Fatalf("send = %v, want delivered", err)
	}
	<-got
}

func TestSMTP_NoServerIsAnError(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()
	if err := (SMTP{Addr: addr, From: "a@b", Timeout: time.Second}).Send(context.Background(), "x@y", "s", "b"); err == nil {
		t.Fatal("sending with no server must fail")
	}
}
