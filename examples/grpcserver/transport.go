// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"context"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
)

// Factory is the transport-layer factory producing pb service implementations
// that can be registered onto a grpc.Server. It mirrors bifrost's transport/v1.Factory.
type Factory interface {
	Demo() pbv1.DemoServer
}

type transport struct {
	handlers DemoHandlers
}

// New creates the transport factory. The streaming chunkSize is already injected
// into the StreamHandler at construction (component-base removed the standalone
// transport Options struct).
func New(handlers DemoHandlers) Factory {
	return &transport{handlers: handlers}
}

func (t *transport) Demo() pbv1.DemoServer {
	return &demoServer{handlers: t.handlers}
}

// demoServer implements pbv1.DemoServer by delegating to the handler set:
// unary Handler for Echo, StreamHandler for Download.
type demoServer struct {
	pbv1.UnimplementedDemoServer
	handlers DemoHandlers
}

// Echo is a unary RPC: delegate to the handler for decode -> endpoint -> encode.
func (s *demoServer) Echo(ctx context.Context, req *pbv1.EchoRequest) (*pbv1.EchoResponse, error) {
	_, resp, err := s.handlers.Echo().ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Download is a server-streaming RPC: the StreamHandler encapsulates the full
// "compute + chunked send" pipeline, so this method only forwards the stream.
func (s *demoServer) Download(req *pbv1.DownloadRequest, stream pbv1.Demo_DownloadServer) error {
	return s.handlers.Download().ServeGRPCStream(stream.Context(), req, stream)
}
