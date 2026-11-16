// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package httpclient demonstrates how to build a self-contained business client
// SDK on top of component-base's HTTP client SDK
// (pkg/client-sdk/http/v1). It is the client counterpart of examples/apiserver
// and mirrors the four-layer layout used by the eir apiserver client.
//
// Four-layer architecture (each layer is one package):
//
//	Transport layer  transport/  -- builds one httpclientv1.Client per operation, owns *http.Client and base URL
//	Endpoint layer   endpoint/   -- adapts Client into domain-typed httpclientv1.Endpoint functions
//	Service layer    service/    -- business methods that assemble HTTPRequest and call endpoints
//	Client layer     client.go   -- NewClient wires all layers and exposes the service Factory
//
// Assembly chain (see client.go):
//
//	NewTransport -> NewEndpoint -> NewService -> Client
//
// The domain types (User, CreateUserRequest) are shared with the server through
// examples/apiserver/api/v1, so the wire contract has exactly one definition.
// Path parameters (":id") are declared inside the request path and filled at
// call time through HTTPRequest.PathVars (see transport/user.go, service/…).
//
// Client-side middlewares are supported through the fluent builder: the create
// client composes a logging middleware via NewClientBuilder(...).Use(...).
//
// # Reusing this example in a business project
//
//  1. Copy these directories (transport, endpoint, service, client.go) and
//     rename the packages; point the apiv1 import at your own shared api package.
//  2. Replace each transport operation to point at your real routes and request
//     /response types; add or remove client-side middlewares with Use/UseMany.
//  3. Keep the layer contracts and the NewTransport -> NewEndpoint -> NewService
//     assembly chain unchanged; each layer only delegates to the one below.
//  4. Naming conventions: each layer entry point is XxxFactory; a resource
//     transport exposes one method per remote operation returning a Client.
//
// # Trying the client
//
// Boot examples/apiserver (it listens on 127.0.0.1:8080 by default), then:
//
//	c := httpclient.NewClient(context.Background(), "http://127.0.0.1:8080", nil)
//	defer c.Close()
//
//	u, err := c.Users().Create(&apiv1.CreateUserRequest{Name: "alice"})
//	got, err := c.Users().Get(u.ID)
//	list, err := c.Users().List()
//
// # Testing
//
// The accompanying test boots examples/apiserver on a free port and exercises
// the full Create -> Get -> List round trip through this client SDK:
//
//	go test ./examples/httpclient/...
package httpclient // import "github.com/ClessLi/component-base/examples/httpclient"
