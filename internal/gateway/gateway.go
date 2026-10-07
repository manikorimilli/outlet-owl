package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Store is the budget log in PostgreSQL (internal/store). The running total
// is read there before every live call, never held in memory (tenet 2).
type Store interface {
	ReserveModelCall(ctx context.Context, purpose, model string, promptVersion int, reservedUSD, limitUSD string) (int64, error)
	SettleModelCall(ctx context.Context, id int64, inputTokens, outputTokens int, costUSD string) (bool, error)
	FailModelCall(ctx context.Context, id int64) (bool, error)
	RunningTotalUSD(ctx context.Context) (string, error)
	InsertReconciliation(ctx context.Context, providerUsageUSD string) (addedUSD string, added bool, err error)
}

// Config sets up a gateway. AttemptTimeout, Backoff and BaseURL default to
// the HLD's numbers and OpenRouter; tests shorten them.
type Config struct {
	Mode          Mode
	APIKey        string
	BaseURL       string
	RecordingsDir string
	Store         Store
	Logger        *slog.Logger
	// AttemptTimeout bounds one HTTP attempt (30 s, HLD section 8).
	AttemptTimeout time.Duration
	// Backoff is the wait before each retry of a 429 or 5xx; its length is
	// the number of retries (2 s and 4 s, HLD section 8).
	Backoff []time.Duration
}

// settleTimeout bounds the cost-row write after an attempt.
const settleTimeout = 5 * time.Second

// Gateway is safe for concurrent use; it keeps no state between calls.
type Gateway struct {
	cfg    Config
	sender *sender // nil in replay mode: replay never builds an HTTP client
}

// New checks the configuration and, outside replay mode, builds the HTTP
// client.
func New(cfg Config) (*Gateway, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.DiscardHandler)
	}
	if cfg.AttemptTimeout == 0 {
		cfg.AttemptTimeout = 30 * time.Second
	}
	if cfg.Backoff == nil {
		cfg.Backoff = []time.Duration{2 * time.Second, 4 * time.Second}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.RecordingsDir == "" {
		cfg.RecordingsDir = "testdata/recordings"
	}
	g := &Gateway{cfg: cfg}
	switch cfg.Mode {
	case Replay:
		return g, nil
	case Live, Record:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("gateway: %s mode needs OPENROUTER_API_KEY", cfg.Mode)
		}
		if cfg.Store == nil {
			return nil, errors.New("gateway: live calls need the budget store")
		}
		g.sender = &sender{client: &http.Client{}, baseURL: cfg.BaseURL, apiKey: cfg.APIKey}
		return g, nil
	}
	return nil, fmt.Errorf("gateway: unknown mode %q", cfg.Mode)
}

// Mode reports how this gateway answers.
func (g *Gateway) Mode() Mode { return g.cfg.Mode }

// Complete makes one model call. In live and record mode it reserves the
// worst-case price, sends, and settles or fails the cost row of every
// attempt; in replay mode it answers from a recording and writes no row.
func (g *Gateway) Complete(ctx context.Context, r Request) (Response, error) {
	if !r.Purpose.valid() {
		return Response{}, fmt.Errorf("gateway: unknown purpose %q", r.Purpose)
	}
	body, maxTokens, err := buildBody(r)
	if err != nil {
		return Response{}, err
	}
	if g.cfg.Mode == Replay {
		return g.replay(r, body)
	}
	resp, raw, err := g.live(ctx, r, body, maxTokens)
	if err != nil {
		return Response{}, fmt.Errorf("%s call: %w", r.Purpose, err)
	}
	if g.cfg.Mode == Record {
		if err := g.save(r, body, raw); err != nil {
			return Response{}, err
		}
	}
	return resp, nil
}

// live runs the first attempt and at most len(Backoff) retries of a 429 or
// 5xx. Each attempt reserves its own row, so a retry re-checks the budget.
func (g *Gateway) live(ctx context.Context, r Request, body []byte, maxTokens int) (Response, []byte, error) {
	reserved := worstCaseUSD(len(body), maxTokens)
	for attempt := 0; ; attempt++ {
		id, err := g.cfg.Store.ReserveModelCall(ctx, string(r.Purpose), Model, r.Prompt.Number, reserved, LimitUSD)
		if err != nil {
			g.logCall(r, attempt, "refused", 0, reserved, "", Response{}, 0)
			return Response{}, nil, err
		}
		start := time.Now()
		status, raw, sendErr := g.sender.post(ctx, body, g.cfg.AttemptTimeout)
		took := time.Since(start)
		if sendErr == nil && status == http.StatusOK {
			p, perr := parseChat(raw)
			if perr == nil {
				cost := p.costUSD
				if cost == "" {
					cost = reserved // no usage.cost: the reserved price stays (HLD section 6)
				}
				g.settle(ctx, id, p.resp.InputTokens, p.resp.OutputTokens, cost)
				p.resp.Mode = g.cfg.Mode
				g.logCall(r, attempt, "settled", status, reserved, cost, p.resp, took)
				return p.resp, raw, nil
			}
			sendErr = perr
		}
		g.fail(ctx, id)
		g.logCall(r, attempt, "failed", status, reserved, "", Response{}, took)
		if sendErr != nil {
			// A timeout or network error may have been billed: never resent.
			if errors.Is(sendErr, ErrModelUnavailable) {
				return Response{}, nil, sendErr
			}
			return Response{}, nil, fmt.Errorf("%w: %v", ErrModelUnavailable, sendErr)
		}
		retry, cerr := classify(status, raw)
		if !retry || attempt >= len(g.cfg.Backoff) {
			return Response{}, nil, cerr
		}
		select {
		case <-ctx.Done():
			return Response{}, nil, fmt.Errorf("%w: %v", ErrModelUnavailable, ctx.Err())
		case <-time.After(g.cfg.Backoff[attempt]):
		}
	}
}

// settle and fail run on a context the caller cannot cancel, so a caller that
// gives up never leaves a paid call unlogged. If the write itself fails, the
// row stays reserved and keeps counting at its reserved price.
func (g *Gateway) settle(ctx context.Context, id int64, in, out int, cost string) {
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleTimeout)
	defer cancel()
	if _, err := g.cfg.Store.SettleModelCall(wctx, id, in, out, cost); err != nil {
		g.cfg.Logger.Error("model call not settled; it stays counted at its reserved price", "call_id", id, "err", err)
	}
}

func (g *Gateway) fail(ctx context.Context, id int64) {
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleTimeout)
	defer cancel()
	if _, err := g.cfg.Store.FailModelCall(wctx, id); err != nil {
		g.cfg.Logger.Error("model call not marked failed; it stays counted at its reserved price", "call_id", id, "err", err)
	}
}

// logCall writes the one line per model call (HLD section 10). It carries
// ids, tokens and cost, never the prompt or the user message.
func (g *Gateway) logCall(r Request, attempt int, outcome string, status int, reserved, settled string, resp Response, took time.Duration) {
	g.cfg.Logger.Info("model call",
		"purpose", string(r.Purpose), "prompt", r.Prompt.Name, "prompt_version", r.Prompt.Number,
		"mode", string(g.cfg.Mode), "attempt", attempt+1, "outcome", outcome, "status", status,
		"input_tokens", resp.InputTokens, "output_tokens", resp.OutputTokens,
		"reserved_usd", reserved, "settled_usd", settled, "duration_ms", took.Milliseconds())
}
