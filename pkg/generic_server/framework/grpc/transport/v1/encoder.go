// Package v1 provides generic gRPC transport layer utilities.
package v1

import "context"

// Encoder defines the response encoder interface.
// It converts a domain response type to a protobuf response type.
// DomainResp: domain response type
// ProtoResp: protobuf response type
type Encoder[DomainResp, ProtoResp any] func(ctx context.Context, r DomainResp) (ProtoResp, error)
