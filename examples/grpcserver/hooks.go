// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	grpcmwv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/middleware/v1"
	svcregisterv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/service_register/v1"
	genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
)

// bizHooks implements the generic gRPC server business lifecycle hooks.
// Each method only delegates to its own layer constructor; no cross-layer
// assembly happens here (the generic server owns the initialization order).
type bizHooks struct{}

// Compile-time check that bizHooks satisfies the generic server contract.
var _ genericserver.GRPCServerBizHooks[Factory, DemoEndpoints, ServiceFactory, StoreFactory] = (*bizHooks)(nil)

// NewBizHooks creates the business hooks. The hooks are stateless: chunkSize is
// injected by the generic server via InitTransport, and ServiceRegistrations
// receives the initialized transport factory.
func NewBizHooks() *bizHooks {
	return &bizHooks{}
}

// InitStore initializes the store layer.
func (h *bizHooks) InitStore() (StoreFactory, error) {
	return NewStoreFactory(), nil
}

// InitService initializes the service layer with the store instance.
func (h *bizHooks) InitService(store StoreFactory) (ServiceFactory, error) {
	return NewServiceFactory(store), nil
}

// RegisteredMiddlewares returns the middleware registry for the service factory.
func (h *bizHooks) RegisteredMiddlewares() (grpcmwv1.Middlewares[ServiceFactory], error) {
	return GetMiddlewares(), nil
}

// InitEndpoints initializes the endpoint layer from the middleware-wrapped service factory.
func (h *bizHooks) InitEndpoints(svc ServiceFactory) (DemoEndpoints, error) {
	return NewDemoEndpoints(svc.Demo()), nil
}

// InitTransport initializes the transport layer; chunkSize is injected by the
// generic server from GRPCServing.ChunkSize.
func (h *bizHooks) InitTransport(eps DemoEndpoints, chunkSize int) (Factory, error) {
	return NewFactory(NewDemoHandlers(eps, chunkSize)), nil
}

// ServiceRegistrations declares the services to register from the initialized
// transport factory, delegating to the pure declaration in register.go.
func (h *bizHooks) ServiceRegistrations(txp Factory) ([]svcregisterv1.ServiceRegistration, error) {
	return NewServiceRegistrations(txp), nil
}

// Startup starts business resources before the server begins listening.
func (h *bizHooks) Startup() error {
	logV1.Debug("grpcserver example business resources started")
	return nil
}

// Shutdown cleans up business resources during graceful shutdown.
func (h *bizHooks) Shutdown() error {
	logV1.Debug("grpcserver example business resources stopped")
	return nil
}
