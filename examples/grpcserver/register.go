// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	svcregisterv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/service_register/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// grpcInstancePrefix is the RA service instance prefix, mirroring bifrost.
const grpcInstancePrefix = "com.github.ClessLi.api"

// serviceRegisterGenerator implements svcregisterv1.ServiceRegisterGenerator.
type serviceRegisterGenerator struct {
	instancePrefixName string
	factory            Factory
}

// NewServiceRegisterGenerator creates the register generator from the transport factory.
func NewServiceRegisterGenerator(factory Factory) svcregisterv1.ServiceRegisterGenerator {
	return &serviceRegisterGenerator{instancePrefixName: grpcInstancePrefix, factory: factory}
}

// Generate builds the service register map consumed by the generic server's
// GRPCServerBizHooks.ServiceRegisters hook.
func (g *serviceRegisterGenerator) Generate() map[string]svcregisterv1.ServiceRegister {
	name := g.instancePrefixName + ".grpcserverpb.Demo"
	return map[string]svcregisterv1.ServiceRegister{
		name: func(server *grpc.Server, healthzSvr *health.Server) {
			pbv1.RegisterDemoServer(server, g.factory.Demo())
			// 通用服务端 ping 要求注册服务为 SERVING，注册后置位以通过启动健康检查。
			if healthzSvr != nil {
				healthzSvr.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
			}
		},
	}
}
