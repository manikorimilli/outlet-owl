package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// DefaultBaseURL is OpenRouter's API; tests point the gateway at a loopback
// server instead (decided in session).
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// maxResponseBytes bounds what is read from one answer.
const maxResponseBytes = 1 << 20

// sender makes HTTP requests to OpenRouter. Replay mode has none.
type sender struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// post sends one chat completion attempt with its own timeout.
func (s *sender) post(ctx context.Context, body []byte, timeout time.Duration) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }() // read fully below; a close error changes nothing
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, data, nil
}

// keyUsage reads the key's lifetime usage in credits from GET /key, as a
// decimal string (HLD section 6).
func (s *sender) keyUsage(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/key", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET /key answered %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			Usage json.Number `json:"usage"`
		} `json:"data"`
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return "", fmt.Errorf("GET /key: unreadable body: %w", err)
	}
	if !decimal.MatchString(out.Data.Usage.String()) {
		return "", errors.New("GET /key: data.usage is not a non-negative number")
	}
	return out.Data.Usage.String(), nil
}

// decimal admits the non-negative JSON numbers PostgreSQL's numeric accepts.
var decimal = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?([eE][-+]?[0-9]+)?$`)

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int         `json:"prompt_tokens"`
		CompletionTokens int         `json:"completion_tokens"`
		Cost             json.Number `json:"cost"`
	} `json:"usage"`
}

// parsed is a 200 answer: the response and the cost to settle, empty when
// usage.cost was absent or not a number (the reserved price then stays).
type parsed struct {
	resp    Response
	costUSD string
}

func parseChat(body []byte) (parsed, error) {
	var cr chatResponse
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&cr); err != nil {
		return parsed{}, fmt.Errorf("%w: unreadable body: %v", ErrModelUnavailable, err)
	}
	if len(cr.Choices) == 0 {
		return parsed{}, fmt.Errorf("%w: the answer has no choices", ErrModelUnavailable)
	}
	p := parsed{resp: Response{Text: cr.Choices[0].Message.Content, FinishReason: cr.Choices[0].FinishReason}}
	if cr.Usage != nil {
		p.resp.InputTokens = cr.Usage.PromptTokens
		p.resp.OutputTokens = cr.Usage.CompletionTokens
		if decimal.MatchString(cr.Usage.Cost.String()) {
			p.costUSD = cr.Usage.Cost.String()
		}
	}
	return p, nil
}

// classify turns a non-200 status into the caller's error and says whether
// the attempt may be retried (429 and 5xx only; HLD section 8).
func classify(status int, body []byte) (retry bool, err error) {
	switch {
	case status == http.StatusPaymentRequired:
		var e struct {
			Error struct {
				Metadata struct {
					LimitSource string `json:"limit_source"`
				} `json:"metadata"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &e) // the source is a detail; its absence changes nothing
		if src := e.Error.Metadata.LimitSource; src != "" {
			return false, fmt.Errorf("%w (limit source %s)", ErrProviderCreditExhausted, src)
		}
		return false, ErrProviderCreditExhausted
	case status == http.StatusTooManyRequests || status >= 500:
		return true, fmt.Errorf("%w: OpenRouter answered %d", ErrModelUnavailable, status)
	default:
		return false, fmt.Errorf("%w: OpenRouter answered %d", ErrModelUnavailable, status)
	}
}
