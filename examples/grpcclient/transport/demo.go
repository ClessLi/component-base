// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package transport

import (
	"context"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	grpcclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"google.golang.org/grpc"
)

const demoService = "grpcserverpb.Demo"

// DemoTransport defines the transport layer interface for the Demo service.
// Each method returns a fully configured grpcclientv1.Client bound to one RPC.
type DemoTransport interface {
	Echo() grpcclientv1.Client[string, string]
	Download() grpcclientv1.Client[string, []byte]
}

// demoTransport implements DemoTransport.
type demoTransport struct {
	echoClient     grpcclientv1.Client[string, string]
	downloadClient grpcclientv1.Client[string, []byte]
}

// Compile-time check that demoTransport satisfies the transport contract.
var _ DemoTransport = (*demoTransport)(nil)

func newDemoTransport(t *transport) DemoTransport {
	// Unary RPC, built through the fluent builder to demonstrate middleware. The
	// request/response codecs come from the encoder and decoder sub-packages.
	echoClient := grpcclientv1.NewUnaryClientBuilder[string, string, *pbv1.EchoRequest, *pbv1.EchoResponse](
		t.conn, demoService, "Echo",
		t.encoderFactory.Demo().Echo(),
		t.decoderFactory.Demo().Echo(),
		&pbv1.EchoResponse{},
	).
		Use(loggingEchoMiddleware()).
		Build()

	// Server-streaming RPC: the opener is conn-bound and stays here, while the
	// chunk extractor and stream decoder come from the decoder factory. The SDK
	// transport aggregates every Chunk into one []byte response.
	downloadClient := grpcclientv1.NewStreamClientBuilder[string, []byte, *pbv1.Chunk](
		t.conn, demoService, "Download",
		func(ctx context.Context, key string) (grpc.ClientStream, error) {
			return pbv1.NewDemoClient(t.conn).Download(ctx, &pbv1.DownloadRequest{Key: key})
		},
		t.decoderFactory.Demo().DownloadChunk(),
		t.decoderFactory.Demo().Download(),
	).
		Build()

	return &demoTransport{
		echoClient:     echoClient,
		downloadClient: downloadClient,
	}
}

func (d *demoTransport) Echo() grpcclientv1.Client[string, string] {
	return d.echoClient
}

func (d *demoTransport) Download() grpcclientv1.Client[string, []byte] {
	return d.downloadClient
}

// loggingEchoMiddleware is a demonstration client-side middleware that logs the
// outgoing unary Echo request before delegating to the next endpoint.
func loggingEchoMiddleware() grpcclientv1.Middleware[string, string] {
	return func(next grpcclientv1.Endpoint[string, string]) grpcclientv1.Endpoint[string, string] {
		return func(ctx context.Context, request grpcclientv1.GRPCRequest[string]) (string, error) {
			logV1.Debugf("grpcclient: Demo/Echo message=%q", request.Body)
			return next(ctx, request)
		}
	}
}
