BASE = llama3.2:3b
BIN = bin
MODULE = magi

.PHONY: setup build models run test vet proto clean

# One-shot local setup: pull the base model, build the three persona models, compile binaries.
setup: models build
	@echo "setup complete — start 'ollama serve', then run: make run"

build:
	@mkdir -p $(BIN)
	go build -o $(BIN)/melchior ./services/melchior
	go build -o $(BIN)/balthasar ./services/balthasar
	go build -o $(BIN)/casper ./services/casper
	go build -o $(BIN)/gateway ./gateway
	@echo "built -> $(BIN)/"

models:
	ollama pull $(BASE)
	ollama create melchior  -f Modelfile.melchior
	ollama create balthasar -f Modelfile.balthasar
	ollama create casper    -f Modelfile.casper

run: build
	./scripts/run.sh

test:
	go test ./...

vet:
	go vet ./...

proto:
	protoc --go_out=. --go_opt=module=$(MODULE) \
	  --go-grpc_out=. --go-grpc_opt=module=$(MODULE) proto/magi.proto

clean:
	rm -rf $(BIN)
