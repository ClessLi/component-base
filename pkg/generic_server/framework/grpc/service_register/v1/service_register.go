package v1

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ServiceRegister is the low-level registration contract consumed by the generic
// server: it mounts one business service onto a grpc.Server and manages the service
// health status. Business code should not build it by hand; declare services via
// ServiceRegistration and let GenerateServiceRegisters apply the control flow.
type ServiceRegister func(server *grpc.Server, healthzSvr *health.Server)

// ServiceRegistration declares one business service to register. It carries only
// business decisions (which service, which implementation, which pb register func);
// the registration control flow (health status transitions, invocation order) is
// owned by the framework (GenerateServiceRegisters).
type ServiceRegistration struct {
	// Name is the fully-qualified service name, used as the health status key
	// and the RA registration name.
	Name string

	// Register mounts the service implementation onto the grpc.Server
	// (e.g. pbv1.RegisterDemoServer). It must not touch the health server.
	Register func(server *grpc.Server)
}

// NewServiceRegistration builds a ServiceRegistration from a pb-generated register
// func and its implementation, so business hooks never write registration closures:
//
//	svcregisterv1.NewServiceRegistration(name, factory.Demo(), pbv1.RegisterDemoServer)
func NewServiceRegistration[T any](name string, impl T, register func(server *grpc.Server, srv T)) ServiceRegistration {
	return ServiceRegistration{
		Name: name,
		Register: func(server *grpc.Server) {
			register(server, impl)
		},
	}
}

// GenerateServiceRegisters turns business declarations into the register map
// consumed by the generic server, applying the unified registration control flow:
// mark NOT_SERVING, mount the implementation, then mark SERVING so the generic
// server's startup health check passes. Declarations with an empty name or a nil
// Register func are skipped.
func GenerateServiceRegisters(registrations ...ServiceRegistration) map[string]ServiceRegister {
	registers := make(map[string]ServiceRegister, len(registrations))
	for _, registration := range registrations {
		if registration.Name == "" || registration.Register == nil {
			continue
		}
		registers[registration.Name] = func(server *grpc.Server, healthzSvr *health.Server) {
			if healthzSvr != nil {
				healthzSvr.SetServingStatus(registration.Name, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			}

			registration.Register(server)

			// The generic server pings registered services until SERVING, so the status
			// is set only after the implementation is mounted.
			if healthzSvr != nil {
				healthzSvr.SetServingStatus(registration.Name, grpc_health_v1.HealthCheckResponse_SERVING)
			}
		}
	}
	return registers
}
