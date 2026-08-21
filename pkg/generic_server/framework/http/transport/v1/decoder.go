// Package v1 provides generic HTTP transport layer utilities.
package v1

import "context"

// Decoder defines the request decoder interface.
// It converts an HTTP request type to a domain request type.
// HTTPReq: HTTP request type
// DomainReq: domain request type
type Decoder[HTTPReq, DomainReq any] func(ctx context.Context, r HTTPReq) (DomainReq, error)
