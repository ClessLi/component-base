// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import "context"

// UserService defines the business operations for the user resource.
type UserService interface {
	Get(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Create(ctx context.Context, name string) (*User, error)
}

// ServiceFactory provides access to all service instances. It is the entry point
// of the service layer, mirroring gjallarhorn's service/v1.Factory.
type ServiceFactory interface {
	Users() UserService
}

// NewServiceFactory creates a ServiceFactory backed by the given store.
func NewServiceFactory(store StoreFactory) ServiceFactory {
	return &serviceFactory{users: NewUserService(store.Users())}
}

type serviceFactory struct {
	users UserService
}

func (f *serviceFactory) Users() UserService {
	return f.users
}

type userService struct {
	store UserStore
}

// NewUserService creates a UserService backed by the given UserStore.
func NewUserService(store UserStore) UserService {
	return &userService{store: store}
}

func (s *userService) Get(ctx context.Context, id string) (*User, error) {
	return s.store.Get(id)
}

func (s *userService) List(ctx context.Context) ([]*User, error) {
	return s.store.List()
}

func (s *userService) Create(ctx context.Context, name string) (*User, error) {
	return s.store.Create(name)
}
