# Magi — Roadmap
> Updated 2026-08-07. Focus: function-calling tool-use loop so the brains gather instead of abstaining. Terse; `git log` holds detail.

## What is Magi
Local tripartite autonomous decision engine (Evangelion Magi). Three gRPC brain services
(Melchior=logic, Balthasar=protect, Casper=linear), each a persona-baked `llama3.2:3b` Ollama
model, deliberate over a scenario and must reach a unanimous verdict (Protocol 111); a Fiber
gateway orchestrates and serves a live React dashboard. Stack: Go 1.26, gRPC, Fiber+WS, Ollama.
House rules: strictly sequential (WIP=1); config in `magi.yaml`, never hardcode; RAM < 10GB; no
hardcoded inference — models decide from inputs/tools.

## Focus — function-calling tool-use loop
The brains deadlock on inputs they want to look up (e.g. "HP 15%, Mana 10%" → they ask for the
operator's state). Give them a real function-calling catalog they can call; Magi (the hub) runs the
tool and shares the result to all three next round, so they gather then decide. Done = the HP/Mana
scenario reaches Protocol 111 (no deadlock) by the brains calling a tool, not by any hardcoded
interpretation.

## 🎯 Now
Next: **#1 Tool kind (INFO vs ACTION)**.
> **Sync rule (WIP=1):** exactly one task is 🟡 at a time. On finish: mark ✅, log under Done (dated), promote next to 🟡, update this block. Never two 🟡; never lag.

## Plan — 3 days (deploy at end of each day)

**📅 Day 1 — 2026-08-08 · Tool taxonomy + real INFO tools**
1. 🟡 **Tool kind (INFO vs ACTION).** *(current)* Add `Kind` to `tools.Spec` (`pkg/tools/tools.go`); tag `notify_user`/`log_decision` ACTION in `RegisterBuiltins` (`pkg/tools/native.go`); compute unanimity on ACTION tools only (`internal/consensus/consensus.go` `unanimous()`).
2. ⬜ **web_search + fetch_url.** New `pkg/tools/web.go`: `web_search` (keyless DuckDuckGo IA API), `fetch_url` (GET → strip to text), both INFO; register in `RegisterBuiltins`.
> 🚀 **Deploy (end of Day 1)** — `go test ./...`; live smoke a `web_search` call.

**📅 Day 2 — 2026-08-09 · Gather loop + proto**
3. ⬜ **ToolResult over gRPC.** Add `repeated ToolResult` to `DeliberateRequest` (`proto/magi.proto`), regen; include results in `buildUserMessage` (`internal/brain/brain.go`).
4. ⬜ **Gather loop.** `internal/consensus/consensus.go` `Run()`: a brain proposing an INFO tool → Magi executes via registry → result shared to ALL brains next round (like peer dissent) → re-deliberate; bounded gather steps; consensus only on ACTION.
> 🚀 **Deploy (end of Day 2)** — consensus tests incl. a gather→decide fake; `go test ./...` green.

**📅 Day 3 — 2026-08-10 · Operator-state + dashboard + verify**
5. ⬜ **get_operator_state (8neu).** New tool → personal 8neu health API; token+base from local config (gitignored, never commit); returns raw vitals as INFO.
6. ⬜ **Dashboard tool packets.** New `tool_call`/`tool_result` events (`internal/consensus`); render in the packet inspector + graph (`gateway/static/index.html`).
7. ⬜ **Verify end-to-end.** Browser: "HP 15%, Mana 10%…" → brains call `get_operator_state` → Protocol 111 + action, no deadlock; RAM < 10GB.
> 🚀 **Deploy (end of Day 3)** — `make run`; `localhost:8111`; drive the HP/Mana scenario, confirm no deadlock.

## Done
- ✅ 2026-08-07 **Answer field + relaxed consensus.** Each brain now returns a direct `answer` (proto `DeliberateResponse.answer`, `VerdictSchema`, contract in `internal/brain/brain.go`); Protocol 111 = all AFFIRMATIVE (`allAffirmative`), shared tool optional (`agreedAction`), agreed answer surfaced (`agreedAnswer`) in `internal/consensus/consensus.go`; dashboard shows each brain's answer + reply. Verified: "What is 2 plus 2?" → consensus 1 round, all answer "4", Magi replies "4" (was: 3-round deadlock / observe-only).
- ✅ 2026-08-07 **Realtime packet visualization + persona models.** Custom React Flow edge animates a glowing packet along the wire (gateway▶brain dispatch, brain◀gateway verdict) carrying a short content label; Wireshark-style inspector (list + full-payload detail); responsive/mobile. Personas baked into lean `Modelfile.*` (`llama3.2:3b`, temps 0.1/0.2/0.15). Hardcoded HP/Mana inference removed. Verified in browser (consensus + deadlock paths).
- ✅ 2026-08-07 **Protocol 111 MVP.** proto/gen (Brain gRPC); `pkg/ollama` (lean tool-calling client), `pkg/config`, `pkg/tools` (registry + native builtins + MCP go-sdk client); `internal/brain` (structured verdict via Ollama `format`); three services; `internal/consensus` (unanimity, Balthasar veto, re-deliberation with peer dissent, auto-exec) — 4/4 outcome tests green; gateway (Fiber HTTP/WS on `:8111`) + React dashboard. End-to-end: drove a scenario to Protocol 111 consensus + auto-exec of `notify_user`. Pushed to `github.com:brutalzinn/magi`.

## Parked (post-MVP — not now)
Token-by-token streaming to dashboard, multi-scenario history/audit log, richer MCP presets, per-brain model hot-swap UI, vendoring React/React-Flow locally (currently esm.sh CDN).

---
## Reference
- Module `magi`. Go 1.26.1. Config `magi.yaml`. Deploy: `make setup` then `make run` (needs `ollama serve`).
- gRPC ports melchior 50051 / balthasar 50052 / casper 50053. Gateway `:8111` (avoid 80/443/3000/8000/8080).
- Models: base `llama3.2:3b` → persona models `melchior`/`balthasar`/`casper` via `Modelfile.*` (`make models`). RAM guard: `num_ctx` 2048, `keep_alive` 5m, ~7GB peak.
- Codegen: `make proto`. Tests: `make test`.
