package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"magi/pkg/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConnectMCP dials each configured MCP server over stdio, lists its tools, and
// registers each into the catalog. Sessions are retained and closed by
// Registry.Close. An empty server list is a no-op.
func (r *Registry) ConnectMCP(ctx context.Context, servers []config.MCPServer) error {
	for _, s := range servers {
		if s.Command == "" {
			continue
		}
		client := mcp.NewClient(&mcp.Implementation{Name: "magi", Version: "0.1.0"}, nil)
		transport := &mcp.CommandTransport{Command: exec.CommandContext(ctx, s.Command, s.Args...)}

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("connect mcp server %q: %w", s.Name, err)
		}
		r.mu.Lock()
		r.closers = append(r.closers, session)
		r.mu.Unlock()

		lt, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			return fmt.Errorf("list tools from mcp server %q: %w", s.Name, err)
		}
		for _, t := range lt.Tools {
			schema, err := json.Marshal(t.InputSchema)
			if err != nil {
				schema = json.RawMessage(`{"type":"object"}`)
			}
			r.Register(&mcpTool{
				spec: Spec{
					Name:        t.Name,
					Description: t.Description,
					Schema:      schema,
					Source:      "mcp:" + s.Name,
					Kind:        KindAction,
				},
				toolName: t.Name,
				session:  session,
			})
		}
	}
	return nil
}

// mcpTool wraps a tool exposed by a connected MCP server.
type mcpTool struct {
	spec     Spec
	toolName string
	session  *mcp.ClientSession
}

func (t *mcpTool) Spec() Spec { return t.spec }

func (t *mcpTool) Call(ctx context.Context, args json.RawMessage) (Result, error) {
	var argv any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &argv); err != nil {
			return Result{IsError: true}, fmt.Errorf("mcp tool %q args: %w", t.toolName, err)
		}
	}
	res, err := t.session.CallTool(ctx, &mcp.CallToolParams{Name: t.toolName, Arguments: argv})
	if err != nil {
		return Result{IsError: true}, fmt.Errorf("call mcp tool %q: %w", t.toolName, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return Result{Content: b.String(), IsError: res.IsError}, nil
}
