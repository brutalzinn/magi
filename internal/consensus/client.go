package consensus

import (
	"context"
	"fmt"

	magipb "magi/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCBrain is a BrainClient backed by a gRPC connection to a brain service.
type GRPCBrain struct {
	name   string
	conn   *grpc.ClientConn
	client magipb.BrainClient
}

// DialBrain connects to a brain service at addr.
func DialBrain(name, addr string) (*GRPCBrain, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial brain %s at %s: %w", name, addr, err)
	}
	return &GRPCBrain{name: name, conn: conn, client: magipb.NewBrainClient(conn)}, nil
}

func (g *GRPCBrain) Name() string { return g.name }

func (g *GRPCBrain) Deliberate(ctx context.Context, req *magipb.DeliberateRequest) (*magipb.DeliberateResponse, error) {
	return g.client.Deliberate(ctx, req)
}

// Health calls the brain's Health RPC (used by the gateway for node status).
func (g *GRPCBrain) Health(ctx context.Context) (*magipb.HealthResponse, error) {
	return g.client.Health(ctx, &magipb.HealthRequest{})
}

func (g *GRPCBrain) Close() error { return g.conn.Close() }
