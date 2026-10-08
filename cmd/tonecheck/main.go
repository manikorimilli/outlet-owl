// Command tonecheck drafts 30 replies through the gateway into a Markdown
// sheet for a person to score against the tone rubric (US-02-005), and with
// -report totals a scored sheet. Drafts are not stored as replies.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/config"
	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/replies"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/tonecheck"
	"github.com/manikorimilli/outlet-owl/prompts"
)

func main() {
	out := flag.String("out", "tone-check.md", "where to write the scoring sheet")
	report := flag.String("report", "", "total the scores in this filled-in sheet instead of drafting")
	flag.Parse()
	var err error
	if *report != "" {
		err = summarise(*report)
	} else {
		err = draft(*out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tonecheck:", err)
		os.Exit(1)
	}
}

func summarise(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	scores, err := tonecheck.Totals(f)
	if err != nil {
		return err
	}
	fmt.Print(tonecheck.Summary(scores))
	return nil
}

func draft(out string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()
	gw, err := gateway.New(gateway.Config{Mode: cfg.GatewayMode, APIKey: cfg.OpenRouterKey, RecordingsDir: cfg.RecordingsDir, Store: st})
	if err != nil {
		return err
	}
	reg, err := prompts.Load()
	if err != nil {
		return err
	}
	prompt, err := reg.Current("reply")
	if err != nil {
		return err
	}
	all, err := st.ListReviews(ctx, auth.Scope{All: true}, reviews.Filter{}, 5000)
	if err != nil {
		return err
	}
	picked := tonecheck.Pick(all)
	if len(picked) < tonecheck.Size {
		return fmt.Errorf("found %d reviews; seed or import at least %d first", len(picked), tonecheck.Size)
	}
	var drafts []tonecheck.Draft
	for _, r := range picked {
		managers, err := st.ListActiveManagers(ctx, []int64{r.OutletID})
		if err != nil {
			return err
		}
		signer := "Outlet manager"
		if len(managers) > 0 {
			signer = managers[0].Name
		}
		resp, err := gw.Complete(ctx, gateway.Request{Purpose: gateway.ToneCheck, Prompt: prompt,
			User: replies.Message(replies.Input{Outlet: r.OutletName, ManagerName: signer, ReviewerName: r.ReviewerName, Rating: r.Rating, Text: r.Text})})
		if err != nil {
			return fmt.Errorf("draft for review %d: %w", r.ID, err)
		}
		drafts = append(drafts, tonecheck.Draft{Review: r, Text: resp.Text})
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := tonecheck.Write(f, drafts, prompt, gateway.Model); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("tonecheck: %d drafts written to %s with reply prompt v%d; score them, then run with -report %s\n", len(drafts), out, prompt.Number, out)
	return nil
}
