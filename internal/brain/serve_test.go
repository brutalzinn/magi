package brain

import (
	"context"
	"net"
	"testing"
	"time"

	magipb "magi/gen"
	"magi/pkg/ollama"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestBrainServer_HealthOverGRPC verifies the Brain server serves over real gRPC
// (registration + Health), no Ollama needed.
func TestBrainServer_HealthOverGRPC(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	client := ollama.New("http://localhost:11434")
	srv := grpc.NewServer()
	magipb.RegisterBrainServer(srv, NewServer("melchior", "melchior", client))
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := magipb.NewBrainClient(conn).Health(ctx, &magipb.HealthRequest{})
	if err != nil {
		t.Fatalf("Health rpc: %v", err)
	}
	if resp.Brain != "melchior" || resp.Model != "melchior" || !resp.Ready {
		t.Fatalf("unexpected health: %+v", resp)
	}
}
