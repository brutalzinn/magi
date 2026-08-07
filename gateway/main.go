// Magi Gateway — gRPC client orchestrator + Fiber HTTP/WebSocket server.
// Serves the dashboard, accepts chat/scenario input, runs Protocol 111 across
// the three brains, and streams live telemetry to the browser over /ws.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"magi/internal/consensus"
	"magi/pkg/config"
	"magi/pkg/tools"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

var brainOrder = []string{"melchior", "balthasar", "casper"}

func main() {
	cfg, err := config.Load(configPath())
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}

	h := newHub()

	// Tool layer: builtins + any configured MCP servers.
	reg := tools.NewRegistry(cfg.Tools.Allowlist)
	tools.RegisterBuiltins(reg, func(tool, msg string) {
		h.broadcast(consensus.Event{Type: "notify", Tool: tool, Message: msg})
	})
	if err := reg.ConnectMCP(context.Background(), cfg.Tools.MCPServers); err != nil {
		log.Printf("gateway: MCP connect (continuing without): %v", err)
	}
	defer reg.Close()

	// Dial the three brains in fixed order.
	brains := make([]*consensus.GRPCBrain, 0, 3)
	clients := make([]consensus.BrainClient, 0, 3)
	for _, name := range brainOrder {
		bc, err := cfg.Brain(name)
		if err != nil {
			log.Fatalf("gateway: %v", err)
		}
		gb, err := consensus.DialBrain(name, bc.GRPCAddr)
		if err != nil {
			log.Fatalf("gateway: %v", err)
		}
		defer gb.Close()
		brains = append(brains, gb)
		clients = append(clients, gb)
	}

	orch := consensus.New(clients, reg, cfg.Consensus.MaxRounds, func(e consensus.Event) {
		h.broadcast(e)
	})

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	// WebSocket telemetry stream.
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		h.add(c)
		defer h.remove(c)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))

	// Chat / scenario endpoint — runs one full Protocol 111 deliberation.
	app.Post("/api/deliberate", func(c *fiber.Ctx) error {
		var body struct {
			Scenario string `json:"scenario"`
		}
		if err := c.BodyParser(&body); err != nil || body.Scenario == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "scenario required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), cfg.RequestTimeoutDuration())
		defer cancel()
		out, err := orch.Run(ctx, body.Scenario)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	})

	// Brain node status for the graph.
	app.Get("/api/status", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		type node struct {
			Name  string `json:"name"`
			Model string `json:"model"`
			Ready bool   `json:"ready"`
		}
		out := make([]node, 0, len(brains))
		for _, b := range brains {
			n := node{Name: b.Name()}
			if hr, err := b.Health(ctx); err == nil {
				n.Model, n.Ready = hr.Model, hr.Ready
			}
			out = append(out, n)
		}
		return c.JSON(out)
	})

	// Tool catalog (for display).
	app.Get("/api/tools", func(c *fiber.Ctx) error {
		return c.JSON(reg.Catalog())
	})

	app.Static("/", staticDir(), fiber.Static{CacheDuration: -1})

	addr := cfg.Gateway.HTTPAddr
	if v := os.Getenv("MAGI_HTTP_ADDR"); v != "" {
		addr = v
	}
	log.Printf("[gateway] dashboard on %s (brains: melchior/balthasar/casper, max_rounds=%d)",
		addr, cfg.Consensus.MaxRounds)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("gateway: %v", err)
	}
}

func configPath() string {
	if v := os.Getenv("MAGI_CONFIG"); v != "" {
		return v
	}
	return "magi.yaml"
}

func staticDir() string {
	if v := os.Getenv("MAGI_STATIC"); v != "" {
		return v
	}
	return "gateway/static"
}
