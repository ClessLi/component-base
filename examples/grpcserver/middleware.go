// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"context"
	"fmt"

	grpcmwv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/middleware/v1"
	methodlog "github.com/ClessLi/component-base/pkg/methodlog/v1"
)

// GetMiddlewares returns the middleware registry for the service factory.
func GetMiddlewares() grpcmwv1.Middlewares[ServiceFactory] {
	return defaultMiddlewares
}

var defaultMiddlewares = grpcmwv1.Middlewares[ServiceFactory]{
	"logging": loggingMiddleware,
}

// loggingMiddleware wraps the service factory with structured method logging
// backed by pkg/methodlog/v1: automatic info/error/panic leveling, elapsed time,
// gRPC peer field extraction and panic recovery.
func loggingMiddleware(next ServiceFactory) ServiceFactory {
	return &loggingServiceFactory{next: next}
}

type loggingServiceFactory struct {
	next ServiceFactory
}

func (l *loggingServiceFactory) Demo() DemoService {
	return &loggingDemoService{next: l.next.Demo()}
}

type loggingDemoService struct {
	next DemoService
}

func (l *loggingDemoService) Echo(ctx context.Context, message string) (string, error) {
	var resp string
	err := methodlog.New(ctx, l.next.Echo, methodlog.WithFieldExtractor(methodlog.GRPCFieldExtractor)).
		AddInfos(func() []interface{} {
			return []interface{}{"message", message}
		}).
		Result(func() string {
			return resp
		}).
		Action(func(m func(context.Context, string) (string, error)) error {
			var e error
			resp, e = m(ctx, message)
			return e
		}).
		Do()
	return resp, err
}

func (l *loggingDemoService) Download(ctx context.Context, key string) ([]byte, error) {
	var resp []byte
	err := methodlog.New(ctx, l.next.Download, methodlog.WithFieldExtractor(methodlog.GRPCFieldExtractor)).
		AddInfos(func() []interface{} {
			return []interface{}{"key", key}
		}).
		Result(func() string {
			return fmt.Sprintf("%d bytes", len(resp))
		}).
		Action(func(m func(context.Context, string) ([]byte, error)) error {
			var e error
			resp, e = m(ctx, key)
			return e
		}).
		Do()
	return resp, err
}
