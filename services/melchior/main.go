// Melchior — the Analytical Self. Serves the Brain gRPC on its configured port.
package main

import (
	"log"

	"magi/internal/brain"
	"magi/pkg/config"
)

func main() {
	cfg, err := config.Load(brain.ConfigPath())
	if err != nil {
		log.Fatalf("melchior: %v", err)
	}
	if err := brain.Serve(cfg, "melchior"); err != nil {
		log.Fatalf("melchior: %v", err)
	}
}
