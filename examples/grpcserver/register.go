// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	svcregisterv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/service_register/v1"
)

// grpcInstancePrefix is the RA service instance prefix, mirroring bifrost.
// Registration names follow "<prefix>.<proto package>.<Service>" and double as
// the gRPC health-check service keys.
const grpcInstancePrefix = "com.github.ClessLi.api"

// NewServiceRegistrations is the default registration scheme. It carries only
// business decisions (service name, implementation, pb register func); the
// registration control flow (health status transitions, invocation order) is owned
// by the framework service_register package.
func NewServiceRegistrations(txp Factory) []svcregisterv1.ServiceRegistration {
	return []svcregisterv1.ServiceRegistration{
		svcregisterv1.NewServiceRegistration(
			grpcInstancePrefix+".grpcserverpb.Demo",
			txp.Demo(),
			pbv1.RegisterDemoServer,
		),
	}
}
