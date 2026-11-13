package v1

import "context"

// Encoder converts a domain request to a protobuf request message.
// It is the client-side counterpart of the server framework's Decoder.
// REQ: domain request type; ProtoReq: protobuf request type.
type Encoder[REQ, ProtoReq any] func(ctx context.Context, req REQ) (ProtoReq, error)
