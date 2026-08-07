package consensus

import (
	"context"
	"testing"

	magipb "magi/gen"
	"magi/pkg/config"
	"magi/pkg/tools"
)

// fakeBrain returns a scripted verdict per round (index = round-1; last entry repeats).
type fakeBrain struct {
	name    string
	scripts []*magipb.DeliberateResponse
}

func (f *fakeBrain) Name() string { return f.name }

func (f *fakeBrain) Deliberate(_ context.Context, req *magipb.DeliberateRequest) (*magipb.DeliberateResponse, error) {
	i := int(req.Round) - 1
	if i >= len(f.scripts) {
		i = len(f.scripts) - 1
	}
	r := f.scripts[i]
	r.Brain = f.name
	return r, nil
}

func aff(conf int32, tool string) *magipb.DeliberateResponse {
	return &magipb.DeliberateResponse{
		Verdict: magipb.Verdict_AFFIRMATIVE, Confidence: conf,
		ProposedAction: &magipb.ProposedAction{ToolName: tool, ArgsJson: `{"message":"go"}`},
	}
}
func neg(tool string) *magipb.DeliberateResponse {
	return &magipb.DeliberateResponse{
		Verdict: magipb.Verdict_NEGATIVE, Confidence: 50,
		ProposedAction: &magipb.ProposedAction{ToolName: tool, ArgsJson: `{}`},
	}
}

func newRegistry() *tools.Registry {
	r := tools.NewRegistry([]config.ToolPolicy{{Name: "notify_user", Mode: "auto"}})
	tools.RegisterBuiltins(r, nil)
	return r
}

func brains(m, b, c BrainClient) []BrainClient { return []BrainClient{m, b, c} }

func TestRun_ConsensusRound1_AutoExecutes(t *testing.T) {
	o := New(brains(
		&fakeBrain{name: "melchior", scripts: []*magipb.DeliberateResponse{aff(80, "notify_user")}},
		&fakeBrain{name: "balthasar", scripts: []*magipb.DeliberateResponse{aff(70, "notify_user")}},
		&fakeBrain{name: "casper", scripts: []*magipb.DeliberateResponse{aff(90, "notify_user")}},
	), newRegistry(), 3, nil)

	out, err := o.Run(context.Background(), "scenario")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "consensus" || out.Rounds != 1 {
		t.Fatalf("status=%s rounds=%d, want consensus/1", out.Status, out.Rounds)
	}
	if !out.Executed || out.Action.ToolName != "notify_user" {
		t.Fatalf("expected notify_user executed, got %+v", out)
	}
}

func TestRun_ReDeliberationThenConsensus(t *testing.T) {
	// Casper disagrees round 1 (NEGATIVE, no veto brain), agrees round 2.
	o := New(brains(
		&fakeBrain{name: "melchior", scripts: []*magipb.DeliberateResponse{aff(80, "notify_user")}},
		&fakeBrain{name: "balthasar", scripts: []*magipb.DeliberateResponse{aff(70, "notify_user")}},
		&fakeBrain{name: "casper", scripts: []*magipb.DeliberateResponse{
			{Verdict: magipb.Verdict_ABSTAIN, ProposedAction: &magipb.ProposedAction{}},
			aff(60, "notify_user"),
		}},
	), newRegistry(), 3, nil)

	out, err := o.Run(context.Background(), "scenario")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "consensus" || out.Rounds != 2 {
		t.Fatalf("status=%s rounds=%d, want consensus/2", out.Status, out.Rounds)
	}
}

func TestRun_BalthasarVeto(t *testing.T) {
	o := New(brains(
		&fakeBrain{name: "melchior", scripts: []*magipb.DeliberateResponse{aff(80, "notify_user")}},
		&fakeBrain{name: "balthasar", scripts: []*magipb.DeliberateResponse{neg("notify_user")}},
		&fakeBrain{name: "casper", scripts: []*magipb.DeliberateResponse{aff(90, "notify_user")}},
	), newRegistry(), 3, nil)

	out, err := o.Run(context.Background(), "scenario")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "veto" || out.Executed {
		t.Fatalf("status=%s executed=%v, want veto/false", out.Status, out.Executed)
	}
}

func TestRun_Deadlock(t *testing.T) {
	// Melchior always AFFIRMATIVE, casper always ABSTAIN, balthasar AFFIRMATIVE (no veto) -> never unanimous.
	o := New(brains(
		&fakeBrain{name: "melchior", scripts: []*magipb.DeliberateResponse{aff(80, "notify_user")}},
		&fakeBrain{name: "balthasar", scripts: []*magipb.DeliberateResponse{aff(70, "notify_user")}},
		&fakeBrain{name: "casper", scripts: []*magipb.DeliberateResponse{
			{Verdict: magipb.Verdict_ABSTAIN, ProposedAction: &magipb.ProposedAction{}},
		}},
	), newRegistry(), 3, nil)

	out, err := o.Run(context.Background(), "scenario")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "deadlock" || out.Rounds != 3 {
		t.Fatalf("status=%s rounds=%d, want deadlock/3", out.Status, out.Rounds)
	}
}
