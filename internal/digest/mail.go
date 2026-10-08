package digest

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTP sends one plain-text message to the mail catcher. It talks to
// SMTP_ADDR only, with no authentication and no TLS: MailHog runs locally
// (PRD section 6, Q-006).
type SMTP struct {
	Addr    string
	From    string
	Timeout time.Duration
}

// Send delivers the message or returns why it could not.
func (s SMTP) Send(ctx context.Context, to, subject, body string) error {
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", s.Addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	host, _, _ := net.SplitHostPort(s.Addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("greet %s: %w", s.Addr, err)
	}
	defer func() { _ = c.Close() }()
	if err := c.Mail(s.From); err != nil {
		return fmt.Errorf("sender: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("recipient: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	msg := strings.Join([]string{
		"From: OutletOwl <" + s.From + ">",
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"),
	}, "\r\n")
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finish: %w", err)
	}
	return c.Quit()
}
