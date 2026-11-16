// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package grpcclient demonstrates how to build a self-contained business client
// SDK on top of component-base's gRPC client SDK
// (pkg/client-sdk/grpc/v1). It is the client counterpart of examples/grpcserver
// and is structured like the task_scheduler client SDK in real projects (eir):
// a five-layer split whose transport owns dedicated encoder/decoder sub-packages
// and whose client owns the connection and a Config-driven constructor.
//
// Layer architecture (each layer is one package):
//
//	Transport layer  transport/          -- builds one grpcclientv1.Client per RPC, owns *grpc.ClientConn,
//	                                          injects codecs from the encoder/ and decoder/ sub-packages
//	  request codecs transport/encoder/   -- Factory handing out grpcclientv1.Encoder per unary RPC
//	  response codecs transport/decoder/  -- Factory handing out Decoder / MsgExtractor / StreamDecoder
//	Endpoint layer   endpoint/           -- adapts Client into domain-typed grpcclientv1.Endpoint functions
//	Service layer    service/            -- business methods that assemble GRPCRequest and call endpoints
//	Client layer     client.go, config.go-- New/Config wire all layers, own the connection and expose Close
//
// Assembly chain (see client.go):
//
//	transport.New -> endpoint.New -> service.New -> Client
//
// Unlike the HTTP SDK, the gRPC SDK cannot marshal protobuf messages
// automatically, so codecs are injected explicitly through the transport's
// encoder/decoder sub-packages:
//   - Encoder[REQ, ProtoReq] and Decoder[ProtoResp, RESP] for unary RPCs
//     (see transport/encoder/demo.go, transport/decoder/demo.go);
//   - StreamOpener[REQ], MsgExtractor[ProtoMsg] and StreamDecoder[RESP] for
//     server-streaming RPCs. The conn-bound opener stays in transport/demo.go
//     while the extractor/decoder come from the decoder factory. The SDK
//     transport aggregates every streamed chunk, so the caller consumes Download
//     as an ordinary unary response and never touches the stream directly.
//
// Client-side middlewares are supported through the fluent builder: the unary
// Echo client composes a logging middleware via NewUnaryClientBuilder(...).Use(...).
//
// # Reusing this example in a business project
//
//  1. Copy these directories (transport incl. encoder/decoder, endpoint, service,
//     client.go, config.go) and rename the packages; regenerate your own pb code
//     from your .proto files.
//  2. Replace each transport RPC: point at your real service/method names and
//     rewrite the encoders/decoders in the sub-packages (and the stream opener
//     in the transport resource file) for your types.
//  3. Keep the layer contracts and the transport.New -> endpoint.New ->
//     service.New assembly chain unchanged; each layer only delegates below.
//  4. Naming conventions: each layer entry point is a New returning XxxFactory;
//     a resource transport exposes one method per RPC returning a Client.
//
// # Trying the client
//
// The client dials and owns the connection, so it is closed via Close. Either
// dial directly:
//
//	client, err := grpcclient.New("127.0.0.1:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
//	defer client.Close()
//
// or drive it from a Config:
//
//	client, err := grpcclient.NewConfig().Complete().NewClient()
//
// then call the business methods:
//
//	resp, err := client.Demo().Echo("hello")   // -> "echo: hello"
//	data, err := client.Demo().Download("greeting")
//
// # Testing
//
// The accompanying test boots examples/grpcserver on a free port and exercises
// the unary Echo and the aggregated Download round trips through this client SDK:
//
//	go test ./examples/grpcclient/...
package grpcclient // import "github.com/ClessLi/component-base/examples/grpcclient"
