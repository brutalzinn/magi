// Package ollama is a minimal, low-footprint client for a local Ollama daemon.
// It targets /api/chat with tool (function) calling and optional structured
// output, and deliberately avoids importing the large github.com/ollama/ollama
// module to keep the binary and memory footprint small.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to a local Ollama daemon.
type Client struct {
	baseURL   string
	http      *http.Client
	numCtx    int
	keepAlive string
}

// Option configures a Client.
type Option func(*Client)

// WithNumCtx sets the context window (num_ctx) applied to every request unless
// overridden per-call. Tight values conserve RAM.
func WithNumCtx(n int) Option { return func(c *Client) { c.numCtx = n } }

// WithKeepAlive sets how long a model stays resident after a request (e.g. "5m").
func WithKeepAlive(s string) Option { return func(c *Client) { c.keepAlive = s } }

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

// New creates a client for baseURL (e.g. http://localhost:11434).
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:   baseURL,
		http:      &http.Client{Timeout: 120 * time.Second},
		numCtx:    2048,
		keepAlive: "5m",
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Tool advertises a callable function to the model.
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema object
}

// Message is one chat message.
type Message struct {
	Role      string     `json:"role"` // system|user|assistant|tool
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"` // for role=tool results
}

// ToolCall is a function invocation the model requested.
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"` // JSON object of args
}

// ChatRequest is the /api/chat payload.
type ChatRequest struct {
	Model     string         `json:"model"`
	Messages  []Message      `json:"messages"`
	Tools     []Tool         `json:"tools,omitempty"`
	Format    json.RawMessage `json:"format,omitempty"` // "json" or a JSON Schema
	Stream    bool           `json:"stream"`
	KeepAlive string         `json:"keep_alive,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}

// ChatResponse is the non-streamed /api/chat reply.
type ChatResponse struct {
	Model           string  `json:"model"`
	Message         Message `json:"message"`
	Done            bool    `json:"done"`
	TotalDuration   int64   `json:"total_duration"`
	PromptEvalCount int     `json:"prompt_eval_count"`
	EvalCount       int     `json:"eval_count"`
}

// Chat performs a single non-streamed chat completion. Client defaults for
// num_ctx and keep_alive are applied unless the request already sets them.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	if req.Options == nil {
		req.Options = map[string]any{}
	}
	if _, ok := req.Options["num_ctx"]; !ok {
		req.Options["num_ctx"] = c.numCtx
	}
	if req.KeepAlive == "" {
		req.KeepAlive = c.keepAlive
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama %s: %s", resp.Status, string(data))
	}

	var out ChatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	return &out, nil
}
