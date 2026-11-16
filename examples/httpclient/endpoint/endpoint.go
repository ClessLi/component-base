// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package endpoint is the endpoint layer of the HTTP client SDK example. It
// adapts the transport layer's Client objects into domain-typed
// httpclientv1.Endpoint functions consumed by the service layer, mirroring the
// eir apiserver client's endpoint layout.
package endpoint

import (
	apiv1 "github.com/ClessLi/component-base/examples/apiserver/api/v1"
	txpclientv1 "github.com/ClessLi/component-base/examples/httpclient/transport"
	httpclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/http/v1"
)

// Factory is the entry point of the endpoint layer.
type Factory interface {
	Users() UserEndpoints
}

// Compile-time check that factory satisfies the factory contract.
var _ Factory = (*factory)(nil)

type factory struct {
	transport txpclientv1.Factory
}

func (f *factory) Users() UserEndpoints {
	return newUserEndpoints(f)
}

// NewEndpoint creates an endpoint Factory from the transport Factory.
func NewEndpoint(transport txpclientv1.Factory) Factory {
	return &factory{transport: transport}
}

// UserEndpoints exposes the user resource operations as domain-typed endpoints.
type UserEndpoints interface {
	Create() httpclientv1.Endpoint[*apiv1.CreateUserRequest, *apiv1.User]
	Get() httpclientv1.Endpoint[httpclientv1.NilBody, *apiv1.User]
	List() httpclientv1.Endpoint[httpclientv1.NilBody, []*apiv1.User]
}

// Compile-time check that userEndpoints satisfies the endpoint contract.
var _ UserEndpoints = (*userEndpoints)(nil)

type userEndpoints struct {
	transport txpclientv1.UserTransport
}

func (u *userEndpoints) Create() httpclientv1.Endpoint[*apiv1.CreateUserRequest, *apiv1.User] {
	return u.transport.Create().Endpoint()
}

func (u *userEndpoints) Get() httpclientv1.Endpoint[httpclientv1.NilBody, *apiv1.User] {
	return u.transport.Get().Endpoint()
}

func (u *userEndpoints) List() httpclientv1.Endpoint[httpclientv1.NilBody, []*apiv1.User] {
	return u.transport.List().Endpoint()
}

func newUserEndpoints(f *factory) UserEndpoints {
	return &userEndpoints{transport: f.transport.Users()}
}
