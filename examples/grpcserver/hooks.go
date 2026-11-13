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
type bizHooks struct {
	chunkSize int
}

var _ genericserver.GRPCServerBizHooks[ServiceFactory, StoreFactory] = (*bizHooks)(nil)

// NewBizHooks creates the business hooks. chunkSize is injected from the generic
// server config because the ServiceRegisters hook only receives the service factory.
func NewBizHooks(chunkSize int) *bizHooks {
	return &bizHooks{chunkSize: chunkSize}
}

func (h *bizHooks) InitStore() (StoreFactory, error) {
	return NewStoreFactory(), nil
}

func (h *bizHooks) InitService(store StoreFactory) (ServiceFactory, error) {
	return NewServiceFactory(store), nil
}

func (h *bizHooks) RegisteredMiddlewares() (grpcmwv1.Middlewares[ServiceFactory], error) {
	return GetMiddlewares(), nil
}

// ServiceRegisters builds the gRPC service register map (counterpart of HTTP InitController).
func (h *bizHooks) ServiceRegisters(svc ServiceFactory) (map[string]svcregisterv1.ServiceRegister, error) {
	eps := NewDemoEndpoints(svc.Demo())
	hs := NewDemoHandlers(eps, h.chunkSize)
	txp := New(hs)
	return NewServiceRegisterGenerator(txp).Generate(), nil
}

func (h *bizHooks) Startup() error {
	logV1.Debug("grpcserver example business resources started")
	return nil
}

func (h *bizHooks) Shutdown() error {
	logV1.Debug("grpcserver example business resources stopped")
	return nil
}
