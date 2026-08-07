// Package tools is the Magi tool layer: a unified catalog of executable tools
// (native Go functions + tools imported from MCP servers), each with an
// execution mode (auto|confirm) resolved from magi.yaml. The gateway advertises
// the catalog to the brains and, on Protocol 111 unanimity, executes the agreed
// tool through this registry.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"magi/pkg/config"
)

// ExecMode controls whether an agreed tool runs automatically or needs confirmation.
type ExecMode string

const (
	ModeAuto    ExecMode = "auto"
	ModeConfirm ExecMode = "confirm"
)

// Spec is the public description of a tool, advertised to the brains.
type Spec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"` // JSON Schema of parameters
	Source      string          `json:"source"` // "native" or "mcp:<server>"
}

// Result is the outcome of executing a tool.
type Result struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// Tool is an executable capability.
type Tool interface {
	Spec() Spec
	Call(ctx context.Context, args json.RawMessage) (Result, error)
}

// Registry holds the unified tool catalog and per-tool execution policy.
type Registry struct {
	mu          sync.RWMutex
	tools       map[string]Tool
	modes       map[string]ExecMode
	defaultMode ExecMode
	closers     []io.Closer
}

// NewRegistry builds a registry seeded with the allowlist policies from config.
// Tools not named in the allowlist default to ModeConfirm (safer).
func NewRegistry(policies []config.ToolPolicy) *Registry {
	modes := make(map[string]ExecMode, len(policies))
	for _, p := range policies {
		modes[p.Name] = normalizeMode(p.Mode)
	}
	return &Registry{
		tools:       make(map[string]Tool),
		modes:       modes,
		defaultMode: ModeConfirm,
	}
}

func normalizeMode(s string) ExecMode {
	if strings.EqualFold(s, string(ModeAuto)) {
		return ModeAuto
	}
	return ModeConfirm
}

// Register adds a tool. A later registration with the same name overrides the earlier.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Spec().Name] = t
}

// Catalog returns all tool specs, sorted by name for stable display.
func (r *Registry) Catalog() []Spec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Spec, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Spec())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Has reports whether a tool with the given name is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// ModeFor returns the execution mode for a tool (defaultMode if unlisted).
func (r *Registry) ModeFor(name string) ExecMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if m, ok := r.modes[name]; ok {
		return m
	}
	return r.defaultMode
}

// Execute runs a tool by name with JSON arguments.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) (Result, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return Result{IsError: true}, fmt.Errorf("tool %q not found", name)
	}
	return t.Call(ctx, args)
}

// Close shuts down any MCP sessions the registry owns.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var firstErr error
	for _, c := range r.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	r.closers = nil
	return firstErr
}
