// Balthasar — the Protective Self. Serves the Brain gRPC on its configured port.
package main

import (
	"log"

	"magi/internal/brain"
	"magi/pkg/config"
)

func main() {
	cfg, err := config.Load(brain.ConfigPath())
	if err != nil {
		log.Fatalf("balthasar: %v", err)
	}
	if err := brain.Serve(cfg, "balthasar"); err != nil {
		log.Fatalf("balthasar: %v", err)
	}
}
