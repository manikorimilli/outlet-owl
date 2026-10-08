//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/internal/store/storetest"
)

func TestMigration00003_UpDownUp(t *testing.T) {
	db := storetest.New(t)
	ctx := context.Background()
	exists := func() bool {
		var ok bool
		_ = db.Pool.QueryRow(ctx, "SELECT to_regclass('public.replies') IS NOT NULL").Scan(&ok)
		return ok
	}
	if !exists() {
		t.Fatal("replies missing after up")
	}
	db.Goose(t, "down-to", "2")
	if exists() {
		t.Fatal("replies left after down")
	}
	db.Goose(t, "up")
	if !exists() {
		t.Fatal("replies missing after up again")
	}
}

// managerUser inserts an active manager of the outlet and returns its id.
func managerUser(ctx context.Context, t *testing.T, s *Store, outlet int64) int64 {
	t.Helper()
	var id int64
	if err := s.Pool.QueryRow(ctx, `INSERT INTO users (email, name, role, outlet_id, password_hash)
		VALUES ('m@example.in', 'Arjun Mehta', 'outlet_manager', $1, 'x') RETURNING id`, outlet).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// The claim, its token, edits and approval as the data model section 5 sets
// them: one claim wins, a stale token writes nothing, approval is final, and
// another outlet's manager writes nothing (AC-US-00-003-2, -3).
func TestReplies_ClaimLandEditApprove(t *testing.T) {
	s, ctx, a, b := seedReviews(t)
	ids, _ := s.ListReviews(ctx, auth.Scope{OutletID: a}, reviews.Filter{}, 10)
	review := ids[0].ID
	user := managerUser(ctx, t, s, a)

	if _, ok, _ := s.ClaimDraft(ctx, review, b); ok {
		t.Fatal("another outlet's manager claimed the review")
	}
	claimed, ok, err := s.ClaimDraft(ctx, review, a)
	if err != nil || !ok {
		t.Fatalf("claim: %v %v", ok, err)
	}
	if _, again, _ := s.ClaimDraft(ctx, review, a); again {
		t.Fatal("a second claim won")
	}
	if _, took, _ := s.TakeOverClaim(ctx, review); took {
		t.Fatal("a fresh claim was taken over")
	}
	if landed, _ := s.LandDraft(ctx, review, claimed.Add(time.Microsecond), "x", 1); landed {
		t.Fatal("a draft landed with the wrong token")
	}
	if landed, err := s.LandDraft(ctx, review, claimed, "Hi Asha", 1); err != nil || !landed {
		t.Fatalf("land: %v %v", landed, err)
	}
	r, _ := s.Reply(ctx, review)
	if r.Status != "draft" || *r.DraftText != "Hi Asha" || *r.PromptVersion != 1 {
		t.Fatalf("reply = %+v", r)
	}
	if ok, _ := s.SaveReplyText(ctx, review, b, "x", r.UpdatedAt); ok {
		t.Fatal("another outlet saved the reply")
	}
	if ok, _ := s.SaveReplyText(ctx, review, a, "Hi Asha, sorry.", r.UpdatedAt); !ok {
		t.Fatal("edit not saved")
	}
	if ok, _ := s.MarkReplied(ctx, review, a, user, "Hi Asha, sorry.", r.UpdatedAt); ok {
		t.Fatal("approved with a stale token")
	}
	r, _ = s.Reply(ctx, review)
	if ok, err := s.MarkReplied(ctx, review, a, user, "Hi Asha, sorry.", r.UpdatedAt); err != nil || !ok {
		t.Fatalf("mark replied: %v %v", ok, err)
	}
	r, _ = s.Reply(ctx, review)
	if r.Status != "replied" || r.RepliedBy == nil || r.RepliedBy.Name != "Arjun Mehta" || r.RepliedAt == nil {
		t.Fatalf("replied = %+v", r)
	}
	if ok, _ := s.SaveReplyText(ctx, review, a, "late", r.UpdatedAt); ok {
		t.Fatal("an approved text changed")
	}

	// AC-US-00-003-4 and the list: status shown, filters by it, trends count it.
	replied := "replied"
	got, _ := s.ListReviews(ctx, auth.Scope{All: true}, reviews.Filter{ReplyStatus: replied}, 10)
	none, _ := s.CountReviews(ctx, auth.Scope{All: true}, reviews.Filter{ReplyStatus: "none"})
	if len(got) != 1 || got[0].ReplyStatus != "replied" || none != 3 {
		t.Fatalf("replied %d, none %d", len(got), none)
	}
	from, _ := time.Parse(time.DateOnly, "2026-09-21")
	weeks, _ := s.WeeklyOutletStats(ctx, auth.Scope{OutletID: a}, from, from.AddDate(0, 0, 13))
	if weeks[0].Replied != 1 {
		t.Fatalf("trends replied = %d", weeks[0].Replied)
	}
}

func TestReplies_ReleaseAndHandWritten(t *testing.T) {
	s, ctx, a, _ := seedReviews(t)
	ids, _ := s.ListReviews(ctx, auth.Scope{OutletID: a}, reviews.Filter{}, 10)
	review := ids[0].ID
	claimed, _, _ := s.ClaimDraft(ctx, review, a)
	if err := s.ReleaseClaim(ctx, review, claimed); err != nil {
		t.Fatal(err)
	}
	if r, _ := s.Reply(ctx, review); r != nil {
		t.Fatalf("claim left after release: %+v", r)
	}
	if ok, err := s.InsertHandReply(ctx, review, a, "Sorry, we are on it."); err != nil || !ok {
		t.Fatalf("hand reply: %v %v", ok, err)
	}
	if ok, _ := s.InsertHandReply(ctx, review, a, "again"); ok {
		t.Fatal("a second hand reply was inserted")
	}
	if _, err := s.Pool.Exec(ctx, "UPDATE replies SET status = 'drafting', reply_text = NULL, updated_at = now() - interval '2 minutes'"); err != nil {
		t.Fatal(err)
	}
	if _, took, err := s.TakeOverClaim(ctx, review); err != nil || !took {
		t.Fatalf("stale claim not taken over: %v %v", took, err)
	}
}
