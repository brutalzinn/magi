# Magi — Roadmap
> Updated 2026-08-07. Focus: local tripartite autonomous decision engine (Protocol 111). Terse; `git log` holds detail.

## What is Magi
A tripartite autonomous decision engine (Neon Genesis Evangelion Magi). Reads the user's current
scenario; when its three isolated brains unanimously agree (Protocol 111) it chooses and executes
an action for the user via Ollama function-calling + MCP tools. Stack: Go 1.26, gRPC, Fiber+WS,
local Ollama. House rules: strictly sequential (WIP=1); config in `magi.yaml`, never hardcode;
RAM budget < 10GB.

## Focus — Protocol 111 end-to-end
Three local brains (Melchior=logic, Balthasar=protective/veto, Casper=linear) deliberate in
isolation, converge through a gateway with a live dashboard, and auto-execute the agreed action.
Done = drive a scenario at `localhost:8080` and watch unanimity → tool execution, under 10GB RAM.

## 🎯 Now
MVP complete — Protocol 111 verified end-to-end in the browser (consensus + auto-exec + deadlock), RAM ~7.4GB peak. Next work is in Parked.
> **Sync rule (WIP=1):** exactly one task is 🟡 at a time. On finish: mark ✅, log under Done (dated), promote next to 🟡, update this block. Never two 🟡; never lag.

## Plan — 3 days (deploy at end of each day)

**📅 Day 1 — 2026-08-07 · Foundation + tool layer**
> 🚀 **Deploy (end of Day 1)** — DONE: `go build ./...` clean, `go test ./...` green, ollama tool-calling verified live.

**📅 Day 2 — 2026-08-08 · Brains + Protocol 111**
> 🚀 **Deploy (end of Day 2)** — DONE: consensus unit tests green (4/4 outcomes), gRPC Health verified in-process.

**📅 Day 3 — 2026-08-09 · Gateway + dashboard + verify**
> 🚀 **Deploy (end of Day 3)** — DONE: `make run` starts all 4; browser at `localhost:8111` drove a scenario to Protocol 111 consensus + auto-exec of notify_user.

## Done
- ✅ 2026-08-07 **Dashboard.** Single `gateway/static/index.html`: React 18 + React Flow (@xyflow/react) via esm.sh import map (no build, pure browser); chat box, gateway↔brain packet edges, per-brain verdict cards, WS client, reduce-motion. Renders + WS-connected verified in real browser (Playwright), 0 console errors.
- ✅ 2026-08-07 **Gateway.** Fiber HTTP+WS on `:8111` (MAGI_HTTP_ADDR override): `/api/deliberate` (chat/scenario → Protocol 111), `/api/status`, `/api/tools`, `/ws` broadcast hub; orchestrator wired to live telemetry. Boots + serves catalog (verified). Brains dialled lazily.
- ✅ 2026-08-07 **Consensus orchestrator.** `internal/consensus`: concurrent fan-out, unanimity (same AFFIRMATIVE + same tool, args from highest-confidence brain), Balthasar veto, re-deliberation with peer dissent, auto-exec gate, event emitter. 4/4 outcome tests green + gRPC brain client.
- ✅ 2026-08-07 **Three services.** `services/melchior|balthasar|casper` thin mains over shared `brain.Serve(cfg, name)` (persona/model/port from config, graceful shutdown, gRPC reflection). In-process gRPC Health test green.
- ✅ 2026-08-07 **Persona + shared Brain server.** `pkg/persona` (Melchior/Balthasar/Casper + Protocol 111 contract) + `internal/brain` gRPC server using Ollama structured `format` (VerdictSchema). Live smoke green: Balthasar returned AFFIRMATIVE/conf 80/notify_user in 1.84s.
- ✅ 2026-08-07 **pkg/tools.** Registry (unified catalog, auto|confirm mode resolver, default confirm) + native builtins (`notify_user`, `log_decision`) + MCP client over go-sdk v1.7.0 (CommandTransport/stdio, ListTools/CallTool). `go test` green; full `go test ./...` green.
- ✅ 2026-08-07 **pkg/ollama.** Lightweight `/api/chat` client (tools, structured `format`, num_ctx/keep_alive from config; no heavy ollama dep) + `pkg/config` loader. Live smoke test green: llama3.2:1b returned a parsed `log_decision` tool_call in 2.25s.
- ✅ 2026-08-07 **Proto + codegen.** `proto/magi.proto` (Brain: Deliberate/Health, Verdict enum, ToolSpec/ProposedAction/PeerDissent) → `gen/*.pb.go`; grpc+protobuf deps added; `go build ./gen/...` green.
- ✅ 2026-08-07 **Scaffold + config.** dir tree, `go mod init github.com/danielbarboza/magi`, `magi.yaml` (models/addrs/consensus/tool allowlist, RAM guards), `.gitignore`, `ROADMAP.md`, `llm/` context. `git init` done.

## Parked (post-MVP — not now)
Streaming token-by-token to dashboard, multi-scenario history/audit log, richer MCP server presets, per-brain model hot-swap UI.

---
## Reference
- Module: `github.com/danielbarboza/magi`. Go 1.26.1. Config: `magi.yaml`.
- gRPC ports: melchior 50051, balthasar 50052, casper 50053. Gateway HTTP `:8111` (avoid 80/443/3000/8000/8080).
- Models: qwen2.5:1.5b / llama3.2:1b / phi4-mini. RAM guard: `num_ctx` 2048, `keep_alive` 5m.
- Build: `go build ./...`. Codegen: `protoc --go_out --go-grpc_out proto/magi.proto`.
