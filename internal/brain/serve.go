package brain

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	magipb "magi/gen"
	"magi/pkg/config"
	"magi/pkg/ollama"
	"magi/pkg/persona"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Serve boots a single Magi brain service: it loads the persona and model from
// config, builds an Ollama client, and runs the gRPC Brain server until a signal
// arrives. personaName is one of melchior|balthasar|casper.
func Serve(cfg *config.Config, personaName string) error {
	p, ok := persona.ByName(personaName)
	if !ok {
		return fmt.Errorf("unknown persona %q", personaName)
	}
	bc, err := cfg.Brain(personaName)
	if err != nil {
		return err
	}

	client := ollama.New(
		cfg.Runtime.OllamaURL,
		ollama.WithNumCtx(cfg.Runtime.NumCtx),
		ollama.WithKeepAlive(cfg.Runtime.KeepAlive),
		ollama.WithTimeout(cfg.RequestTimeoutDuration()),
	)

	lis, err := net.Listen("tcp", bc.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", bc.GRPCAddr, err)
	}

	srv := grpc.NewServer()
	magipb.RegisterBrainServer(srv, NewServer(p.Name, bc.Model, client))
	reflection.Register(srv) // enables grpcurl and dynamic clients

	log.Printf("[%s] %s ready on %s (model=%s, num_ctx=%d)",
		p.Name, p.Title, bc.GRPCAddr, bc.Model, cfg.Runtime.NumCtx)

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		log.Printf("[%s] shutting down", p.Name)
		srv.GracefulStop()
	}()

	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// ConfigPath returns the config file path from MAGI_CONFIG or the default.
func ConfigPath() string {
	if v := os.Getenv("MAGI_CONFIG"); v != "" {
		return v
	}
	return "magi.yaml"
}
