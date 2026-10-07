package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/manikorimilli/outlet-owl/prompts"
)

var testPrompt = prompts.Version{Name: "reply", Number: 3, Text: "Reply in the brand's tone.", MaxTokens: 1000}

func request() Request {
	return Request{Purpose: Drafting, Prompt: testPrompt, User: "Review 7: the biryani was cold."}
}

func decodeBody(t *testing.T, r Request) chatBody {
	t.Helper()
	body, _, err := buildBody(r)
	if err != nil {
		t.Fatal(err)
	}
	var cb chatBody
	if err := json.Unmarshal(body, &cb); err != nil {
		t.Fatal(err)
	}
	return cb
}

func TestBody_ClampsMaxTokensTo1000(t *testing.T) {
	r := request()
	r.Prompt.MaxTokens = 4000

	if got := decodeBody(t, r).MaxTokens; got != 1000 {
		t.Fatalf("max_tokens = %d, want 1000 (AC-US-02-001-3)", got)
	}
}

func TestBody_ZeroMaxTokensBecomes1000(t *testing.T) {
	r := request()
	r.Prompt.MaxTokens = 0

	if got := decodeBody(t, r).MaxTokens; got != 1000 {
		t.Fatalf("max_tokens = %d, want 1000", got)
	}
}

func TestBody_SendsTheHaikuModelWithTheSystemPrompt(t *testing.T) {
	cb := decodeBody(t, request())

	if cb.Model != "anthropic/claude-haiku-4.5" {
		t.Fatalf("model = %q, want anthropic/claude-haiku-4.5 (AC-US-02-001-7)", cb.Model)
	}
	if len(cb.Messages) != 2 || cb.Messages[0].Role != "system" || cb.Messages[0].Content != testPrompt.Text || cb.Messages[1].Role != "user" {
		t.Fatalf("messages = %+v, want the prompt as system and the review as user", cb.Messages)
	}
}

func TestBody_OmitsTemperatureWhenUnset(t *testing.T) {
	body, _, _ := buildBody(request())
	if strings.Contains(string(body), "temperature") {
		t.Fatalf("body %s carries a temperature the version did not set", body)
	}
	zero := 0.0
	r := request()
	r.Prompt.Temperature = &zero
	if body, _, _ := buildBody(r); !strings.Contains(string(body), `"temperature":0`) {
		t.Fatalf("body %s lacks temperature 0", body)
	}
}

func TestWorstCasePrice_CountsBodyBytesAndMaxTokens(t *testing.T) {
	if got := worstCaseUSD(10_000, 1000); got != "0.01500000" {
		t.Fatalf("worst case = %s, want 0.01500000 (USD 0.01 input, USD 0.005 output)", got)
	}
	if got := worstCaseUSD(0, 0); got != "0.00000000" {
		t.Fatalf("worst case = %s, want zero", got)
	}
}

func TestParseMode_EmptyIsReplay(t *testing.T) {
	if m, err := ParseMode(""); err != nil || m != Replay {
		t.Fatalf("ParseMode(\"\") = %q, %v; want replay", m, err)
	}
	if _, err := ParseMode("Live"); err == nil {
		t.Fatal("an unknown mode must be refused")
	}
}
