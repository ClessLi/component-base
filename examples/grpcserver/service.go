// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import "context"

// DemoService defines the business operations for the demo resource.
type DemoService interface {
	Echo(ctx context.Context, message string) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
}

// ServiceFactory provides access to all service instances. It is the entry point
// of the service layer, mirroring bifrost's service/v1.ServiceFactory.
type ServiceFactory interface {
	Demo() DemoService
}

// NewServiceFactory creates a ServiceFactory backed by the given store.
func NewServiceFactory(store StoreFactory) ServiceFactory {
	return &serviceFactory{store: store}
}

type serviceFactory struct {
	store StoreFactory
}

// Compile-time check that serviceFactory satisfies the factory contract.
var _ ServiceFactory = (*serviceFactory)(nil)

func (f *serviceFactory) Demo() DemoService {
	return NewDemoService(f.store.Demo())
}

type demoService struct {
	store DemoStore
}

// Compile-time check that demoService satisfies the service contract.
var _ DemoService = (*demoService)(nil)

// NewDemoService creates a DemoService backed by the given store.
func NewDemoService(store DemoStore) DemoService {
	return &demoService{store: store}
}

func (s *demoService) Echo(ctx context.Context, message string) (string, error) {
	return "echo: " + message, nil
}

func (s *demoService) Download(ctx context.Context, key string) ([]byte, error) {
	return s.store.Get(key)
}
