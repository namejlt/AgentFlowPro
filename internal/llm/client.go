package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
)

type Client struct {
	HTTP *http.Client
}

func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 0}, // per-request timeout via context
	}
}

type ChatOpts struct {
	Endpoint   string
	APIKey     string
	Model      string
	Messages   []Message
	Temp       float64
	MaxTokens  int
	Timeout    time.Duration
	Retries    int
	Stream     bool
	OnChunk    func(StreamChunk) error
}

func (c *Client) Chat(ctx context.Context, o ChatOpts) (*ChatResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= o.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		res, err := c.doOnce(ctx, o)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !isRetryable(err) {
			break
		}
	}
	return nil, lastErr
}

func backoff(n int) time.Duration {
	d := time.Duration(200*(1<<min(n, 5))) * time.Millisecond
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isRetryable(err error) bool {
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		switch ae.Code {
		case apperr.ErrUpstreamTimeout.Code, apperr.ErrRateLimited.Code, apperr.ErrUpstream.Code:
			return true
		}
	}
	return false
}

func (c *Client) doOnce(ctx context.Context, o ChatOpts) (*ChatResponse, error) {
	if o.Timeout <= 0 {
		o.Timeout = 180 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()

	body, _ := json.Marshal(ChatRequest{
		Model:       o.Model,
		Messages:    o.Messages,
		Temperature: o.Temp,
		MaxTokens:   o.MaxTokens,
		Stream:      o.Stream,
	})
	ep := strings.TrimSpace(o.Endpoint)
	if !strings.Contains(ep, "/chat/completions") {
		ep = strings.TrimRight(ep, "/") + "/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, apperr.ErrUpstreamTimeout
		}
		return nil, apperr.ErrUpstream
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, apperr.ErrUpstreamAuth
	case http.StatusTooManyRequests:
		return nil, apperr.ErrRateLimited
	case http.StatusRequestTimeout:
		return nil, apperr.ErrUpstreamTimeout
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%w: status=%d body=%s", apperr.ErrUpstream, resp.StatusCode, string(b))
	}

	if o.Stream {
		return c.readStream(resp.Body, o.OnChunk)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("empty choices")
	}
	tokens := 0
	if parsed.Usage != nil {
		tokens = parsed.Usage.TotalTokens
	}
	return &ChatResponse{Content: parsed.Choices[0].Message.Content, Tokens: tokens, RawStatus: resp.StatusCode}, nil
}

// OpenAI-compatible SSE: data: {json}\n\n
func (c *Client) readStream(r io.Reader, onChunk func(StreamChunk) error) (*ChatResponse, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var full strings.Builder
	tokens := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				if onChunk != nil {
					_ = onChunk(StreamChunk{Done: true})
				}
				break
			}
			var ev struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &ev); err != nil {
				continue
			}
			if len(ev.Choices) == 0 {
				continue
			}
			delta := ev.Choices[0].Delta.Content
			if delta != "" {
				full.WriteString(delta)
				tokens += len(delta) / 4 // rough estimate when usage not streamed
				if onChunk != nil {
					if err := onChunk(StreamChunk{Delta: delta}); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &ChatResponse{Content: full.String(), Tokens: tokens, RawStatus: 200}, nil
}
