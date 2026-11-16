// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package endpoint is the endpoint layer of the gRPC client SDK example. It
// adapts the transport layer's Client objects into domain-typed
// grpcclientv1.Endpoint functions consumed by the service layer, mirroring the
// HTTP client SDK example's endpoint layout.
package endpoint

import (
	txpclientv1 "github.com/ClessLi/component-base/examples/grpcclient/transport"
	grpcclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1"
)

// Factory is the entry point of the endpoint layer.
type Factory interface {
	Demo() DemoEndpoints
}

// Compile-time check that factory satisfies the factory contract.
var _ Factory = (*factory)(nil)

type factory struct {
	transport txpclientv1.Factory
}

func (f *factory) Demo() DemoEndpoints {
	return newDemoEndpoints(f)
}

// New creates an endpoint Factory from the transport Factory.
func New(transport txpclientv1.Factory) Factory {
	return &factory{transport: transport}
}

// DemoEndpoints exposes the Demo RPCs as domain-typed endpoints.
type DemoEndpoints interface {
	Echo() grpcclientv1.Endpoint[string, string]
	Download() grpcclientv1.Endpoint[string, []byte]
}

// Compile-time check that demoEndpoints satisfies the endpoint contract.
var _ DemoEndpoints = (*demoEndpoints)(nil)

type demoEndpoints struct {
	transport txpclientv1.DemoTransport
}

func (d *demoEndpoints) Echo() grpcclientv1.Endpoint[string, string] {
	return d.transport.Echo().Endpoint()
}

func (d *demoEndpoints) Download() grpcclientv1.Endpoint[string, []byte] {
	return d.transport.Download().Endpoint()
}

func newDemoEndpoints(f *factory) DemoEndpoints {
	return &demoEndpoints{transport: f.transport.Demo()}
}
