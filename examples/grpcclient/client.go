// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"

	epclientv1 "github.com/ClessLi/component-base/examples/grpcclient/endpoint"
	svcclientv1 "github.com/ClessLi/component-base/examples/grpcclient/service"
	txpclientv1 "github.com/ClessLi/component-base/examples/grpcclient/transport"
	"google.golang.org/grpc"
)

// Client is the example gRPC client. It embeds the service-layer Factory so
// callers use business methods directly (e.g. client.Demo().Echo("hi")), and it
// owns the underlying connection so Close can release it.
type Client struct {
	conn *grpc.ClientConn
	svcclientv1.Factory
}

// Close releases the gRPC connection owned by the client.
func (c *Client) Close() error {
	return c.conn.Close()
}

// newClient wires the transport -> endpoint -> service -> client layers over an
// existing connection. ctx seeds every request's context.
func newClient(ctx context.Context, conn *grpc.ClientConn) *Client {
	if ctx == nil {
		ctx = context.Background()
	}

	return &Client{
		conn:    conn,
		Factory: svcclientv1.New(ctx, epclientv1.New(txpclientv1.New(conn))),
	}
}

// New dials the given server address and returns a Client that owns the
// connection; the caller must call Close when finished.
func New(svraddr string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.NewClient(svraddr, opts...)
	if err != nil {
		return nil, err
	}

	return newClient(context.Background(), conn), nil
}
