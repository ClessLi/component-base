// Package v1 provides generic gRPC transport layer utilities.
// It defines Decoder, Encoder, and Handler interfaces with generic type constraints
// for building type-safe gRPC handlers.
package v1

import (
	"context"

	"github.com/ClessLi/component-base/pkg/errors"
	epv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/endpoint/v1"
	"github.com/go-kit/kit/transport/grpc"
)

// Handler defines the gRPC handler interface with generic type constraints.
// ProtoReq is the protobuf request type, ProtoResp is the protobuf response type.
type Handler[ProtoReq, ProtoResp any] interface {
	ServeGRPC(ctx context.Context, request ProtoReq) (context.Context, ProtoResp, error)
}

// handler implements the Handler interface with full type constraints.
// ProtoReq: protobuf request type
// DomainReq: domain request type (after decoding)
// DomainResp: domain response type (before encoding)
// ProtoResp: protobuf response type
type handler[ProtoReq, DomainReq, DomainResp, ProtoResp any] struct {
	endpoint epv1.Endpoint[DomainReq, DomainResp]
	decoder  Decoder[ProtoReq, DomainReq]
	encoder  Encoder[DomainResp, ProtoResp]
}

// ServeGRPC handles the gRPC request by decoding, calling the endpoint, and encoding the response.
func (h *handler[ProtoReq, DomainReq, DomainResp, ProtoResp]) ServeGRPC(ctx context.Context, request ProtoReq) (context.Context, ProtoResp, error) {
	retctx, response, err := grpc.NewServer(
		// Endpoint handler
		func(ctx context.Context, request interface{}) (response interface{}, err error) {
			req, ok := request.(DomainReq)
			if !ok {
				return nil, errors.Errorf("endpoint: request type %T is not DomainReq", request)
			}
			return h.endpoint(ctx, req)
		},
		// Request decoder
		func(ctx context.Context, req interface{}) (request interface{}, err error) {
			r, ok := req.(ProtoReq)
			if !ok {
				return nil, errors.Errorf("decoder: request type %T is not ProtoReq", req)
			}
			return h.decoder(ctx, r)
		},
		// Response encoder
		func(ctx context.Context, resp interface{}) (response interface{}, err error) {
			r, ok := resp.(DomainResp)
			if !ok {
				return nil, errors.Errorf("encoder: response type %T is not DomainResp", resp)
			}
			return h.encoder(ctx, r)
		},
	).ServeGRPC(ctx, request)

	if err != nil {
		var zeroResp ProtoResp
		return retctx, zeroResp, errors.Wrap(err, "grpc server handler failed")
	}

	resp, ok := response.(ProtoResp)
	if !ok {
		var zeroResp ProtoResp
		return retctx, zeroResp, errors.Errorf("response type %T is not ProtoResp", response)
	}
	return retctx, resp, nil
}

// NewHandler creates a new gRPC handler with the given endpoint, decoder, and encoder.
// The handler provides type-safe conversion between protobuf types and domain types.
func NewHandler[ProtoReq, DomainReq, DomainResp, ProtoResp any](ep epv1.Endpoint[DomainReq, DomainResp], decoder Decoder[ProtoReq, DomainReq], encoder Encoder[DomainResp, ProtoResp]) Handler[ProtoReq, ProtoResp] {
	return &handler[ProtoReq, DomainReq, DomainResp, ProtoResp]{
		endpoint: ep,
		decoder:  decoder,
		encoder:  encoder,
	}
}
