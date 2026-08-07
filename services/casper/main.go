// Casper — the Pragmatic Self. Serves the Brain gRPC on its configured port.
package main

import (
	"log"

	"magi/internal/brain"
	"magi/pkg/config"
)

func main() {
	cfg, err := config.Load(brain.ConfigPath())
	if err != nil {
		log.Fatalf("casper: %v", err)
	}
	if err := brain.Serve(cfg, "casper"); err != nil {
		log.Fatalf("casper: %v", err)
	}
}
