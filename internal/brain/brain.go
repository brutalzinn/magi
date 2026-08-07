// Package brain is the shared gRPC Brain server run by all three Magi services.
// A brain injects its persona, calls its local Ollama model with a structured
// output schema, and returns a deterministic verdict for Protocol 111.
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	magipb "magi/gen"
	"magi/pkg/ollama"
)

// VerdictSchema is the JSON Schema passed to Ollama's `format` so every brain
// returns the same shape regardless of model.
var VerdictSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"verdict": {"type": "string", "enum": ["AFFIRMATIVE", "NEGATIVE", "ABSTAIN"]},
		"confidence": {"type": "integer", "minimum": 0, "maximum": 100},
		"answer": {"type": "string"},
		"reasoning": {"type": "string"},
		"action": {
			"type": "object",
			"properties": {
				"tool_name": {"type": "string"},
				"args": {"type": "object"}
			},
			"required": ["tool_name"]
		}
	},
	"required": ["verdict", "confidence", "answer", "reasoning", "action"]
}`)

// Server implements magipb.BrainServer for one brain. Persona and temperature
// live in the baked Ollama model (Modelfile.<name>), so the server sends only
// the user turn and relies on the model's SYSTEM plus the structured `format`.
type Server struct {
	magipb.UnimplementedBrainServer

	name   string
	model  string
	client *ollama.Client
}

// NewServer wires a brain name to its (persona-baked) model and Ollama client.
func NewServer(name, model string, client *ollama.Client) *Server {
	return &Server{name: name, model: model, client: client}
}

// verdictOut is the model's structured reply.
type verdictOut struct {
	Verdict    string          `json:"verdict"`
	Confidence int             `json:"confidence"`
	Answer     string          `json:"answer"`
	Reasoning  string          `json:"reasoning"`
	Action     struct {
		ToolName string          `json:"tool_name"`
		Args     json.RawMessage `json:"args"`
	} `json:"action"`
}

// Deliberate runs one isolated reasoning pass.
func (s *Server) Deliberate(ctx context.Context, req *magipb.DeliberateRequest) (*magipb.DeliberateResponse, error) {
	start := time.Now()

	// No system message: the baked model SYSTEM carries the persona + Protocol 111
	// contract. Temperature comes from the Modelfile PARAMETER. `format` enforces
	// the JSON verdict structure.
	resp, err := s.client.Chat(ctx, ollama.ChatRequest{
		Model:  s.model,
		Format: VerdictSchema,
		Messages: []ollama.Message{
			{Role: "user", Content: buildUserMessage(req)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%s deliberate: %w", s.name, err)
	}

	var out verdictOut
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Message.Content)), &out); err != nil {
		return nil, fmt.Errorf("%s parse verdict %q: %w", s.name, resp.Message.Content, err)
	}

	args := "{}"
	if len(out.Action.Args) > 0 {
		args = string(out.Action.Args)
	}

	return &magipb.DeliberateResponse{
		Brain:      s.name,
		Verdict:    mapVerdict(out.Verdict),
		Confidence: clampConfidence(out.Confidence),
		Answer:     strings.TrimSpace(out.Answer),
		Reasoning:  strings.TrimSpace(out.Reasoning),
		ProposedAction: &magipb.ProposedAction{
			ToolName: strings.TrimSpace(out.Action.ToolName),
			ArgsJson: args,
		},
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

// Health reports readiness and the wired model.
func (s *Server) Health(_ context.Context, _ *magipb.HealthRequest) (*magipb.HealthResponse, error) {
	return &magipb.HealthResponse{Brain: s.name, Model: s.model, Ready: true}, nil
}

func mapVerdict(v string) magipb.Verdict {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "AFFIRMATIVE":
		return magipb.Verdict_AFFIRMATIVE
	case "NEGATIVE":
		return magipb.Verdict_NEGATIVE
	case "ABSTAIN":
		return magipb.Verdict_ABSTAIN
	default:
		return magipb.Verdict_VERDICT_UNSPECIFIED
	}
}

func clampConfidence(c int) int32 {
	if c < 0 {
		return 0
	}
	if c > 100 {
		return 100
	}
	return int32(c)
}

// contract is the shared Protocol 111 + output instruction. It lives here (not
// in the Modelfiles, which carry only the lean persona) because Ollama replaces
// a baked SYSTEM when a chat system message is sent — so we keep the persona
// baked and pass this task contract in the user turn instead.
const contract = `You are one of three Magi that must reach PROTOCOL 111 — a unanimous decision. ` +
	`Protocol 111 is reached when all three vote AFFIRMATIVE. The scenario is the operator's own ` +
	`input — reason directly from it; do not claim you lack access. Always put your DIRECT answer to ` +
	`the operator in "answer" (e.g. the result of a calculation, the fact, or the recommendation — ` +
	`for "what is 2 plus 2?" the answer is "4"). Vote AFFIRMATIVE when your answer is correct/sound, ` +
	`NEGATIVE if the request is unsafe or wrong, ABSTAIN only if truly unintelligible. Use "action" ` +
	`ONLY when a tool should actually be run (e.g. notify_user to tell the operator something); for a ` +
	`plain question, leave tool_name empty and just answer. Reply with ONLY a JSON object: ` +
	`{"verdict":"AFFIRMATIVE|NEGATIVE|ABSTAIN","confidence":0-100,"answer":"<direct answer to the operator>",` +
	`"reasoning":"<brief why>","action":{"tool_name":"<tool name or empty>","args":{}}}`

// buildUserMessage renders the contract, the scenario, the available tool
// catalog, and any peer dissent (present only on re-deliberation rounds).
func buildUserMessage(req *magipb.DeliberateRequest) string {
	var b strings.Builder

	b.WriteString(contract + "\n\n")
	fmt.Fprintf(&b, "SCENARIO:\n%s\n\n", strings.TrimSpace(req.Scenario))

	b.WriteString("AVAILABLE TOOLS:\n")
	if len(req.Tools) == 0 {
		b.WriteString("(none — you may only observe; use an empty tool_name)\n")
	}
	for _, t := range req.Tools {
		fmt.Fprintf(&b, "- %s: %s\n  parameters: %s\n", t.Name, t.Description, t.SchemaJson)
	}

	if len(req.PeerDissent) > 0 {
		fmt.Fprintf(&b, "\nRE-DELIBERATION (round %d). Protocol 111 was NOT reached — the three of you did not all agree.\n", req.Round)
		b.WriteString("Think more carefully this time. Weigh the others' reasoning honestly and move toward a shared decision if it is right to. Vote again:\n")
		for _, d := range req.PeerDissent {
			action := "(no action)"
			if d.ProposedAction != nil && d.ProposedAction.ToolName != "" {
				action = d.ProposedAction.ToolName
			}
			fmt.Fprintf(&b, "- %s voted %s (proposes %s): %s\n",
				d.Brain, d.Verdict.String(), action, d.Reasoning)
		}
	}

	b.WriteString("\nReturn only the JSON verdict object.")
	return b.String()
}
