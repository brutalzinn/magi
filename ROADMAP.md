# Magi — Roadmap
> Updated 2026-08-08. Focus: full-context autonomous decisions — brains decide from the scenario they're given, Magi gates which tools are exposed. Terse; `git log` holds detail.

## What is Magi
Local tripartite autonomous decision engine (Evangelion Magi). Three gRPC brain services
(Melchior=logic, Balthasar=protect, Casper=linear), each a persona-baked `llama3.2:3b` Ollama
model, deliberate over a scenario and must reach a unanimous verdict (Protocol 111); a Fiber
gateway orchestrates and serves a live React dashboard. Stack: Go 1.26, gRPC, Fiber+WS, Ollama.
House rules: strictly sequential (WIP=1); config in `magi.yaml`, never hardcode; RAM < 10GB; no
hardcoded inference — models decide from inputs/tools.

## Focus — full-context autonomous decisions
Magi decides from the full context it is given, not by searching the internet. The brains receive
the whole scenario in their prompt and think cold on every new input (no cross-interaction state).
Magi is the gatekeeper of tool exposure: a curated `expose` list in `magi.yaml` decides which tools
reach the brains (fail-closed). Brains propose only from the exposed set; Magi executes on Protocol
111. Internet function-calling (`web_search`/`fetch_url`) is removed.

## 🎯 Now
Next: **#5 get_operator_state (8neu)** (🟡) — a LOCAL context tool, exposed by name. Timer stopped between tasks — start on go-ahead.
> Note: brains no longer look things up on the internet. If a scenario needs grounding, expose a LOCAL tool (e.g. operator-state) via `expose` — Magi decides what's available, never an open web search.
> **Sync rule (WIP=1):** exactly one task is 🟡 at a time. On finish: mark ✅, log under Done (dated), promote next to 🟡, update this block. Never two 🟡; never lag.
> **Per-task finish:** TDD (test first) → run `/simplify` on the diff → then mark done. Code: KISS, no comments, avoid `else`, design for reuse/extension.

## Plan — 3 days (deploy at end of each day)

**📅 Day 1 — 2026-08-08 · Tool taxonomy + Magi-gated exposure**
1. ✅ **Tool kind (INFO vs ACTION).** Done — see Done section.
2. ✅ **Magi-gated tool exposure + remove internet tools.** Done — see Done section.
> 🚀 **Deploy (end of Day 1)** — `go test ./...`; confirm the packet viewer advertises only `expose`d tools.

**📅 Day 2 — 2026-08-09 · Local context tool**
3. ⬜ **get_operator_state (8neu).** New LOCAL tool → personal 8neu health API; token+base from local config (gitignored, never commit); returns raw vitals. Add to `expose` so Magi advertises it.
> 🚀 **Deploy (end of Day 2)** — `go test ./...` green; expose the tool and drive a scenario that needs it.

**📅 Day 3 — 2026-08-10 · Dashboard + verify**
4. ⬜ **Dashboard tool packets.** New `tool_call`/`tool_result` events (`internal/consensus`); render in the packet inspector + graph (`gateway/static/index.html`).
5. ⬜ **Verify end-to-end.** Browser: "HP 15%, Mana 10%…" → brains decide from the scenario (+ any exposed local tool) → Protocol 111 + action, no deadlock; RAM < 10GB.
> 🚀 **Deploy (end of Day 3)** — `make run`; `localhost:8111`; drive the HP/Mana scenario, confirm no deadlock.

## Done
- ✅ 2026-08-08 **Magi-gated tool exposure + remove internet tools.** Reversed the internet-gather direction: deleted `pkg/tools/web.go` + `web_test.go` (`web_search`/`fetch_url` gone). Magi is now the gatekeeper of the deliberation surface — a curated `expose` list in `magi.yaml` → `ToolsConfig.Expose` → `reg.Expose(cfg.Tools.Expose...)` (gateway, after MCP connect) → `Registry.ExposedCatalog()`; consensus `Run` advertises the exposed subset (was full `Catalog()`), fail-closed by default. Exposure curates what brains see/propose; execution stays gated by allowlist mode + unanimity + Balthasar veto. Also locked **cold thinking**: `internal/brain/cold_test.go` proves each `Deliberate` sends one fresh user turn, no history/context, and no prior scenario leaks into the next request. TDD: `TestExposedCatalog_*`, `TestRun_AdvertisesOnlyExposedTools` (capturing fake brain), config parse test. `/simplify` applied (`specsWhere` predicate dedup, `names` test helper reuse, honest curation comments). `go test ./...` green, vet+gofmt clean.
- ✅ 2026-08-08 **web_search + fetch_url (INFO tools).** *(reverted 2026-08-08 — see above.)* New `pkg/tools/web.go`: `web_search` (keyless DuckDuckGo Instant Answer API → text summary) + `fetch_url` (bounded GET → HTML stripped to text), both `KindInfo`. Explicit opt-in `RegisterWeb` at the gateway composition root (kept out of local-safe `RegisterBuiltins`); `registerWeb(client, base)` seam injects an httptest server. TDD: `web_test.go` drives real `Execute`/HTTP against test servers (search summary, HTML strip, empty-url error, INFO not action). `/simplify` applied (shared `stringArg`, `[]byte` httpGet, dead-field drop). `go test ./...` green, vet+gofmt clean. Pushed `5b99f10`.
- ✅ 2026-08-08 **Tool kind (INFO vs ACTION).** `Kind` (action|info) + `IsAction()` on `tools.Spec` (zero value = action, so MCP tools stay executable); `NewNativeInfoTool` constructor; builtins tagged ACTION; consensus `agreedAction(resps, isAction)` filters proposed tools to ACTION-kind only via `actionSet(catalog)` — an INFO-tool proposal reaches answer-only consensus and executes nothing (gather loop lands in #4). TDD: `TestToolKind_Taxonomy`, `TestRun_InfoToolProposal_ConsensusButNoAction`; `go test ./...` green, vet clean. (Roadmap's original grounding named `unanimous()`, gone since the answer-field refactor; intent implemented on `allAffirmative`/`agreedAction`.)
- ✅ 2026-08-07 **Answer field + relaxed consensus.** Each brain now returns a direct `answer` (proto `DeliberateResponse.answer`, `VerdictSchema`, contract in `internal/brain/brain.go`); Protocol 111 = all AFFIRMATIVE (`allAffirmative`), shared tool optional (`agreedAction`), agreed answer surfaced (`agreedAnswer`) in `internal/consensus/consensus.go`; dashboard shows each brain's answer + reply. Verified: "What is 2 plus 2?" → consensus 1 round, all answer "4", Magi replies "4" (was: 3-round deadlock / observe-only).
- ✅ 2026-08-07 **Realtime packet visualization + persona models.** Custom React Flow edge animates a glowing packet along the wire (gateway▶brain dispatch, brain◀gateway verdict) carrying a short content label; Wireshark-style inspector (list + full-payload detail); responsive/mobile. Personas baked into lean `Modelfile.*` (`llama3.2:3b`, temps 0.1/0.2/0.15). Hardcoded HP/Mana inference removed. Verified in browser (consensus + deadlock paths).
- ✅ 2026-08-07 **Protocol 111 MVP.** proto/gen (Brain gRPC); `pkg/ollama` (lean tool-calling client), `pkg/config`, `pkg/tools` (registry + native builtins + MCP go-sdk client); `internal/brain` (structured verdict via Ollama `format`); three services; `internal/consensus` (unanimity, Balthasar veto, re-deliberation with peer dissent, auto-exec) — 4/4 outcome tests green; gateway (Fiber HTTP/WS on `:8111`) + React dashboard. End-to-end: drove a scenario to Protocol 111 consensus + auto-exec of `notify_user`. Pushed to `github.com:brutalzinn/magi`.

## Parked (post-MVP — not now)
Expose Magi as a decision plugin for **Tesseract** (`../tesseract` calls Magi when it needs a decision — stable request/response wrapping `Orchestrator.Run`; tracker #8/172), token-by-token streaming to dashboard, multi-scenario history/audit log, richer MCP presets, per-brain model hot-swap UI, vendoring React/React-Flow locally (currently esm.sh CDN).

---
## Reference
- Module `magi`. Go 1.26.1. Config `magi.yaml`. Deploy: `make setup` then `make run` (needs `ollama serve`).
- gRPC ports melchior 50051 / balthasar 50052 / casper 50053. Gateway `:8111` (avoid 80/443/3000/8000/8080).
- Models: base `llama3.2:3b` → persona models `melchior`/`balthasar`/`casper` via `Modelfile.*` (`make models`). RAM guard: `num_ctx` 2048, `keep_alive` 5m, ~7GB peak.
- Codegen: `make proto`. Tests: `make test`.
