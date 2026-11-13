// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"context"

	epv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/endpoint/v1"
)

// DemoEndpoints wraps the service methods into generic go-kit endpoints.
// It is the endpoint layer, mirroring bifrost's endpoint/v1.
type DemoEndpoints interface {
	Echo() epv1.Endpoint[string, string]
	Download() epv1.Endpoint[string, []byte]
}

type demoEndpoints struct {
	svc DemoService
}

// NewDemoEndpoints creates the endpoint set for the demo service.
func NewDemoEndpoints(svc DemoService) DemoEndpoints {
	return &demoEndpoints{svc: svc}
}

func (e *demoEndpoints) Echo() epv1.Endpoint[string, string] {
	return func(ctx context.Context, message string) (string, error) {
		return e.svc.Echo(ctx, message)
	}
}

func (e *demoEndpoints) Download() epv1.Endpoint[string, []byte] {
	return func(ctx context.Context, key string) ([]byte, error) {
		return e.svc.Download(ctx, key)
	}
}
