// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package transport

import (
	"context"

	apiv1 "github.com/ClessLi/component-base/examples/apiserver/api/v1"
	httpclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/http/v1"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	http_transport "github.com/go-kit/kit/transport/http"
)

// UserTransport defines the transport layer interface for the user resource.
// Each method returns a fully configured httpclientv1.Client bound to one
// remote HTTP operation.
type UserTransport interface {
	Create() httpclientv1.Client[*apiv1.CreateUserRequest, *apiv1.User]
	Get() httpclientv1.Client[httpclientv1.NilBody, *apiv1.User]
	List() httpclientv1.Client[httpclientv1.NilBody, []*apiv1.User]
}

// userTransport implements UserTransport.
type userTransport struct {
	createClient httpclientv1.Client[*apiv1.CreateUserRequest, *apiv1.User]
	getClient    httpclientv1.Client[httpclientv1.NilBody, *apiv1.User]
	listClient   httpclientv1.Client[httpclientv1.NilBody, []*apiv1.User]
}

// Compile-time check that userTransport satisfies the transport contract.
var _ UserTransport = (*userTransport)(nil)

func newUserTransport(t *transport) UserTransport {
	return &userTransport{
		// Build via the fluent builder to demonstrate middleware composition.
		createClient: httpclientv1.NewClientBuilder[*apiv1.CreateUserRequest, *apiv1.User](
			httpclientv1.HTTPMethodPost,
			t.baseURL+"/v1/users",
		).
			Use(loggingCreateMiddleware()).
			WithOptions(http_transport.SetClient(t.client)).
			Build(),
		getClient: httpclientv1.NewHTTPClient[httpclientv1.NilBody, *apiv1.User](
			httpclientv1.HTTPMethodGet,
			t.baseURL+"/v1/users/:id",
			http_transport.SetClient(t.client),
		),
		listClient: httpclientv1.NewHTTPClient[httpclientv1.NilBody, []*apiv1.User](
			httpclientv1.HTTPMethodGet,
			t.baseURL+"/v1/users",
			http_transport.SetClient(t.client),
		),
	}
}

func (u *userTransport) Create() httpclientv1.Client[*apiv1.CreateUserRequest, *apiv1.User] {
	return u.createClient
}

func (u *userTransport) Get() httpclientv1.Client[httpclientv1.NilBody, *apiv1.User] {
	return u.getClient
}

func (u *userTransport) List() httpclientv1.Client[httpclientv1.NilBody, []*apiv1.User] {
	return u.listClient
}

// loggingCreateMiddleware is a demonstration client-side middleware. It wraps
// the create endpoint to log each outgoing request before delegating to the
// next handler, showing how Use/Chain compose behaviour onto a Client.
func loggingCreateMiddleware() httpclientv1.Middleware[*apiv1.CreateUserRequest, *apiv1.User] {
	return func(next httpclientv1.Endpoint[*apiv1.CreateUserRequest, *apiv1.User]) httpclientv1.Endpoint[*apiv1.CreateUserRequest, *apiv1.User] {
		return func(ctx context.Context, request httpclientv1.HTTPRequest[*apiv1.CreateUserRequest]) (*apiv1.User, error) {
			logV1.Debugf("httpclient: POST /v1/users name=%q", request.Body.Name)
			return next(ctx, request)
		}
	}
}
