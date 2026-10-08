package gateway

import (
	"encoding/json"
	"fmt"

	"github.com/manikorimilli/outlet-owl/prompts"
)

// Model is the default model every call uses: Claude Haiku 4.5 under
// OpenRouter's identifier, with no fallback (Q-016, AC-US-02-001-7). MODEL_ID
// replaces it for every call (ADR-0009); there is still one model per run.
const Model = "anthropic/claude-haiku-4.5"

// MaxTokensCap is the most any request asks for (REQ-032).
const MaxTokensCap = prompts.MaxTokensCap

// LimitUSD is the recorded total above which every call is refused
// (REQ-031, Q-015).
const LimitUSD = "8"

// Prices in units of 1e-8 USD per token (the numeric(12,8) scale): USD 1 and
// USD 5 per million input and output tokens (GenAI design section 5; an
// assumption until checked on the model page).
const (
	inputUnitsPerToken  = 100
	outputUnitsPerToken = 500
	unitsPerUSD         = 100_000_000
)

// Mode is how the gateway answers: live, record (live and save the response)
// or replay (only from saved responses, never the network).
type Mode string

const (
	Live   Mode = "live"
	Record Mode = "record"
	Replay Mode = "replay"
)

// ParseMode accepts the three modes; an empty value is replay (decided in
// session: nothing spends money unless the operator chooses it).
func ParseMode(s string) (Mode, error) {
	switch m := Mode(s); m {
	case "":
		return Replay, nil
	case Live, Record, Replay:
		return m, nil
	}
	return "", fmt.Errorf("must be live, record or replay, not %q", s)
}

// Purpose names the caller; it is the budget.model_call_purpose of the row.
type Purpose string

const (
	Tagging    Purpose = "tagging"
	Drafting   Purpose = "drafting"
	Evaluation Purpose = "evaluation"
	ToneCheck  Purpose = "tone_check"
)

func (p Purpose) valid() bool {
	switch p {
	case Tagging, Drafting, Evaluation, ToneCheck:
		return true
	}
	return false
}

// Request is one model call: the prompt version is the system message and
// User the user message.
type Request struct {
	Purpose Purpose
	Prompt  prompts.Version
	User    string
}

// Response is the model's answer, the same in every mode.
type Response struct {
	Text         string
	FinishReason string
	InputTokens  int
	OutputTokens int
	Mode         Mode
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatBody is the request body. Its field order is fixed by the struct, and
// the recording key is the hash of these exact bytes.
type chatBody struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature *float64  `json:"temperature,omitempty"`
	Reasoning   reasoning `json:"reasoning"`
}

// reasoning is always sent off: max_tokens (1000, REQ-032) covers reasoning
// and answer together, and a reasoning model otherwise spends it all
// thinking and returns an empty, cut-off answer (ADR-0009).
type reasoning struct {
	Enabled bool `json:"enabled"`
}

// clampMaxTokens caps what a caller asks for at 1000; zero means 1000
// (AC-US-02-001-3).
func clampMaxTokens(n int) int {
	if n <= 0 || n > MaxTokensCap {
		return MaxTokensCap
	}
	return n
}

// buildBody returns the request body for model and the max_tokens it carries.
func buildBody(r Request, model string) ([]byte, int, error) {
	maxTokens := clampMaxTokens(r.Prompt.MaxTokens)
	body, err := json.Marshal(chatBody{
		Model: model,
		Messages: []message{
			{Role: "system", Content: r.Prompt.Text},
			{Role: "user", Content: r.User},
		},
		MaxTokens:   maxTokens,
		Temperature: r.Prompt.Temperature,
		Reasoning:   reasoning{Enabled: false},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("gateway: build the request body: %w", err)
	}
	return body, maxTokens, nil
}

// worstCaseUSD is the price reserved before a call: every body byte counts as
// one input token (a token is at least one byte), plus max_tokens of output.
func worstCaseUSD(bodyBytes, maxTokens int) string {
	units := int64(bodyBytes)*inputUnitsPerToken + int64(maxTokens)*outputUnitsPerToken
	return fmt.Sprintf("%d.%08d", units/unitsPerUSD, units%unitsPerUSD)
}
