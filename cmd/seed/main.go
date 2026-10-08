// Command seed resets the demo data: 5 outlets, one brand admin and one
// manager per outlet in the users file, and 1,500 reviews over 26 weeks with
// a planted wait-time spike in the latest complete week (US-02-006, HLD flow
// D). It then tags the reviews in this process through the gateway, in the
// mode MODEL_GATEWAY_MODE sets. The budget record is never touched.
//
// Run it with the server stopped or idle; it holds the tagging lock while it
// works, so a server pass waits for it.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/config"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/seed"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
	"github.com/manikorimilli/outlet-owl/prompts"
)

// defaultPassword is the local demo password of every seeded account; set
// SEED_PASSWORD to choose another. Never reuse it for real data.
const defaultPassword = "outletowl-demo"

var managers = []string{"Neha Kulkarni", "Arjun Mehta", "Farhan Sheikh", "Deepa Iyer", "Vikram Rao"}

func main() {
	tag := flag.Bool("tag", true, "tag the reviews after seeding (replay needs recordings; record and live spend budget)")
	flag.Parse()
	if err := run(*tag); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

// lockedStore is the store with the tagging lock already held by the seed:
// the pass must not take it again on another connection.
type lockedStore struct{ *store.Store }

func (lockedStore) LockTagging(context.Context) (func(), error) { return func() {}, nil }

func run(tag bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	unlock, err := st.LockTagging(ctx)
	if err != nil {
		return fmt.Errorf("take the tagging lock: %w", err)
	}
	defer unlock()

	if err := st.ResetDomain(ctx); err != nil {
		return err
	}
	outletIDs := make([]int64, len(seed.Outlets))
	for i, name := range seed.Outlets {
		o, _, err := st.CreateOutlet(ctx, name)
		if err != nil {
			return err
		}
		outletIDs[i] = o.ID
	}
	if err := writeUsers(ctx, st, cfg.UsersFile); err != nil {
		return err
	}

	latest := dashboard.LatestCompleteWeek(time.Now(), cfg.BrandTimezone)
	gen := seed.Generate(latest.Start)
	rows := make([]store.SeedReview, len(gen))
	for i, r := range gen {
		rows[i] = store.SeedReview{OutletID: outletIDs[r.Outlet], Source: r.Source, Date: r.Date, Rating: r.Rating,
			Text: r.Text, ReviewerName: r.ReviewerName}
	}
	n, err := st.InsertSeedReviews(ctx, rows)
	if err != nil {
		return err
	}
	fmt.Printf("seed: %d outlets, %d reviews from %s to %s, users written to %s (password %q unless SEED_PASSWORD is set)\n",
		len(outletIDs), n, latest.Start.AddDate(0, 0, -7*(seed.Weeks-1)).Format(time.DateOnly), latest.End.Format(time.DateOnly),
		cfg.UsersFile, passwordShown())

	if !tag {
		fmt.Println("seed: tagging skipped (-tag=false); the server tags the reviews on its next pass")
		return nil
	}
	return tagAll(ctx, cfg, st, logger)
}

func passwordShown() string {
	if os.Getenv("SEED_PASSWORD") != "" {
		return "from SEED_PASSWORD"
	}
	return defaultPassword
}

// writeUsers writes the users file the server loads at start and applies it
// now, so the accounts exist at once (AC-US-02-006-5).
func writeUsers(ctx context.Context, st *store.Store, path string) error {
	pw := os.Getenv("SEED_PASSWORD")
	if pw == "" {
		pw = defaultPassword
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	entries := []auth.UsersFileEntry{{Email: "ritika.rao@example.in", Name: "Ritika Rao", Role: auth.RoleBrandAdmin, PasswordHash: hash}}
	for i, outlet := range seed.Outlets {
		o := outlet
		entries = append(entries, auth.UsersFileEntry{
			Email: fmt.Sprintf("manager%d@example.in", i+1), Name: managers[i], Role: auth.RoleOutletManager,
			Outlet: &o, PasswordHash: hash,
		})
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := st.SyncUsers(ctx, entries); err != nil {
		return fmt.Errorf("load the users: %w", err)
	}
	return nil
}

// tagAll tags every review once in this process, under the lock the seed
// holds, with the operator's gateway mode. A missing recording fails the
// seed rather than calling the model (tenet 5).
func tagAll(ctx context.Context, cfg config.Config, st *store.Store, logger *slog.Logger) error {
	gw, err := gateway.New(gateway.Config{Mode: cfg.GatewayMode, Model: cfg.ModelID, APIKey: cfg.OpenRouterKey, RecordingsDir: cfg.RecordingsDir, Store: st, Logger: logger})
	if err != nil {
		return err
	}
	reg, err := prompts.Load()
	if err != nil {
		return err
	}
	prompt, err := reg.Current("tagging")
	if err != nil {
		return err
	}
	w := tagging.New(tagging.Config{Store: lockedStore{st}, Model: gw, Prompt: prompt, Enabled: true, Logger: logger})
	start := time.Now()
	rep, err := w.Pass(ctx)
	fmt.Printf("seed: tagging in %s mode: %d tagged, %d unresolved, %d calls, %s\n",
		gw.Mode(), rep.Tagged, rep.Unresolved, rep.Calls, time.Since(start).Round(time.Second))
	if errors.Is(err, gateway.ErrRecordingMissing) {
		return fmt.Errorf("no recording for the tagging requests in %s; the reviews are seeded but untagged. "+
			"Record once on purpose (MODEL_GATEWAY_MODE=record, about USD 0.57, needs the model key in .env), "+
			"or run with -tag=false: %w", cfg.RecordingsDir, err)
	}
	return err
}
