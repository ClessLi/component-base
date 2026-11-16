// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package httpclient_test

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/ClessLi/component-base/examples/apiserver"
	apiv1 "github.com/ClessLi/component-base/examples/apiserver/api/v1"
	"github.com/ClessLi/component-base/examples/httpclient"
)

// TestHTTPClientSDK runs a round trip against the example HTTP server: create a
// user, fetch it by ID, then list users, all through the layered client SDK.
func TestHTTPClientSDK(t *testing.T) {
	port := freePort(t)

	opts := apiserver.NewOptions()
	opts.InsecureServingOptions.BindAddress = "127.0.0.1"
	opts.InsecureServingOptions.BindPort = port

	cfg, err := apiserver.CreateConfigFromOptions(opts)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prepared := apiserver.NewServer(ctx, cfg).PrepareRun()
	runErr := make(chan error, 1)
	go func() { runErr <- prepared.Run() }()
	defer func() {
		_ = prepared.Close()
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
		}
	}()

	baseURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	client := httpclient.NewClient(context.Background(), baseURL, nil)
	defer client.Close()

	// Wait until the server accepts connections.
	if !waitReady(baseURL) {
		t.Fatal("server did not become ready in time")
	}

	t.Run("Create", func(t *testing.T) {
		created, err := client.Users().Create(&apiv1.CreateUserRequest{Name: "alice"})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if created.ID != "1" || created.Name != "alice" {
			t.Fatalf("unexpected created user: %+v", created)
		}
	})

	t.Run("Get", func(t *testing.T) {
		got, err := client.Users().Get("1")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Name != "alice" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("List", func(t *testing.T) {
		list, err := client.Users().List()
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 user, got %d", len(list))
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

// waitReady polls the healthz endpoint until it responds or a deadline passes.
func waitReady(baseURL string) bool {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", mustHost(baseURL))
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func mustHost(baseURL string) string {
	// baseURL is "http://127.0.0.1:port"; strip the scheme for a TCP dial.
	for i := 0; i < len(baseURL); i++ {
		if baseURL[i] == '/' && i+1 < len(baseURL) && baseURL[i+1] == '/' {
			return baseURL[i+2:]
		}
	}
	return baseURL
}
