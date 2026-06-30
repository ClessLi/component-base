package v1

//go:generate mockgen -self_package=github.com/ClessLi/component-base/pkg/client-sdk/http/v1 -destination=mock_client.go -package=v1 github.com/ClessLi/component-base/pkg/client-sdk/http/v1 Client
import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	http_transport "github.com/go-kit/kit/transport/http"
	"github.com/marmotedu/errors"
)

// Client interface provides a standard endpoint with request/response conversion capabilities
type Client[REQ any, RESP any] interface {
	Endpoint() Endpoint[REQ, RESP]
}

type kitClient interface {
	Endpoint() endpoint.Endpoint
}

type httpClient[REQ any, RESP any] struct {
	kitClient kitClient
}

func (h *httpClient[REQ, RESP]) Endpoint() Endpoint[REQ, RESP] {
	return NewEndpoint[REQ, RESP](h.kitClient.Endpoint())
}

func NewHTTPClient[REQ any, RESP any](method, path string, options ...http_transport.ClientOption) Client[REQ, RESP] {
	kitclient := http_transport.NewExplicitClient(
		func(ctx context.Context, request interface{}) (*http.Request, error) {
			req, err := http.NewRequest(method, path, nil)
			if err != nil {
				return nil, err
			}
			//err = enc(ctx, req, request)
			r, ok := request.(HTTPRequest[REQ])
			if !ok {
				return nil, errors.Errorf("request data type(%T) error", request)
			}
			err = EncodeRequest[REQ](ctx, req, r)
			if err != nil {
				return nil, err
			}
			return req, nil
		},
		func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			return DecodeResponse[RESP](ctx, resp)
		},
		options...,
	)
	return &httpClient[REQ, RESP]{kitClient: kitclient}
}
