// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcclient_test

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ClessLi/component-base/examples/grpcclient"
	"github.com/ClessLi/component-base/examples/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGRPCClientSDK runs a round trip against the example gRPC server: a unary
// Echo and a server-streaming Download that the SDK aggregates into one response.
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

	// The client dials lazily and owns the connection, so it is closed here.
	client, err := grpcclient.New(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer client.Close()

	t.Run("Echo", func(t *testing.T) {
		got, err := client.Demo().Echo("hello")
		if err != nil {
			t.Fatalf("echo: %v", err)
		}
		if got != "echo: hello" {
			t.Fatalf("echo = %q, want %q", got, "echo: hello")
		}
	})

	t.Run("Download", func(t *testing.T) {
		data, err := client.Demo().Download("greeting")
		if err != nil {
			t.Fatalf("download: %v", err)
		}
		want := []byte(strings.Repeat("component-base generic gRPC server streaming demo. ", 64))
		if !bytes.Equal(data, want) {
			t.Fatalf("download payload mismatch: got %d bytes, want %d bytes", len(data), len(want))
		}
	})
}

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
