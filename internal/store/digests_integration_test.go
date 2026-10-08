//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/digest"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

func TestMigration00004_UpDownUp(t *testing.T) {
	db := storetest.New(t)
	ctx := context.Background()
	exists := func() bool {
		var ok bool
		_ = db.Pool.QueryRow(ctx, "SELECT to_regclass('public.digests') IS NOT NULL").Scan(&ok)
		return ok
	}
	db.Goose(t, "down-to", "3")
	if exists() {
		t.Fatal("digests left after down")
	}
	db.Goose(t, "up")
	if !exists() {
		t.Fatal("digests missing after up")
	}
}

// Tenet 8: one digest per request id, claimed as sending, then sent once.
func TestDigests_ClaimOnceThenSent(t *testing.T) {
	db := storetest.New(t)
	s := &Store{Pool: db.Pool}
	ctx := context.Background()
	monday, _ := time.Parse(time.DateOnly, "2026-09-28")
	key := "7192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b"
	first, created, err := s.ClaimDigest(ctx, key, digest.Digest{WeekStart: monday, Recipient: "r@example.in", Subject: "s", Body: "b"})
	if err != nil || !created || first.Status != "sending" {
		t.Fatalf("claim = %+v, %v, %v", first, created, err)
	}
	again, created, err := s.ClaimDigest(ctx, key, digest.Digest{WeekStart: monday, Recipient: "other@example.in", Subject: "x", Body: "y"})
	if err != nil || created || again.ID != first.ID || again.Recipient != "r@example.in" {
		t.Fatalf("second claim = %+v, %v, %v", again, created, err)
	}
	if _, err := s.MarkDigestSent(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	got, ok, _ := s.DigestByRequestID(ctx, key)
	if !ok || got.Status != "sent" || got.SentAt == nil {
		t.Fatalf("stored = %+v", got)
	}
	if err := s.MarkDigestFailed(ctx, first.ID, "late"); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := s.DigestByRequestID(ctx, key); got.Status != "sent" {
		t.Fatal("a sent digest was marked failed")
	}
	if _, _, err := s.ClaimDigest(ctx, "8192b1c2-7f3a-7c4e-9a1b-2c3d4e5f6a7b", digest.Digest{WeekStart: monday.AddDate(0, 0, 1), Recipient: "r", Subject: "s", Body: "b"}); err == nil {
		t.Fatal("a week that does not start on Monday was stored")
	}
}
