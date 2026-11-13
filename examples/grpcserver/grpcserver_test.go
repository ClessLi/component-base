// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGRPCServer 启动通用 gRPC 服务端（真实端口），用生成的 pb 客户端验证
// 一元 Echo 与服务器流式 Download（分块）两条链路。
func TestGRPCServer(t *testing.T) {
	port := freePort(t)

	opts := NewOptions()
	opts.InsecureServingOptions.BindPort = port

	cfg, err := CreateConfigFromOptions(opts)
	if err != nil {
		t.Fatalf("CreateConfigFromOptions failed: %+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewServer(ctx, cfg)
	prepared := srv.PrepareRun()

	runErr := make(chan error, 1)
	go func() { runErr <- prepared.Run() }()
	defer func() {
		_ = prepared.Close()
		// 等待 Run 退出，避免 goroutine 泄漏。
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
		}
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dialCancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial %s failed: %+v", addr, err)
	}
	defer conn.Close()

	client := pbv1.NewDemoClient(conn)

	// 一元 Echo：验证 解码 -> 端点 -> 服务 -> 编码 链路。
	echoResp, err := client.Echo(ctx, &pbv1.EchoRequest{Message: "hello"})
	if err != nil {
		t.Fatalf("Echo failed: %+v", err)
	}
	if got := echoResp.GetMessage(); got != "echo: hello" {
		t.Fatalf("Echo response = %q, want %q", got, "echo: hello")
	}

	// 服务器流式 Download：拼接所有分块应等于预置数据，且分块数 > 1（证明 StreamSendMsg 分块）。
	stream, err := client.Download(ctx, &pbv1.DownloadRequest{Key: sampleKey})
	if err != nil {
		t.Fatalf("Download failed: %+v", err)
	}
	var buf bytes.Buffer
	chunks := 0
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Download recv failed: %+v", err)
		}
		buf.Write(chunk.GetData())
		chunks++
	}
	if !bytes.Equal(buf.Bytes(), sampleData) {
		t.Fatalf("Download payload mismatch: got %d bytes, want %d bytes", buf.Len(), len(sampleData))
	}
	if chunks <= 1 {
		t.Fatalf("Download should produce multiple chunks, got %d", chunks)
	}
}

// freePort 获取一个空闲端口（监听 :0 后释放），供测试绑定。
func freePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port failed: %+v", err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	_ = lis.Close()
	return port
}
