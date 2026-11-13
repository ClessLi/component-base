package v1

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ClessLi/component-base/examples/grpcserver"
	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGRPCClientSDK runs a round-trip against the example gRPC server: a unary Echo
// and a server-streaming Download that the SDK transparently aggregates into one response.
func TestGRPCClientSDK(t *testing.T) {
	port := freePort(t)
	opts := grpcserver.NewOptions()
	opts.InsecureServingOptions.BindPort = port

	cfg, err := grpcserver.CreateConfigFromOptions(opts)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prepared := grpcserver.NewServer(ctx, cfg).PrepareRun()
	runErr := make(chan error, 1)
	go func() { runErr <- prepared.Run() }()
	defer func() {
		_ = prepared.Close()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
		}
	}()

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dialCancel()
	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Unary RPC: domain string -> EchoRequest -> EchoResponse -> domain string.
	echoClient := NewUnaryClient[string, string, *pbv1.EchoRequest, *pbv1.EchoResponse](
		conn, "grpcserverpb.Demo", "Echo",
		func(ctx context.Context, msg string) (*pbv1.EchoRequest, error) {
			return &pbv1.EchoRequest{Message: msg}, nil
		},
		func(ctx context.Context, resp *pbv1.EchoResponse) (string, error) {
			return resp.GetMessage(), nil
		},
		&pbv1.EchoResponse{},
	)
	got, err := echoClient.Endpoint()(context.Background(), GRPCRequest[string]{Body: "hello"})
	if err != nil {
		t.Fatalf("echo: %v", err)
	}
	if got != "echo: hello" {
		t.Fatalf("echo = %q, want %q", got, "echo: hello")
	}

	// Server-streaming RPC: the transport layer aggregates all chunks; the caller
	// receives the complete payload as a single unary response.
	downloadClient := NewStreamClient[string, []byte, *pbv1.Chunk](
		conn, "grpcserverpb.Demo", "Download",
		func(ctx context.Context, key string) (grpc.ClientStream, error) {
			return pbv1.NewDemoClient(conn).Download(ctx, &pbv1.DownloadRequest{Key: key})
		},
		(*pbv1.Chunk).GetData,
		func(ctx context.Context, data []byte) ([]byte, error) { return data, nil },
	)
	data, err := downloadClient.Endpoint()(context.Background(), GRPCRequest[string]{Body: "greeting"})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	want := []byte(strings.Repeat("component-base generic gRPC server streaming demo. ", 64))
	if !bytes.Equal(data, want) {
		t.Fatalf("download payload mismatch: got %d bytes, want %d bytes", len(data), len(want))
	}
}

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
