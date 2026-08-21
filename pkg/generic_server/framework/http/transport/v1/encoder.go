// Package v1 provides generic HTTP transport layer utilities.
package v1

import "context"

// Encoder defines the response encoder interface.
// It converts a domain response type to an HTTP response type.
// DomainResp: domain response type
// HTTPResp: HTTP response type
type Encoder[DomainResp, HTTPResp any] func(ctx context.Context, r DomainResp) (HTTPResp, error)
