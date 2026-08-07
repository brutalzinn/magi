// Package consensus is the Protocol 111 orchestrator. It fans a scenario out to
// the three brains, computes unanimity, runs re-deliberation rounds with peer
// dissent, honours Balthasar's protective veto, and auto-executes the agreed
// tool on unanimity.
package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	magipb "magi/gen"
	"magi/pkg/tools"
)

// BrainClient is one deliberating brain (real gRPC or a test fake).
type BrainClient interface {
	Name() string
	Deliberate(ctx context.Context, req *magipb.DeliberateRequest) (*magipb.DeliberateResponse, error)
}

// ToolExecutor is the subset of *tools.Registry the orchestrator needs.
type ToolExecutor interface {
	Catalog() []tools.Spec
	Has(name string) bool
	ModeFor(name string) tools.ExecMode
	Execute(ctx context.Context, name string, args json.RawMessage) (tools.Result, error)
}

// Event is a single telemetry step, forwarded to the dashboard over websocket.
type Event struct {
	Type       string `json:"type"` // round_start|verdict|consensus|veto|deadlock|action|final
	Round      int    `json:"round,omitempty"`
	Brain      string `json:"brain,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
	Confidence int32  `json:"confidence"`
	Answer     string `json:"answer,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
	Tool       string `json:"tool,omitempty"`
	Args       string `json:"args,omitempty"`
	Message    string `json:"message,omitempty"`
	Payload    string `json:"payload,omitempty"` // raw request body for the packet viewer
}

// Emit is the sink for events (nil is tolerated).
type Emit func(Event)

// Outcome is the final result of a deliberation.
type Outcome struct {
	Status     string                        `json:"status"` // consensus|veto|deadlock
	Rounds     int                           `json:"rounds"`
	Verdicts   []*magipb.DeliberateResponse  `json:"verdicts"` // last round
	Answer     string                        `json:"answer,omitempty"` // agreed answer to the operator
	Action     *magipb.ProposedAction        `json:"action,omitempty"`
	Executed   bool                          `json:"executed"`
	ExecResult string                        `json:"exec_result,omitempty"`
	Summary    string                        `json:"summary"` // human reply for chat
}

// Orchestrator runs Protocol 111.
type Orchestrator struct {
	brains    []BrainClient // fixed order: melchior, balthasar, casper
	tools     ToolExecutor
	maxRounds int
	emit      Emit
}

// New builds an orchestrator. maxRounds < 1 is treated as 1.
func New(brains []BrainClient, tx ToolExecutor, maxRounds int, emit Emit) *Orchestrator {
	if maxRounds < 1 {
		maxRounds = 1
	}
	if emit == nil {
		emit = func(Event) {}
	}
	return &Orchestrator{brains: brains, tools: tx, maxRounds: maxRounds, emit: emit}
}

// Run executes the full Protocol 111 flow for a scenario.
func (o *Orchestrator) Run(ctx context.Context, scenario string) (*Outcome, error) {
	catalog := toToolSpecs(o.tools.Catalog())

	var last []*magipb.DeliberateResponse
	var dissent []*magipb.PeerDissent

	for round := 1; round <= o.maxRounds; round++ {
		o.emit(Event{Type: "round_start", Round: round})

		resps, err := o.deliberateAll(ctx, scenario, round, catalog, dissent)
		if err != nil {
			return nil, err
		}
		last = resps

		if allAffirmative(resps) {
			answer := agreedAnswer(resps)
			action := agreedAction(resps)
			ev := Event{Type: "consensus", Round: round, Answer: answer}
			if action != nil {
				ev.Tool, ev.Args = action.ToolName, action.ArgsJson
			}
			o.emit(ev)
			return o.finish(ctx, "consensus", round, resps, answer, action), nil
		}

		if vetoed(resps) {
			o.emit(Event{Type: "veto", Round: round, Brain: "balthasar",
				Message: "protective veto — action withheld"})
			return o.finish(ctx, "veto", round, resps, "", nil), nil
		}

		dissent = toPeerDissent(resps) // feed back for the next round
	}

	o.emit(Event{Type: "deadlock", Round: o.maxRounds, Message: "no unanimity"})
	return o.finish(ctx, "deadlock", o.maxRounds, last, "", nil), nil
}

// deliberateAll fans out to every brain concurrently and preserves order.
func (o *Orchestrator) deliberateAll(ctx context.Context, scenario string, round int, catalog []*magipb.ToolSpec, dissent []*magipb.PeerDissent) ([]*magipb.DeliberateResponse, error) {
	resps := make([]*magipb.DeliberateResponse, len(o.brains))
	errs := make([]error, len(o.brains))
	var wg sync.WaitGroup
	for i, b := range o.brains {
		wg.Add(1)
		go func(i int, b BrainClient) {
			defer wg.Done()
			// Packet out: gateway -> brain (emitted the instant we dispatch), with
			// the raw request body for the packet viewer.
			o.emit(Event{
				Type: "dispatch", Round: round, Brain: b.Name(),
				Payload: requestPayload(scenario, round, catalog, dissent),
			})
			r, err := b.Deliberate(ctx, &magipb.DeliberateRequest{
				Scenario: scenario, Round: int32(round), Tools: catalog, PeerDissent: dissent,
			})
			resps[i], errs[i] = r, err
			if err != nil {
				return
			}
			// Packet back: brain -> gateway (emitted the instant this brain returns,
			// independently of the others).
			o.emit(Event{
				Type: "verdict", Round: round, Brain: r.Brain,
				Verdict: r.Verdict.String(), Confidence: r.Confidence,
				Answer: r.Answer, Reasoning: r.Reasoning,
				Tool: r.ProposedAction.GetToolName(), Args: r.ProposedAction.GetArgsJson(),
			})
		}(i, b)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("brain %s: %w", o.brains[i].Name(), err)
		}
	}
	return resps, nil
}

// finish builds the outcome and executes the agreed action when consensus was reached.
func (o *Orchestrator) finish(ctx context.Context, status string, rounds int, resps []*magipb.DeliberateResponse, answer string, action *magipb.ProposedAction) *Outcome {
	out := &Outcome{Status: status, Rounds: rounds, Verdicts: resps, Answer: answer, Action: action}

	switch status {
	case "consensus":
		lead := answer
		if lead == "" {
			lead = "The three Magi reached unanimous agreement."
		}
		if action != nil && action.ToolName != "" && o.tools.Has(action.ToolName) {
			if o.tools.ModeFor(action.ToolName) == tools.ModeAuto {
				res, err := o.tools.Execute(ctx, action.ToolName, json.RawMessage(action.ArgsJson))
				out.Executed = err == nil && !res.IsError
				if err != nil {
					out.ExecResult = "error: " + err.Error()
				} else {
					out.ExecResult = res.Content
				}
				out.Summary = fmt.Sprintf("%s\n\n(executed %s: %s)", lead, action.ToolName, out.ExecResult)
			} else {
				out.Summary = fmt.Sprintf("%s\n\n(action %s awaits confirmation)", lead, action.ToolName)
			}
			o.emit(Event{Type: "action", Tool: action.ToolName, Args: action.ArgsJson,
				Message: out.ExecResult})
		} else {
			out.Summary = lead
		}
	case "veto":
		out.Summary = "Balthasar veto — the action was withheld to protect you. " + balthasarReason(resps)
	default: // deadlock
		out.Summary = fmt.Sprintf("No unanimous decision after %d round(s). Best guidance: %s",
			rounds, bestReasoning(resps))
	}

	o.emit(Event{Type: "final", Round: rounds, Answer: answer, Message: out.Summary})
	return out
}

// allAffirmative reports Protocol 111: every brain voted AFFIRMATIVE.
func allAffirmative(resps []*magipb.DeliberateResponse) bool {
	if len(resps) == 0 {
		return false
	}
	for _, r := range resps {
		if r.Verdict != magipb.Verdict_AFFIRMATIVE {
			return false
		}
	}
	return true
}

// agreedAction returns an action only when all brains proposed the SAME non-empty
// tool (args from the highest-confidence brain); otherwise nil (answer-only consensus).
func agreedAction(resps []*magipb.DeliberateResponse) *magipb.ProposedAction {
	if len(resps) == 0 {
		return nil
	}
	tool := resps[0].ProposedAction.GetToolName()
	if tool == "" {
		return nil
	}
	best := resps[0]
	for _, r := range resps {
		if r.ProposedAction.GetToolName() != tool {
			return nil
		}
		if r.Confidence > best.Confidence {
			best = r
		}
	}
	return best.ProposedAction
}

// agreedAnswer returns the highest-confidence non-empty answer among the brains.
func agreedAnswer(resps []*magipb.DeliberateResponse) string {
	var best *magipb.DeliberateResponse
	for _, r := range resps {
		if r.Answer == "" {
			continue
		}
		if best == nil || r.Confidence > best.Confidence {
			best = r
		}
	}
	if best == nil {
		return ""
	}
	return best.Answer
}

// vetoed reports a held Balthasar NEGATIVE (protective veto).
func vetoed(resps []*magipb.DeliberateResponse) bool {
	for _, r := range resps {
		if r.Brain == "balthasar" && r.Verdict == magipb.Verdict_NEGATIVE {
			return true
		}
	}
	return false
}

// bestReasoning returns the reasoning of the highest-confidence brain that gave one.
func bestReasoning(resps []*magipb.DeliberateResponse) string {
	var best *magipb.DeliberateResponse
	for _, r := range resps {
		if r.Reasoning == "" {
			continue
		}
		if best == nil || r.Confidence > best.Confidence {
			best = r
		}
	}
	if best == nil {
		return "the three could not converge on a next step."
	}
	return best.Reasoning
}

func balthasarReason(resps []*magipb.DeliberateResponse) string {
	for _, r := range resps {
		if r.Brain == "balthasar" {
			return r.Reasoning
		}
	}
	return ""
}

func toPeerDissent(resps []*magipb.DeliberateResponse) []*magipb.PeerDissent {
	out := make([]*magipb.PeerDissent, 0, len(resps))
	for _, r := range resps {
		out = append(out, &magipb.PeerDissent{
			Brain: r.Brain, Verdict: r.Verdict, Reasoning: r.Reasoning,
			ProposedAction: r.ProposedAction,
		})
	}
	return out
}

// requestPayload renders the gateway->brain request as a compact JSON body for
// the packet viewer.
func requestPayload(scenario string, round int, catalog []*magipb.ToolSpec, dissent []*magipb.PeerDissent) string {
	names := make([]string, 0, len(catalog))
	for _, t := range catalog {
		names = append(names, t.Name)
	}
	body, _ := json.Marshal(map[string]any{
		"round":        round,
		"scenario":     scenario,
		"tools":        names,
		"peer_dissent": len(dissent),
	})
	return string(body)
}

func toToolSpecs(specs []tools.Spec) []*magipb.ToolSpec {
	out := make([]*magipb.ToolSpec, 0, len(specs))
	for _, s := range specs {
		out = append(out, &magipb.ToolSpec{
			Name: s.Name, Description: s.Description, SchemaJson: string(s.Schema),
		})
	}
	return out
}
