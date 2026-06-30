package v1

import (
	"context"

	"github.com/ClessLi/component-base/pkg/utils"
	"github.com/go-kit/kit/endpoint"
	"github.com/marmotedu/errors"
)

type Endpoint[REQ any, RESP any] func(ctx context.Context, request HTTPRequest[REQ]) (response RESP, err error)

func NewEndpoint[REQ any, RESP any](ep endpoint.Endpoint) Endpoint[REQ, RESP] {
	return func(ctx context.Context, request HTTPRequest[REQ]) (response RESP, err error) {
		resp, err := ep(ctx, request)
		var i RESP
		if err != nil {
			return i, err
		}

		if utils.IsNil(resp) {
			return i, nil
		}

		response, ok := resp.(RESP)
		if !ok {
			return i, errors.Errorf("failed to convert response to '%T'", i)
		}
		return
	}
}
