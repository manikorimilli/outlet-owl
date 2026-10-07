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
	"syscall"
	"time"
	// Embeds the IANA zone database so BRAND_TIMEZONE loads on machines and
	// images without one (phase 1 server LLD, section 7).
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/manikorimilli/outlet-owl/internal/auth"
	"github.com/manikorimilli/outlet-owl/internal/config"
	"github.com/manikorimilli/outlet-owl/internal/httpapi"
	"github.com/manikorimilli/outlet-owl/internal/outlets"
	"github.com/manikorimilli/outlet-owl/internal/store"
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
	tokens, err := auth.NewTokens(cfg.JWTSecret, nil)
	if err != nil {
		return fmt.Errorf("session tokens: %w", err)
	}

	handler := httpapi.New(httpapi.Deps{
		Logger:  logger,
		DB:      st,
		Auth:    auth.NewService(st, tokens),
		Outlets: outlets.NewService(st),
		Brand:   httpapi.Brand{Name: cfg.BrandName, Timezone: cfg.BrandTimezone.String()},
	})
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

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
