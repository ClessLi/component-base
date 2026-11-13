package v1

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// GRPCRequest carries the outgoing gRPC metadata (headers) and the domain request
// body. It is the gRPC counterpart of the HTTP SDK's HTTPRequest.
type GRPCRequest[REQ any] struct {
	Metadata map[string][]string
	Body     REQ
}

// attachMetadata stores the request metadata into the outgoing context.
func attachMetadata(ctx context.Context, md map[string][]string) context.Context {
	if len(md) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, metadata.MD(md))
}

// preserveOutgoingMD merges metadata already present in the outgoing context into
// the go-kit collected metadata. go-kit's gRPC client rebuilds the outgoing metadata
// from its ClientBefore funcs, so this preserves the metadata attached by attachMetadata.
func preserveOutgoingMD(ctx context.Context, md *metadata.MD) context.Context {
	if existing, ok := metadata.FromOutgoingContext(ctx); ok {
		*md = metadata.Join(*md, existing)
	}
	return ctx
}
