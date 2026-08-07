package brain

import (
	"context"
	"os"
	"testing"
	"time"

	magipb "magi/gen"
	"magi/pkg/ollama"
)

// TestDeliberate_Live runs a real Deliberate through a persona + model.
//
//	OLLAMA_SMOKE=1 OLLAMA_SMOKE_MODEL=llama3.2:1b go test ./internal/brain/ -run Live -v
func TestDeliberate_Live(t *testing.T) {
	if os.Getenv("OLLAMA_SMOKE") != "1" {
		t.Skip("set OLLAMA_SMOKE=1 to run the live brain smoke test")
	}
	model := os.Getenv("OLLAMA_SMOKE_MODEL")
	if model == "" {
		model = "balthasar"
	}

	client := ollama.New("http://localhost:11434", ollama.WithNumCtx(2048), ollama.WithTimeout(90*time.Second))
	s := NewServer("balthasar", model, client)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := s.Deliberate(ctx, &magipb.DeliberateRequest{
		Scenario: "The user has been coding for 6 hours straight and is showing signs of burnout. Should Magi notify them to take a break?",
		Round:    1,
		Tools: []*magipb.ToolSpec{
			{Name: "notify_user", Description: "Send a short notification message to the user.", SchemaJson: `{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}`},
		},
	})
	if err != nil {
		t.Fatalf("Deliberate failed: %v", err)
	}

	t.Logf("brain=%s verdict=%s conf=%d action=%s args=%s latency=%dms reasoning=%q",
		resp.Brain, resp.Verdict, resp.Confidence, resp.ProposedAction.ToolName,
		resp.ProposedAction.ArgsJson, resp.LatencyMs, resp.Reasoning)

	if resp.Brain != "balthasar" {
		t.Fatalf("brain = %s, want balthasar", resp.Brain)
	}
	if resp.Verdict == magipb.Verdict_VERDICT_UNSPECIFIED {
		t.Fatalf("verdict unspecified — model did not return a valid verdict")
	}
	if resp.ProposedAction == nil || resp.ProposedAction.ArgsJson == "" {
		t.Fatalf("proposed action not populated")
	}
}
