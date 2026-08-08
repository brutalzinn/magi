package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// NativeFunc is the implementation of a native (in-process) tool.
type NativeFunc func(ctx context.Context, args json.RawMessage) (Result, error)

// NativeTool adapts a Go function into a Tool.
type NativeTool struct {
	spec Spec
	fn   NativeFunc
}

func NewNativeTool(name, description string, schema json.RawMessage, fn NativeFunc) *NativeTool {
	return newNative(name, description, schema, KindAction, fn)
}

func NewNativeInfoTool(name, description string, schema json.RawMessage, fn NativeFunc) *NativeTool {
	return newNative(name, description, schema, KindInfo, fn)
}

func newNative(name, description string, schema json.RawMessage, kind Kind, fn NativeFunc) *NativeTool {
	return &NativeTool{
		spec: Spec{Name: name, Description: description, Schema: schema, Source: "native", Kind: kind},
		fn:   fn,
	}
}

func (t *NativeTool) Spec() Spec { return t.spec }

func (t *NativeTool) Call(ctx context.Context, args json.RawMessage) (Result, error) {
	return t.fn(ctx, args)
}

// Sink receives side effects from built-in tools so the gateway can surface them
// on the dashboard (and to logs). A nil sink is tolerated.
type Sink func(tool, message string)

// RegisterBuiltins adds the local-safe action tools: notify_user and log_decision.
// Network/INFO tools (see RegisterWeb) are composed separately at the call site.
func RegisterBuiltins(r *Registry, sink Sink) {
	emit := func(tool, msg string) {
		if sink != nil {
			sink(tool, msg)
		}
	}

	r.Register(NewNativeTool(
		"notify_user",
		"Send a short notification message to the user. Use for informing, not for changing anything.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string", "description": "the notification text"}
			},
			"required": ["message"]
		}`),
		func(_ context.Context, args json.RawMessage) (Result, error) {
			var p struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return Result{IsError: true}, fmt.Errorf("notify_user args: %w", err)
			}
			emit("notify_user", p.Message)
			return Result{Content: "notified: " + p.Message}, nil
		},
	))

	r.Register(NewNativeTool(
		"log_decision",
		"Record the final Magi decision with a one-line summary for the audit trail.",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"summary": {"type": "string", "description": "one-line decision summary"}
			},
			"required": ["summary"]
		}`),
		func(_ context.Context, args json.RawMessage) (Result, error) {
			var p struct {
				Summary string `json:"summary"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return Result{IsError: true}, fmt.Errorf("log_decision args: %w", err)
			}
			emit("log_decision", p.Summary)
			return Result{Content: "logged: " + p.Summary}, nil
		},
	))
}
