package tools

import (
	"context"
	"encoding/json"
	"testing"

	"magi/pkg/config"
)

func TestRegistry_BuiltinsExecAndMode(t *testing.T) {
	var emitted []string
	r := NewRegistry([]config.ToolPolicy{
		{Name: "notify_user", Mode: "auto"},
		{Name: "log_decision", Mode: "auto"},
	})
	RegisterBuiltins(r, func(tool, msg string) { emitted = append(emitted, tool+":"+msg) })

	// Catalog has both builtins, sorted.
	cat := r.Catalog()
	if len(cat) != 2 {
		t.Fatalf("catalog len = %d, want 2", len(cat))
	}
	if cat[0].Name != "log_decision" || cat[1].Name != "notify_user" {
		t.Fatalf("catalog not sorted as expected: %s, %s", cat[0].Name, cat[1].Name)
	}

	// Allowlisted tools resolve to auto; unlisted defaults to confirm.
	if got := r.ModeFor("notify_user"); got != ModeAuto {
		t.Fatalf("notify_user mode = %s, want auto", got)
	}
	if got := r.ModeFor("unknown_tool"); got != ModeConfirm {
		t.Fatalf("unknown mode = %s, want confirm (default)", got)
	}

	// Execute a native tool.
	res, err := r.Execute(context.Background(), "notify_user", json.RawMessage(`{"message":"hello"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.IsError || res.Content != "notified: hello" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(emitted) != 1 || emitted[0] != "notify_user:hello" {
		t.Fatalf("sink not called correctly: %v", emitted)
	}

	// Unknown tool errors.
	if _, err := r.Execute(context.Background(), "nope", nil); err == nil {
		t.Fatalf("expected error for unknown tool")
	}
}

func TestToolKind_Taxonomy(t *testing.T) {
	r := NewRegistry(nil)
	RegisterBuiltins(r, nil)

	for _, s := range r.Catalog() {
		if s.Kind != KindAction {
			t.Fatalf("builtin %q kind = %q, want %q", s.Name, s.Kind, KindAction)
		}
		if !s.IsAction() {
			t.Fatalf("builtin %q IsAction() = false, want true", s.Name)
		}
		if !r.IsAction(s.Name) {
			t.Fatalf("registry IsAction(%q) = false, want true", s.Name)
		}
	}

	info := NewNativeInfoTool("peek", "look something up", json.RawMessage(`{"type":"object"}`),
		func(context.Context, json.RawMessage) (Result, error) { return Result{Content: "x"}, nil })
	if info.Spec().Kind != KindInfo {
		t.Fatalf("info tool kind = %q, want %q", info.Spec().Kind, KindInfo)
	}
	if info.Spec().IsAction() {
		t.Fatalf("info tool IsAction() = true, want false")
	}

	if (Spec{}).IsAction() {
		t.Fatalf("unset-Kind Spec IsAction() = true, want false (only KindAction gates)")
	}
	if r.IsAction("unknown_tool") {
		t.Fatalf("registry IsAction(unknown) = true, want false")
	}
}
