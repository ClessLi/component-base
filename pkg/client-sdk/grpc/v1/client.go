package v1

//go:generate mockgen -self_package=github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1 -destination=mock_client.go -package=v1 github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1 Client,ClientBuilder

import (
	"context"

	"github.com/ClessLi/component-base/pkg/errors"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

// Client provides a domain-typed endpoint with request/response conversion.
// It mirrors the HTTP SDK's Client; server-streaming RPCs are exposed as unary responses.
type Client[REQ any, RESP any] interface {
	Endpoint() Endpoint[REQ, RESP]
}

// TODO(channel-transport): add a channel-based streaming client at the service layer
// (e.g. StreamClient[REQ, RESP]) exposing a method that returns a domain-typed message
// channel (<-chan RESP), as an alternative to the unary-aggregated Client. It would
// build on the transport layer's StreamRecvChan plus a per-message Decoder, letting
// service callers consume server-streaming responses incrementally. To be implemented later.

type grpcClient[REQ any, RESP any] struct {
	endpoint Endpoint[REQ, RESP]
}

func (c *grpcClient[REQ, RESP]) Endpoint() Endpoint[REQ, RESP] {
	return c.endpoint
}

// ClientBuilder is the fluent builder for a gRPC client.
type ClientBuilder[REQ any, RESP any] interface {
	Use(middleware Middleware[REQ, RESP]) ClientBuilder[REQ, RESP]
	UseMany(middlewares ...Middleware[REQ, RESP]) ClientBuilder[REQ, RESP]
	WithOptions(options ...grpctransport.ClientOption) ClientBuilder[REQ, RESP]
	Build() Client[REQ, RESP]
}

type clientBuilder[REQ any, RESP any] struct {
	conn        *grpc.ClientConn
	service     string
	method      string
	options     []grpctransport.ClientOption
	middlewares []Middleware[REQ, RESP]

	// unary codec (type-erased)
	encode func(ctx context.Context, req REQ) (interface{}, error)
	decode func(ctx context.Context, resp interface{}) (interface{}, error)
	reply  interface{}

	// streaming (type-erased receive + decode)
	isStream     bool
	streamRecv   func(ctx context.Context, req REQ) ([]byte, error)
	streamDecode StreamDecoder[RESP]
}

// NewUnaryClientBuilder creates a builder for a unary RPC with injected codecs.
func NewUnaryClientBuilder[REQ, RESP, ProtoReq, ProtoResp any](
	conn *grpc.ClientConn,
	service, method string,
	encoder Encoder[REQ, ProtoReq],
	decoder Decoder[ProtoResp, RESP],
	reply ProtoResp,
) ClientBuilder[REQ, RESP] {
	return &clientBuilder[REQ, RESP]{
		conn:    conn,
		service: service,
		method:  method,
		reply:   reply,
		encode: func(ctx context.Context, req REQ) (interface{}, error) {
			return encoder(ctx, req)
		},
		decode: func(ctx context.Context, resp interface{}) (interface{}, error) {
			r, ok := resp.(ProtoResp)
			if !ok {
				return nil, errors.Errorf("response data type(%T) error", resp)
			}
			return decoder(ctx, r)
		},
	}
}

// NewStreamClientBuilder creates a builder for a server-streaming RPC. The transport
// layer aggregates all chunks (StreamRecvMsg) and decodes the full payload, so the
// resulting Client behaves as a unary endpoint.
func NewStreamClientBuilder[REQ, RESP, ProtoMsg any](
	conn *grpc.ClientConn,
	service, method string,
	opener StreamOpener[REQ],
	extractor MsgExtractor[ProtoMsg],
	decoder StreamDecoder[RESP],
) ClientBuilder[REQ, RESP] {
	return &clientBuilder[REQ, RESP]{
		conn:         conn,
		service:      service,
		method:       method,
		isStream:     true,
		streamDecode: decoder,
		streamRecv: func(ctx context.Context, req REQ) ([]byte, error) {
			stream, err := opener(ctx, req)
			if err != nil {
				return nil, err
			}
			return StreamRecvMsg[ProtoMsg](stream, extractor)
		},
	}
}

// Use adds a single middleware to the builder.
func (b *clientBuilder[REQ, RESP]) Use(middleware Middleware[REQ, RESP]) ClientBuilder[REQ, RESP] {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

// UseMany adds multiple middlewares to the builder.
func (b *clientBuilder[REQ, RESP]) UseMany(middlewares ...Middleware[REQ, RESP]) ClientBuilder[REQ, RESP] {
	b.middlewares = append(b.middlewares, middlewares...)
	return b
}

// WithOptions adds go-kit gRPC client options (applied to unary RPCs).
func (b *clientBuilder[REQ, RESP]) WithOptions(options ...grpctransport.ClientOption) ClientBuilder[REQ, RESP] {
	b.options = append(b.options, options...)
	return b
}

// Build constructs the Client with all configured middlewares applied.
func (b *clientBuilder[REQ, RESP]) Build() Client[REQ, RESP] {
	var base Endpoint[REQ, RESP]
	if b.isStream {
		base = b.buildStreamEndpoint()
	} else {
		base = b.buildUnaryEndpoint()
	}

	final := base
	if len(b.middlewares) > 0 {
		final = Chain(b.middlewares...)(base)
	}
	return &grpcClient[REQ, RESP]{endpoint: final}
}

func (b *clientBuilder[REQ, RESP]) buildUnaryEndpoint() Endpoint[REQ, RESP] {
	if b.encode == nil || b.decode == nil {
		return func(ctx context.Context, request GRPCRequest[REQ]) (RESP, error) {
			var zero RESP
			return zero, errors.New("grpc client: unary codec not configured")
		}
	}

	// Preserve the request metadata across go-kit's outgoing-metadata rebuild.
	options := append([]grpctransport.ClientOption{grpctransport.ClientBefore(preserveOutgoingMD)}, b.options...)
	kitClient := grpctransport.NewClient(
		b.conn,
		b.service,
		b.method,
		func(ctx context.Context, request interface{}) (interface{}, error) {
			r, ok := request.(GRPCRequest[REQ])
			if !ok {
				return nil, errors.Errorf("request data type(%T) error", request)
			}
			return b.encode(ctx, r.Body)
		},
		func(ctx context.Context, response interface{}) (interface{}, error) {
			return b.decode(ctx, response)
		},
		b.reply,
		options...,
	)

	kitEndpoint := NewEndpoint[REQ, RESP](kitClient.Endpoint())
	return func(ctx context.Context, request GRPCRequest[REQ]) (RESP, error) {
		ctx = attachMetadata(ctx, request.Metadata)
		return kitEndpoint(ctx, request)
	}
}

func (b *clientBuilder[REQ, RESP]) buildStreamEndpoint() Endpoint[REQ, RESP] {
	return func(ctx context.Context, request GRPCRequest[REQ]) (RESP, error) {
		var zero RESP
		ctx = attachMetadata(ctx, request.Metadata)
		data, err := b.streamRecv(ctx, request.Body)
		if err != nil {
			return zero, err
		}
		return b.streamDecode(ctx, data)
	}
}

// NewUnaryClient creates a unary gRPC client with optional go-kit client options.
func NewUnaryClient[REQ, RESP, ProtoReq, ProtoResp any](
	conn *grpc.ClientConn,
	service, method string,
	encoder Encoder[REQ, ProtoReq],
	decoder Decoder[ProtoResp, RESP],
	reply ProtoResp,
	options ...grpctransport.ClientOption,
) Client[REQ, RESP] {
	return NewUnaryClientBuilder[REQ, RESP, ProtoReq, ProtoResp](conn, service, method, encoder, decoder, reply).
		WithOptions(options...).Build()
}

// NewStreamClient creates a server-streaming gRPC client exposed as a unary endpoint.
func NewStreamClient[REQ, RESP, ProtoMsg any](
	conn *grpc.ClientConn,
	service, method string,
	opener StreamOpener[REQ],
	extractor MsgExtractor[ProtoMsg],
	decoder StreamDecoder[RESP],
	options ...grpctransport.ClientOption,
) Client[REQ, RESP] {
	return NewStreamClientBuilder[REQ, RESP, ProtoMsg](conn, service, method, opener, extractor, decoder).
		WithOptions(options...).Build()
}
