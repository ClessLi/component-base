// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package decoder

import (
	"context"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	grpcclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1"
)

// DemoDecoders exposes the typed response decoders of each Demo RPC, matching the
// generic gRPC client SDK contract. The unary Echo RPC yields a Decoder, while
// the server-streaming Download RPC yields the chunk extractor and the stream
// decoder the SDK transport uses to aggregate the streamed payload.
type DemoDecoders interface {
	Echo() grpcclientv1.Decoder[*pbv1.EchoResponse, string]
	DownloadChunk() grpcclientv1.MsgExtractor[*pbv1.Chunk]
	Download() grpcclientv1.StreamDecoder[[]byte]
}

// Compile-time checks documenting the SDK decoder contracts for each RPC.
var (
	_ DemoDecoders                                     = demoDecoders{}
	_ grpcclientv1.Decoder[*pbv1.EchoResponse, string] = demoDecoders{}.Echo()
	_ grpcclientv1.MsgExtractor[*pbv1.Chunk]           = demoDecoders{}.DownloadChunk()
	_ grpcclientv1.StreamDecoder[[]byte]               = demoDecoders{}.Download()
)

// demoDecoders implements DemoDecoders with the typed response decoders of each RPC.
type demoDecoders struct{}

// Echo converts the EchoResponse protobuf message back into a domain string.
func (demoDecoders) Echo() grpcclientv1.Decoder[*pbv1.EchoResponse, string] {
	return func(_ context.Context, resp *pbv1.EchoResponse) (string, error) {
		return resp.GetMessage(), nil
	}
}

// DownloadChunk extracts the payload bytes from a streamed Chunk message.
func (demoDecoders) DownloadChunk() grpcclientv1.MsgExtractor[*pbv1.Chunk] {
	return (*pbv1.Chunk).GetData
}

// Download turns the fully aggregated streaming payload into the domain response.
func (demoDecoders) Download() grpcclientv1.StreamDecoder[[]byte] {
	return func(_ context.Context, data []byte) ([]byte, error) {
		return data, nil
	}
}
