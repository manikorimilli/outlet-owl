// Command api runs the OutletOwl server. It wires config, logging, the store,
// the HTTP server and signal handling, and nothing else (HLD section 3).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	// Embeds the IANA zone database so BRAND_TIMEZONE loads on machines and
	// images without one (phase 1 server LLD, section 7).
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/config"
	"github.com/manikorimilli/outlet-owl/internal/dashboard"
	"github.com/manikorimilli/outlet-owl/internal/digest"
	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/httpapi"
	"github.com/manikorimilli/outlet-owl/internal/imports"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/replies"
	"github.com/manikorimilli/outlet-owl/internal/reviews"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/internal/tagging"
	"github.com/manikorimilli/outlet-owl/prompts"
)

var version = "dev"

// sqlstateUndefinedTable is PostgreSQL's code for a missing table: the
// migrations have not been applied.
const sqlstateUndefinedTable = "42P01"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer st.Close()

	if err := loadUsers(ctx, logger, st, cfg.UsersFile); err != nil {
		return err
	}
	gw, err := startGateway(ctx, logger, st, cfg)
	if err != nil {
		return err
	}
	reg, err := prompts.Load()
	if err != nil {
		return err
	}
	tagger, err := newTagger(logger, st, gw, reg, cfg.TaggingEnabled)
	if err != nil {
		return err
	}
	replyPrompt, err := reg.Current("reply")
	if err != nil {
		return err
	}
	tokens, err := auth.NewTokens(cfg.JWTSecret, nil)
	if err != nil {
		return fmt.Errorf("session tokens: %w", err)
	}

	reports := dashboard.NewService(st, time.Now, cfg.BrandTimezone)
	handler := httpapi.New(httpapi.Deps{
		Logger:    logger,
		DB:        st,
		Auth:      auth.NewService(st, tokens),
		Outlets:   outlets.NewService(st),
		Imports:   imports.NewService(st, tagger),
		Reviews:   reviews.NewService(st),
		Replies:   replies.NewService(st, gw, replyPrompt, logger),
		Dashboard: reports,
		Digests:   digest.NewService(st, reports, digest.SMTP{Addr: cfg.SMTPAddr, From: cfg.SMTPFrom}),
		Status:    statusReader{Store: st, worker: tagger, gw: gw},
		Brand:     httpapi.Brand{Name: cfg.BrandName, Timezone: cfg.BrandTimezone.String()},
		WebDir:    webDir(logger, cfg.WebDir),
	})
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// One tagging pass at start picks up reviews left untagged by an earlier
	// run (ADR-0006); later passes follow each import.
	taggerCtx, stopTagger := context.WithCancel(context.Background())
	taggerDone := make(chan struct{})
	go func() {
		defer close(taggerDone)
		tagger.Run(taggerCtx)
	}()
	tagger.Signal()
	defer func() {
		stopTagger()
		<-taggerDone
	}()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve: %w", err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("stopped")
	return nil
}

// webDir returns dir when it holds a built web app, else "" and a log line:
// the API still serves, and make web-dev serves the UI in development.
func webDir(logger *slog.Logger, dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		logger.Info("web app not built; serving the API only (make build, or make web-dev for development)", "dir", dir)
		return ""
	}
	return dir
}

// statusReader joins what GET /tagging/status reads: the counts from the
// store, the worker's state and the gateway's credit flag.
type statusReader struct {
	*store.Store
	worker *tagging.Worker
	gw     *gateway.Gateway
}

func (s statusReader) WorkerState() string   { return s.worker.State() }
func (s statusReader) CreditExhausted() bool { return s.gw.CreditExhausted() }

// newTagger builds the tagging worker with the current tagging prompt
// version built into the binary (Q-011).
func newTagger(logger *slog.Logger, st *store.Store, gw *gateway.Gateway, reg *prompts.Registry, enabled bool) (*tagging.Worker, error) {
	prompt, err := reg.Current("tagging")
	if err != nil {
		return nil, err
	}
	return tagging.New(tagging.Config{Store: st, Model: gw, Prompt: prompt, Enabled: enabled, Logger: logger}), nil
}

// loadUsers applies the users file before the server listens (HLD section 3,
// phase 1 server LLD section 4.1). A problem with the whole file, or a
// database without the tables, stops the start; a bad entry is logged and
// skipped, and that account cannot sign in.
func loadUsers(ctx context.Context, logger *slog.Logger, st *store.Store, path string) error {
	file, err := auth.ReadUsersFile(path)
	for _, p := range file.Skipped {
		logger.Warn("users file entry skipped", "file", path, "entry", p.Entry, "field", p.Field, "reason", p.Reason)
	}
	if err != nil {
		return fmt.Errorf("users file %s: %w", path, err)
	}
	res, err := st.SyncUsers(ctx, file.Entries)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == sqlstateUndefinedTable {
			return fmt.Errorf("users file %s: the users table does not exist; run make migrate: %w", path, err)
		}
		return fmt.Errorf("users file %s: %w", path, err)
	}
	for _, p := range res.Skipped {
		logger.Warn("users file entry skipped", "file", path, "entry", p.Entry, "field", p.Field, "reason", p.Reason)
	}
	logger.Info("users synced", "file", path, "entries", len(file.Entries)-len(res.Skipped),
		"changed", res.Changed, "removed", res.Removed, "skipped", len(file.Skipped)+len(res.Skipped))
	return nil
}

// startGateway builds the model gateway, refuses to start without the budget
// table, reconciles the running total with OpenRouter's key usage in live and
// record mode, and logs one budget line (HLD section 10).
func startGateway(ctx context.Context, logger *slog.Logger, st *store.Store, cfg config.Config) (*gateway.Gateway, error) {
	gw, err := gateway.New(gateway.Config{
		Mode:          cfg.GatewayMode,
		APIKey:        cfg.OpenRouterKey,
		RecordingsDir: cfg.RecordingsDir,
		Store:         st,
		Logger:        logger,
	})
	if err != nil {
		return nil, err
	}
	if _, err := st.RunningTotalUSD(ctx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == sqlstateUndefinedTable {
			return nil, fmt.Errorf("budget: the budget table does not exist; run make migrate: %w", err)
		}
		return nil, fmt.Errorf("budget: %w", err)
	}
	rec := gw.Reconcile(ctx)
	if rec.Err != nil {
		logger.Warn("budget unreconciled; the local total stands", "err", rec.Err)
	}
	total, err := st.RunningTotalUSD(ctx)
	if err != nil {
		return nil, fmt.Errorf("budget: %w", err)
	}
	logger.Info("budget", "mode", string(gw.Mode()), "total_usd", total, "limit_usd", gateway.LimitUSD,
		"reconciliation", rec.State, "added_usd", rec.AddedUSD)
	return gw, nil
}
