// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package service is the service layer of the HTTP client SDK example. It hides
// the SDK's request/endpoint mechanics behind plain business methods, matching
// the eir apiserver client's service layout.
package service

import (
	"context"

	apiv1 "github.com/ClessLi/component-base/examples/apiserver/api/v1"
	epclientv1 "github.com/ClessLi/component-base/examples/httpclient/endpoint"
	httpclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/http/v1"
)

// Factory is the entry point of the service layer.
type Factory interface {
	Users() UserService
}

// Compile-time check that factory satisfies the factory contract.
var _ Factory = (*factory)(nil)

type factory struct {
	ctx context.Context
	eps epclientv1.Factory
}

func (f *factory) Users() UserService {
	return newUserService(f)
}

// NewService creates a service Factory from the endpoint Factory.
func NewService(ctx context.Context, eps epclientv1.Factory) Factory {
	return &factory{ctx: ctx, eps: eps}
}

// UserService defines business operations against the remote user resource.
type UserService interface {
	Create(req *apiv1.CreateUserRequest) (*apiv1.User, error)
	Get(id string) (*apiv1.User, error)
	List() ([]*apiv1.User, error)
}

// Compile-time check that userService satisfies the service contract.
var _ UserService = (*userService)(nil)

type userService struct {
	ctx context.Context
	eps epclientv1.UserEndpoints
}

func (s *userService) Create(req *apiv1.CreateUserRequest) (*apiv1.User, error) {
	return s.eps.Create()(s.ctx, httpclientv1.HTTPRequest[*apiv1.CreateUserRequest]{
		Body: req,
	})
}

func (s *userService) Get(id string) (*apiv1.User, error) {
	return s.eps.Get()(s.ctx, httpclientv1.HTTPRequest[httpclientv1.NilBody]{
		PathVars: map[string]string{"id": id},
	})
}

func (s *userService) List() ([]*apiv1.User, error) {
	return s.eps.List()(s.ctx, httpclientv1.HTTPRequest[httpclientv1.NilBody]{})
}

func newUserService(f *factory) UserService {
	return &userService{ctx: f.ctx, eps: f.eps.Users()}
}
