package v1

//go:generate mockgen -self_package=github.com/ClessLi/component-base/pkg/client-sdk/http/v1 -destination=mock_client.go -package=v1 github.com/ClessLi/component-base/pkg/client-sdk/http/v1 Client,ClientBuilder
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
	endpoint Endpoint[REQ, RESP]
}

func (h *httpClient[REQ, RESP]) Endpoint() Endpoint[REQ, RESP] {
	return h.endpoint
}

type ClientBuilder[REQ any, RESP any] interface {
	Use(middleware Middleware[REQ, RESP]) ClientBuilder[REQ, RESP]
	UseMany(middlewares ...Middleware[REQ, RESP]) ClientBuilder[REQ, RESP]
	WithOptions(options ...http_transport.ClientOption) ClientBuilder[REQ, RESP]
	Build() Client[REQ, RESP]
}

// clientBuilder provides a fluent API for constructing HTTP clients with middlewares
type clientBuilder[REQ any, RESP any] struct {
	method      string
	path        string
	options     []http_transport.ClientOption
	middlewares []Middleware[REQ, RESP]
}

// NewClientBuilder creates a new clientBuilder for the specified HTTP method and path
func NewClientBuilder[REQ any, RESP any](method, path string) ClientBuilder[REQ, RESP] {
	return &clientBuilder[REQ, RESP]{
		method: method,
		path:   path,
	}
}

// Use adds a single middleware to the builder
func (b *clientBuilder[REQ, RESP]) Use(middleware Middleware[REQ, RESP]) ClientBuilder[REQ, RESP] {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

// UseMany adds multiple middlewares to the builder
func (b *clientBuilder[REQ, RESP]) UseMany(middlewares ...Middleware[REQ, RESP]) ClientBuilder[REQ, RESP] {
	b.middlewares = append(b.middlewares, middlewares...)
	return b
}

// WithOptions adds http_transport.ClientOption to the builder
func (b *clientBuilder[REQ, RESP]) WithOptions(options ...http_transport.ClientOption) ClientBuilder[REQ, RESP] {
	b.options = append(b.options, options...)
	return b
}

// Build constructs the Client with all configured middlewares applied
func (b *clientBuilder[REQ, RESP]) Build() Client[REQ, RESP] {
	kitclient := http_transport.NewExplicitClient(
		func(ctx context.Context, request interface{}) (*http.Request, error) {
			req, err := http.NewRequest(b.method, b.path, nil)
			if err != nil {
				return nil, err
			}
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
		b.options...,
	)

	baseEndpoint := NewEndpoint[REQ, RESP](kitclient.Endpoint())

	var finalEndpoint Endpoint[REQ, RESP]
	if len(b.middlewares) > 0 {
		finalEndpoint = Chain(b.middlewares...)(baseEndpoint)
	} else {
		finalEndpoint = baseEndpoint
	}

	return &httpClient[REQ, RESP]{endpoint: finalEndpoint}
}

// NewHTTPClient creates a new HTTP client with optional http_transport.ClientOption
func NewHTTPClient[REQ any, RESP any](method, path string, options ...http_transport.ClientOption) Client[REQ, RESP] {
	return NewClientBuilder[REQ, RESP](method, path).WithOptions(options...).Build()
}
