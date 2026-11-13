// This file defines the server-streaming handler (StreamHandler) and its
// functional counterpart ServeStream, encapsulating the "compute + chunked send"
// pipeline for server-streaming RPCs.

package v1

import (
	"context"

	"github.com/ClessLi/component-base/pkg/errors"
	epv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/endpoint/v1"
	"google.golang.org/grpc"
)

// StreamHandler defines the server-streaming gRPC handler interface.
// ProtoReq is the protobuf request type; ProtoMsg is the protobuf message type
// carried by the stream. It encapsulates the full "compute + chunked send" pipeline
// so that pb-service streaming methods only need to call ServeGRPCStream.
type StreamHandler[ProtoReq, ProtoMsg any] interface {
	ServeGRPCStream(ctx context.Context, request ProtoReq, stream grpc.ServerStream) error
}

// streamHandler wires the unary compute pipeline (Handler) with the chunked-send
// concern (chunksize/extractor/assembler).
type streamHandler[ProtoReq, DomainReq, DomainResp, ProtoResp, ProtoMsg any] struct {
	handler   Handler[ProtoReq, ProtoResp]
	chunksize int
	extractor func(ProtoResp) []byte
	assembler MsgAssembler[ProtoMsg]
}

// ServeGRPCStream computes the full response via the unary handler pipeline, then
// sends it over the stream in chunks. Send errors (including io.EOF) are returned
// unwrapped so callers can inspect them with errors.Is.
func (h *streamHandler[ProtoReq, DomainReq, DomainResp, ProtoResp, ProtoMsg]) ServeGRPCStream(ctx context.Context, request ProtoReq, stream grpc.ServerStream) error {
	return ServeStream(ctx, h.handler, request, stream, h.chunksize, h.extractor, h.assembler)
}

// NewStreamHandler creates a server-streaming handler. It wires the unary pipeline
// (endpoint/decoder/encoder) together with the chunked-send concern
// (chunksize/extractor/assembler), keeping the unary Handler reusable.
func NewStreamHandler[ProtoReq, DomainReq, DomainResp, ProtoResp, ProtoMsg any](
	ep epv1.Endpoint[DomainReq, DomainResp],
	decoder Decoder[ProtoReq, DomainReq],
	encoder Encoder[DomainResp, ProtoResp],
	chunksize int,
	extractor func(ProtoResp) []byte,
	assembler MsgAssembler[ProtoMsg],
) StreamHandler[ProtoReq, ProtoMsg] {
	return &streamHandler[ProtoReq, DomainReq, DomainResp, ProtoResp, ProtoMsg]{
		handler:   NewHandler(ep, decoder, encoder),
		chunksize: chunksize,
		extractor: extractor,
		assembler: assembler,
	}
}

// ServeStream computes the full response via the unary Handler pipeline
// (decode -> endpoint -> encode), then sends it over the server stream in chunks
// of at most chunksize bytes. It is the functional counterpart of StreamHandler:
// it keeps Handler unary while encapsulating the "compute + chunked send" composition.
//
// extractor pulls the payload bytes from the computed ProtoResp; assembler wraps
// each chunk into the stream's protobuf message. Send errors (including io.EOF)
// are returned unwrapped so callers can inspect them with errors.Is.
func ServeStream[ProtoReq, ProtoResp, ProtoMsg any](
	ctx context.Context,
	h Handler[ProtoReq, ProtoResp],
	req ProtoReq,
	stream grpc.ServerStream,
	chunksize int,
	extractor func(ProtoResp) []byte,
	assembler MsgAssembler[ProtoMsg],
) error {
	if h == nil {
		return errors.New("handler must not be nil")
	}
	if extractor == nil {
		return errors.New("payload extractor must not be nil")
	}

	_, resp, err := h.ServeGRPC(ctx, req)
	if err != nil {
		return err
	}

	return StreamSendMsg(stream, extractor(resp), chunksize, assembler)
}
