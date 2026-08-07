#!/usr/bin/env bash
# Launch the full Magi system: three brain services + the gateway.
# Ctrl-C stops all four. Requires `ollama serve` running and models pulled.
set -euo pipefail
cd "$(dirname "$0")/.."

: "${MAGI_CONFIG:=magi.yaml}"
: "${MAGI_STATIC:=gateway/static}"
: "${OLLAMA_URL:=http://localhost:11434}"
export MAGI_CONFIG MAGI_STATIC

if [ ! -x bin/gateway ]; then
  echo "binaries missing — run 'make build' first" >&2
  exit 1
fi

if ! curl -s -m 2 "$OLLAMA_URL/api/tags" >/dev/null; then
  echo "Ollama not reachable at $OLLAMA_URL — start it with 'ollama serve'" >&2
  exit 1
fi

pids=()
cleanup() {
  echo
  echo "stopping Magi..."
  for p in "${pids[@]:-}"; do kill "$p" 2>/dev/null || true; done
}
trap cleanup EXIT INT TERM

echo "starting brains..."
./bin/melchior &  pids+=($!)
./bin/balthasar & pids+=($!)
./bin/casper &    pids+=($!)
sleep 1
echo "starting gateway..."
./bin/gateway &   pids+=($!)

wait
