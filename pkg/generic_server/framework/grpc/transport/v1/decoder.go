// Package v1 provides generic gRPC transport layer utilities.
package v1

import "context"

// Decoder defines the request decoder interface.
// It converts a protobuf request type to a domain request type.
// ProtoReq: protobuf request type
// DomainReq: domain request type
type Decoder[ProtoReq, DomainReq any] func(ctx context.Context, r ProtoReq) (DomainReq, error)
