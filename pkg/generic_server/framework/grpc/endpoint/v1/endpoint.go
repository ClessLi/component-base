// Package v1 provides generic gRPC endpoint layer utilities.
package v1

import (
	"context"
)

// Endpoint defines the endpoint interface.
// It processes a request of type DomainReq and returns a response of type DomainResp.
// DomainReq: domain request type
// DomainResp: domain response type
type Endpoint[DomainReq, DomainResp any] func(ctx context.Context, request DomainReq) (response DomainResp, err error)
