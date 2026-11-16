// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package transport is the transport layer of the gRPC client SDK example. It
// owns the shared *grpc.ClientConn and lazily builds one grpcclientv1.Client per
// RPC, injecting the domain <-> protobuf codecs provided by the encoder and
// decoder sub-packages. Streaming is aggregated by the SDK transport, so callers
// see a unary endpoint. This mirrors the layering of the task_scheduler client.
package transport

import (
	"sync"

	"github.com/ClessLi/component-base/examples/grpcclient/transport/decoder"
	"github.com/ClessLi/component-base/examples/grpcclient/transport/encoder"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"google.golang.org/grpc"
)

// Factory is the entry point of the transport layer.
type Factory interface {
	Demo() DemoTransport
}

// transport implements Factory, caching each resource transport behind a
// sync.Once so it is built at most once.
type transport struct {
	conn           *grpc.ClientConn
	encoderFactory encoder.Factory
	decoderFactory decoder.Factory

	onceDemo               sync.Once
	singletonDemoTransport DemoTransport
}

// Compile-time check that transport satisfies the factory contract.
var _ Factory = (*transport)(nil)

func (t *transport) Demo() DemoTransport {
	t.onceDemo.Do(func() {
		if t.singletonDemoTransport == nil {
			t.singletonDemoTransport = newDemoTransport(t)
		}
	})
	if t.singletonDemoTransport == nil {
		logV1.Fatal("demo transport client is nil")
		return nil
	}

	return t.singletonDemoTransport
}

// New creates a transport Factory bound to the given client connection.
func New(conn *grpc.ClientConn) Factory {
	return &transport{
		conn:           conn,
		encoderFactory: encoder.New(),
		decoderFactory: decoder.New(),
		onceDemo:       sync.Once{},
	}
}
