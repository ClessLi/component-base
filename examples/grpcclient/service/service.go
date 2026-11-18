// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package service is the service layer of the gRPC client SDK example. It hides
// the SDK's request/endpoint mechanics behind plain business methods, matching
// the HTTP client SDK example's service layout.
package service

import (
	"context"

	epclientv1 "github.com/ClessLi/component-base/examples/grpcclient/endpoint"
	grpcclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1"
)

// Factory is the entry point of the service layer.
type Factory interface {
	Demo() DemoService
}

// Compile-time check that factory satisfies the factory contract.
var _ Factory = (*factory)(nil)

type factory struct {
	ctx context.Context
	eps epclientv1.Factory
}

func (f *factory) Demo() DemoService {
	return newDemoService(f)
}

// New creates a service Factory from the endpoint Factory.
func New(ctx context.Context, eps epclientv1.Factory) Factory {
	return &factory{ctx: ctx, eps: eps}
}

// DemoService defines business operations against the remote Demo service.
type DemoService interface {
	Echo(message string) (string, error)
	Download(key string) ([]byte, error)
}

// Compile-time check that demoService satisfies the service contract.
var _ DemoService = (*demoService)(nil)

type demoService struct {
	ctx context.Context
	eps epclientv1.DemoEndpoints
}

func (s *demoService) Echo(message string) (string, error) {
	return s.eps.Echo()(s.ctx, grpcclientv1.GRPCRequest[string]{Body: message})
}

func (s *demoService) Download(key string) ([]byte, error) {
	return s.eps.Download()(s.ctx, grpcclientv1.GRPCRequest[string]{Body: key})
}

func newDemoService(f *factory) DemoService {
	return &demoService{ctx: f.ctx, eps: f.eps.Demo()}
}
