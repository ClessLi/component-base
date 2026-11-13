package v1

import (
	"context"

	"github.com/ClessLi/component-base/pkg/errors"
	"github.com/ClessLi/component-base/pkg/utils"
	"github.com/go-kit/kit/endpoint"
)

// Endpoint is the domain-typed client endpoint. It mirrors the HTTP SDK's Endpoint.
type Endpoint[REQ any, RESP any] func(ctx context.Context, request GRPCRequest[REQ]) (response RESP, err error)

// NewEndpoint adapts a go-kit endpoint into a domain-typed Endpoint.
func NewEndpoint[REQ any, RESP any](ep endpoint.Endpoint) Endpoint[REQ, RESP] {
	return func(ctx context.Context, request GRPCRequest[REQ]) (response RESP, err error) {
		resp, err := ep(ctx, request)
		var zero RESP
		if err != nil {
			return zero, err
		}

		if utils.IsNil(resp) {
			return zero, nil
		}

		response, ok := resp.(RESP)
		if !ok {
			return zero, errors.Errorf("failed to convert response to '%T'", zero)
		}
		return response, nil
	}
}
