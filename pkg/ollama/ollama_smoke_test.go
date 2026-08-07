package ollama

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestChatToolCalling_Live exercises the real /api/chat tool-calling path.
// Skipped unless OLLAMA_SMOKE=1 and a model is available.
//
//	OLLAMA_SMOKE=1 OLLAMA_SMOKE_MODEL=llama3.2:1b go test ./pkg/ollama/ -run Live -v
func TestChatToolCalling_Live(t *testing.T) {
	if os.Getenv("OLLAMA_SMOKE") != "1" {
		t.Skip("set OLLAMA_SMOKE=1 to run the live Ollama smoke test")
	}
	model := os.Getenv("OLLAMA_SMOKE_MODEL")
	if model == "" {
		model = "llama3.2:1b"
	}

	c := New("http://localhost:11434", WithNumCtx(2048), WithTimeout(90*time.Second))

	tool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "log_decision",
			Description: "Record a decision with a short summary.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"summary": {"type": "string", "description": "one-line decision summary"}
				},
				"required": ["summary"]
			}`),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := c.Chat(ctx, ChatRequest{
		Model: model,
		Tools: []Tool{tool},
		Messages: []Message{
			{Role: "system", Content: "You are a decision engine. When asked to record a decision, call the log_decision tool."},
			{Role: "user", Content: "Record the decision: approve the deployment."},
		},
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	t.Logf("content=%q tool_calls=%d eval=%d ctx=%d", resp.Message.Content, len(resp.Message.ToolCalls), resp.EvalCount, resp.PromptEvalCount)

	if !resp.Done {
		t.Fatalf("expected done=true")
	}
	if len(resp.Message.ToolCalls) == 0 && resp.Message.Content == "" {
		t.Fatalf("expected either a tool_call or content, got neither")
	}
	for _, tc := range resp.Message.ToolCalls {
		t.Logf("tool_call name=%s args=%s", tc.Function.Name, string(tc.Function.Arguments))
	}
}
