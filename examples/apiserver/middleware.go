// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"context"
	"fmt"

	httpmwv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/http/middleware/v1"
	methodlog "github.com/ClessLi/component-base/pkg/methodlog/v1"
)

// GetMiddlewares returns the named middleware registry for the service factory.
// The generic server installs the middlewares listed in APIServerConfig.Middlewares
// by looking them up here, wrapping the service factory in order.
func GetMiddlewares() httpmwv1.Middlewares[ServiceFactory] {
	return defaultMiddlewares
}

var defaultMiddlewares = httpmwv1.Middlewares[ServiceFactory]{
	"logging": loggingMiddleware,
}

// loggingMiddleware wraps a ServiceFactory with structured method logging backed
// by pkg/methodlog/v1: automatic info/error/panic leveling, elapsed time, HTTP
// client-IP field extraction and panic recovery.
func loggingMiddleware(next ServiceFactory) ServiceFactory {
	return &loggingService{next: next}
}

type loggingService struct {
	next ServiceFactory
}

func (l *loggingService) Users() UserService {
	return &loggingUserService{next: l.next.Users()}
}

type loggingUserService struct {
	next UserService
}

func (l *loggingUserService) Get(ctx context.Context, id string) (*User, error) {
	var u *User
	err := methodlog.New(ctx, l.next.Get, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
		AddInfos(func() []interface{} {
			return []interface{}{"id", id}
		}).
		Result(func() string {
			if u != nil {
				return fmt.Sprintf("user %s (%s)", u.Name, u.ID)
			}
			return "not found"
		}).
		Action(func(m func(context.Context, string) (*User, error)) error {
			var e error
			u, e = m(ctx, id)
			return e
		}).
		Do()
	return u, err
}

func (l *loggingUserService) List(ctx context.Context) ([]*User, error) {
	var users []*User
	err := methodlog.New(ctx, l.next.List, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
		Result(func() string {
			return fmt.Sprintf("%d users", len(users))
		}).
		Action(func(m func(context.Context) ([]*User, error)) error {
			var e error
			users, e = m(ctx)
			return e
		}).
		Do()
	return users, err
}

func (l *loggingUserService) Create(ctx context.Context, name string) (*User, error) {
	var u *User
	err := methodlog.New(ctx, l.next.Create, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
		AddInfos(func() []interface{} {
			return []interface{}{"name", name}
		}).
		Result(func() string {
			if u != nil {
				return fmt.Sprintf("created user %s (%s)", u.Name, u.ID)
			}
			return "created"
		}).
		Action(func(m func(context.Context, string) (*User, error)) error {
			var e error
			u, e = m(ctx, name)
			return e
		}).
		Do()
	return u, err
}
