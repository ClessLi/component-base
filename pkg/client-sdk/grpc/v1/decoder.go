package v1

import "context"

// Decoder converts a protobuf response message to a domain response.
// It is the client-side counterpart of the server framework's Encoder.
// ProtoResp: protobuf response type; RESP: domain response type.
type Decoder[ProtoResp, RESP any] func(ctx context.Context, resp ProtoResp) (RESP, error)

// StreamDecoder converts the fully aggregated streaming payload into a domain
// response. The transport layer concatenates all received chunks before calling it.
type StreamDecoder[RESP any] func(ctx context.Context, data []byte) (RESP, error)
