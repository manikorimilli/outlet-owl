// Command eval runs the tagging evaluation on the product owner's labelled
// reviews (US-02-004) through the gateway in the mode MODEL_GATEWAY_MODE
// sets: replay in CI and by default, live or record on purpose. It prints the
// report and exits 1 when urgent recall is below 90% (Q-017).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"github.com/manikorimilli/outlet-owl/internal/config"
	"github.com/manikorimilli/outlet-owl/internal/eval"
	"github.com/manikorimilli/outlet-owl/internal/gateway"
	"github.com/manikorimilli/outlet-owl/internal/store"
	"github.com/manikorimilli/outlet-owl/prompts"
)

func main() {
	set := flag.String("set", "testdata/eval/reviews.jsonl", "the labelled reviews, one JSON object per line")
	out := flag.String("out", "", "also write the report to this file")
	flag.Parse()
	code, err := run(*set, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "eval:", err)
	}
	os.Exit(code)
}

func run(setPath, out string) (int, error) {
	set, err := eval.Load(setPath)
	if errors.Is(err, fs.ErrNotExist) {
		return 2, fmt.Errorf("%s does not exist: the product owner collects and labels 100 real reviews "+
			"(names removed; at least 20 urgent, 5 per reason, 20 in Hindi or Hinglish) before the evaluation can run", setPath)
	}
	if err != nil {
		return 2, err
	}
	cfg, err := config.Load()
	if err != nil {
		return 2, err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return 2, err
	}
	defer st.Close()
	gw, err := gateway.New(gateway.Config{Mode: cfg.GatewayMode, Model: cfg.ModelID, APIKey: cfg.OpenRouterKey, RecordingsDir: cfg.RecordingsDir, Store: st})
	if err != nil {
		return 2, err
	}
	reg, err := prompts.Load()
	if err != nil {
		return 2, err
	}
	prompt, err := reg.Current("tagging")
	if err != nil {
		return 2, err
	}
	got, calls, err := eval.Run(ctx, gw, prompt, set)
	if err != nil {
		return 2, fmt.Errorf("after %d calls: %w", calls, err)
	}
	m := eval.Measure(set, got)
	report := eval.Report(m, prompt, gw.ModelID(), gw.Mode(), calls)
	fmt.Print(report)
	if out != "" {
		if err := os.WriteFile(out, []byte(report), 0o644); err != nil {
			return 2, err
		}
	}
	if !m.Passed {
		return 1, errors.New("urgent recall is below the 90% gate")
	}
	return 0, nil
}
