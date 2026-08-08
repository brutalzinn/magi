package brain

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	magipb "magi/gen"
	"magi/pkg/ollama"
)

// TestDeliberate_ColdStatelessRequest proves each Deliberate sends a fresh,
// single-turn request with no conversation history, and that a prior scenario
// never leaks into a later one — the brains always think cold on new input.
func TestDeliberate_ColdStatelessRequest(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":{"role":"assistant","content":"{\"verdict\":\"AFFIRMATIVE\",\"confidence\":80,\"answer\":\"ok\",\"reasoning\":\"r\",\"action\":{\"tool_name\":\"\"}}"},"done":true}`))
	}))
	defer srv.Close()

	s := NewServer("melchior", "melchior", ollama.New(srv.URL))

	first := "SECRET-ALPHA scenario one"
	second := "scenario two entirely different"
	for _, scenario := range []string{first, second} {
		if _, err := s.Deliberate(context.Background(), &magipb.DeliberateRequest{Scenario: scenario, Round: 1}); err != nil {
			t.Fatalf("deliberate %q: %v", scenario, err)
		}
	}

	if len(bodies) != 2 {
		t.Fatalf("expected 2 ollama requests, got %d", len(bodies))
	}

	for i, body := range bodies {
		msgs, ok := body["messages"].([]any)
		if !ok || len(msgs) != 1 {
			t.Fatalf("request %d: want exactly 1 message (no history), got %v", i, body["messages"])
		}
		m := msgs[0].(map[string]any)
		if m["role"] != "user" {
			t.Fatalf("request %d: role = %v, want user", i, m["role"])
		}
		if _, hasCtx := body["context"]; hasCtx {
			t.Fatalf("request %d: carries a context field — model must be stateless", i)
		}
	}

	// Cold thinking: the second request must not carry the first scenario.
	secondContent := bodies[1]["messages"].([]any)[0].(map[string]any)["content"].(string)
	if strings.Contains(secondContent, "SECRET-ALPHA") {
		t.Fatalf("prior interaction leaked into the next request: %q", secondContent)
	}
}
