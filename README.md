# Magi

A local, tripartite decision engine (Neon Genesis Evangelion Magi). Three small LLM "brains" —
**Melchior** (logic), **Balthasar** (protection), **Casper** (linear) — deliberate over your
scenario and must reach a **unanimous** verdict (Protocol 111). The whole exchange is shown live
on a web dashboard. Runs fully offline on a Mac mini M4 (16GB) — ~7GB peak RAM.

## Requirements

- macOS (Apple Silicon)
- Go 1.26+ and Ollama — `brew install go ollama`

## Deploy locally

```sh
ollama serve        # 1. start Ollama, leave it running
make setup          # 2. pull llama3.2:3b, build the 3 persona models + binaries
make run            # 3. start all four processes
```

Then open **http://localhost:8111**. `Ctrl-C` stops everything.

## What runs

- Three gRPC brain services — ports `50051` / `50052` / `50053`
- Gateway + dashboard — `http://localhost:8111`
- Models `melchior` / `balthasar` / `casper` — all `FROM llama3.2:3b`, personas in `Modelfile.*`

## Configure

Everything (models, ports, consensus rounds, RAM guards) lives in **`magi.yaml`** — no code changes needed.

## Make targets

`make setup` · `make run` · `make build` · `make models` · `make test` · `make clean`
