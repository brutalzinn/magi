package tools

import (
	"context"
	"encoding/json"
	"sort"
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

	cat := r.Catalog()
	names := make(map[string]bool, len(cat))
	for _, s := range cat {
		names[s.Name] = true
	}
	for _, want := range []string{"log_decision", "notify_user"} {
		if !names[want] {
			t.Fatalf("catalog missing builtin %q; got %v", want, names)
		}
	}
	if !sort.SliceIsSorted(cat, func(i, j int) bool { return cat[i].Name < cat[j].Name }) {
		t.Fatalf("catalog not sorted by name")
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
	RegisterWeb(r)

	for _, name := range []string{"notify_user", "log_decision"} {
		if !r.IsAction(name) {
			t.Fatalf("action builtin %q IsAction() = false, want true", name)
		}
	}
	for _, name := range []string{"web_search", "fetch_url"} {
		if r.IsAction(name) {
			t.Fatalf("info tool %q IsAction() = true, want false", name)
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
